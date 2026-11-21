package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/matteocontrini/notedimatteo/internal/db"
)

// postSizeLimit is the largest save or preview request the server reads.
const postSizeLimit = 1 << 20

// editorPost is a post as the editor receives it in JSON.
type editorPost struct {
	Number      int        `json:"number"`
	Category    string     `json:"category"`
	Title       string     `json:"title"`
	Slug        string     `json:"slug"`
	HideTitle   bool       `json:"hideTitle"`
	Body        string     `json:"body"`
	Tags        []string   `json:"tags"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   *time.Time `json:"updatedAt"`
	PublishedAt *time.Time `json:"publishedAt"`
}

func newEditorPost(post db.EditablePost) editorPost {
	tags := post.Tags
	if tags == nil {
		tags = []string{}
	}
	return editorPost{
		Number:      post.Number,
		Category:    post.Category,
		Title:       post.Title,
		Slug:        post.Slug,
		HideTitle:   post.HideTitle,
		Body:        post.Body,
		Tags:        tags,
		CreatedAt:   post.CreatedAt,
		UpdatedAt:   post.UpdatedAt,
		PublishedAt: post.PublishedAt,
	}
}

// saveResponse is the save endpoint's JSON: the saved post, or the errors of
// the fields that failed validation, by field name.
type saveResponse struct {
	Post   *editorPost       `json:"post,omitempty"`
	Errors map[string]string `json:"errors,omitempty"`
}

var intents = map[string]db.Intent{
	"save":      db.Save,
	"publish":   db.Publish,
	"unpublish": db.Unpublish,
}

// savePost saves the editor's form, and publishes or unpublishes the post if
// the intent field asks to. It answers 400 with field errors if the form is
// invalid.
func (s *Server) savePost(w http.ResponseWriter, r *http.Request) error {
	number, ok := postNumber(r.PathValue("number"))
	if !ok {
		return errNotFound
	}
	if err := parseForm(w, r, postSizeLimit); err != nil {
		return err
	}

	edit, fieldErrors, err := parsePostForm(r.PostForm)
	if err != nil {
		return err
	}
	if len(fieldErrors) > 0 {
		return writeJSON(w, http.StatusBadRequest, saveResponse{Errors: fieldErrors})
	}

	if edit.HTML, err = s.markdown.Render(edit.Body); err != nil {
		return fmt.Errorf("render post %d: %w", number, err)
	}

	post, err := s.store.SavePost(r.Context(), number, edit)
	if errors.Is(err, db.ErrNumberTaken) {
		fieldErrors = map[string]string{"number": fmt.Sprintf("Post #%d already exists", edit.Number)}
		return writeJSON(w, http.StatusBadRequest, saveResponse{Errors: fieldErrors})
	}
	if err != nil {
		return err
	}
	if post == nil {
		return errNotFound
	}

	saved := newEditorPost(*post)
	return writeJSON(w, http.StatusOK, saveResponse{Post: &saved})
}

// parsePostForm reads the editor's fields and checks them like the SvelteKit
// app's schema. It returns the errors by field name. An unknown intent is
// errBadRequest, because the editor only sends known ones.
func parsePostForm(form url.Values) (db.PostEdit, map[string]string, error) {
	intent, ok := intents[form.Get("intent")]
	if !ok {
		return db.PostEdit{}, nil, errBadRequest
	}

	edit := db.PostEdit{
		Category:  strings.TrimSpace(form.Get("category")),
		Title:     strings.TrimSpace(form.Get("title")),
		Slug:      strings.TrimSpace(form.Get("slug")),
		HideTitle: form.Get("hideTitle") == "true" || form.Get("hideTitle") == "on",
		Body:      form.Get("body"),
		Tags:      parseTags(form.Get("tags")),
		Intent:    intent,
	}
	errs := map[string]string{}

	number, err := strconv.Atoi(strings.TrimSpace(form.Get("number")))
	switch {
	case err != nil:
		errs["number"] = "Number must be a whole number"
	case number < 1:
		errs["number"] = "Number must be at least 1"
	}
	edit.Number = number

	if edit.Category == "" {
		errs["category"] = "Category is required"
	}
	if edit.Body == "" {
		errs["body"] = "Body is required"
	}

	return edit, errs, nil
}

// parseTags splits comma-separated tags, dropping empty and repeated ones.
func parseTags(value string) []string {
	var tags []string
	for tag := range strings.SplitSeq(value, ",") {
		tag = strings.TrimSpace(tag)
		if tag != "" && !slices.Contains(tags, tag) {
			tags = append(tags, tag)
		}
	}
	return tags
}

// preview renders a Markdown body sent as {"body": "..."} and answers
// {"html": "..."}.
func (s *Server) preview(w http.ResponseWriter, r *http.Request) error {
	r.Body = http.MaxBytesReader(w, r.Body, postSizeLimit)
	var payload struct {
		Body *string `json:"body"`
	}
	err := json.NewDecoder(r.Body).Decode(&payload)
	if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
		return errTooLarge
	}
	if err != nil || payload.Body == nil {
		return errBadRequest
	}

	html, err := s.markdown.Render(*payload.Body)
	if err != nil {
		return fmt.Errorf("render preview: %w", err)
	}
	return writeJSON(w, http.StatusOK, map[string]string{"html": html})
}
