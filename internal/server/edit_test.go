package server

import (
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/matteocontrini/notedimatteo/internal/db"
)

func validPostForm() url.Values {
	return url.Values{
		"intent":   {"save"},
		"number":   {"5"},
		"category": {"note"},
		"title":    {""},
		"slug":     {""},
		"body":     {"Body"},
		"tags":     {""},
	}
}

func TestParsePostForm(t *testing.T) {
	form := validPostForm()
	form.Set("number", " 7 ")
	form.Set("title", "  Title ")
	form.Set("hideTitle", "on")
	form.Set("tags", "go, , svelte,go ,")
	form.Set("body", "one\r\n\r\ntwo")
	form.Set("intent", "publish")

	edit, errs, err := parsePostForm(form)
	if err != nil || len(errs) != 0 {
		t.Fatalf("valid form: %v, %v", errs, err)
	}
	if edit.Number != 7 || edit.Title != "Title" || !edit.HideTitle || edit.Intent != db.Publish {
		t.Errorf("edit = %+v", edit)
	}
	if !slices.Equal(edit.Tags, []string{"go", "svelte"}) {
		t.Errorf("tags = %q", edit.Tags)
	}
	if edit.Body != "one\n\ntwo" {
		t.Errorf("body = %q", edit.Body)
	}

	tests := []struct {
		field, value string
		errors       map[string]string
	}{
		{"number", "abc", map[string]string{"number": "Number must be a whole number"}},
		{"number", "0", map[string]string{"number": "Number must be at least 1"}},
		{"category", "  ", map[string]string{"category": "Category is required"}},
		{"body", "", map[string]string{"body": "Body is required"}},
		{"hideTitle", "false", nil},
	}
	for _, tt := range tests {
		form := validPostForm()
		form.Set(tt.field, tt.value)
		edit, errs, err := parsePostForm(form)
		if err != nil || len(errs) != len(tt.errors) {
			t.Errorf("%s=%q: errors %v, %v", tt.field, tt.value, errs, err)
		}
		for field, message := range tt.errors {
			if errs[field] != message {
				t.Errorf("%s=%q: %s error = %q", tt.field, tt.value, field, errs[field])
			}
		}
		if tt.field == "hideTitle" && edit.HideTitle {
			t.Error(`hideTitle "false" is true`)
		}
	}

	form = validPostForm()
	form.Set("intent", "delete")
	if _, _, err := parsePostForm(form); !errors.Is(err, errBadRequest) {
		t.Errorf("unknown intent: %v", err)
	}
}

// decodeJSON decodes a response body, failing the test if it isn't JSON.
func decodeJSON[T any](t *testing.T, body string) T {
	t.Helper()

	var v T
	if err := json.Unmarshal([]byte(body), &v); err != nil {
		t.Fatalf("response isn't JSON: %v: %s", err, body)
	}
	return v
}

// multipartRequest posts form as multipart/form-data, like the browser's
// FormData, with the session cookie of loggedInStore.
func multipartRequest(path string, form url.Values) *http.Request {
	var body strings.Builder
	mw := multipart.NewWriter(&body)
	for field, values := range form {
		mw.WriteField(field, values[0])
	}
	mw.Close()

	r := newRequest(http.MethodPost, path, strings.NewReader(body.String()), "token")
	r.Header.Set("Content-Type", mw.FormDataContentType())
	return r
}

func TestSavePostEndpoint(t *testing.T) {
	store := loggedInStore()
	store.post = &db.Post{Number: 5}

	rec := send(t, store, http.MethodPost, "/5/edit", validPostForm(), "token")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	if got := decodeJSON[saveResponse](t, rec.Body.String()); got.Post == nil || got.Post.Number != 5 || got.Post.Tags == nil {
		t.Errorf("response = %s", rec.Body)
	}
	if store.savedEdit.HTML != "<p>Body</p>\n" {
		t.Errorf("saved HTML = %q", store.savedEdit.HTML)
	}

	// The editor sends FormData, which is multipart
	if rec := serve(t, store, multipartRequest("/5/edit", validPostForm())); rec.Code != http.StatusOK {
		t.Errorf("multipart: status = %d: %s", rec.Code, rec.Body)
	}

	invalid := validPostForm()
	invalid.Set("body", "")
	rec = send(t, store, http.MethodPost, "/5/edit", invalid, "token")
	if got := decodeJSON[saveResponse](t, rec.Body.String()); rec.Code != http.StatusBadRequest || got.Errors["body"] == "" {
		t.Errorf("invalid form: %d %s", rec.Code, rec.Body)
	}

	store.savePostErr = db.ErrNumberTaken
	rec = send(t, store, http.MethodPost, "/5/edit", validPostForm(), "token")
	if got := decodeJSON[saveResponse](t, rec.Body.String()); rec.Code != http.StatusBadRequest || got.Errors["number"] != "Post #5 already exists" {
		t.Errorf("taken number: %d %s", rec.Code, rec.Body)
	}
}

func TestEditorEndpointErrors(t *testing.T) {
	tooLarge := validPostForm()
	tooLarge.Set("body", strings.Repeat("a", postSizeLimit))

	tests := []struct {
		name, path string
		form       url.Values
		token      string
		// multipart sends the form as multipart/form-data, which Go parses separately
		multipart bool
		status    int
	}{
		{"logged out", "/5/edit", validPostForm(), "", false, http.StatusUnauthorized},
		{"missing post", "/6/edit", validPostForm(), "token", false, http.StatusNotFound},
		{"too large", "/5/edit", tooLarge, "token", false, http.StatusRequestEntityTooLarge},
		{"too large multipart", "/5/edit", tooLarge, "token", true, http.StatusRequestEntityTooLarge},
		{"preview logged out", "/5/edit/preview", url.Values{}, "", false, http.StatusUnauthorized},
	}
	for _, tt := range tests {
		store := loggedInStore()
		store.post = &db.Post{Number: 5}
		var rec *httptest.ResponseRecorder
		if tt.multipart {
			rec = serve(t, store, multipartRequest(tt.path, tt.form))
		} else {
			rec = send(t, store, http.MethodPost, tt.path, tt.form, tt.token)
		}

		if rec.Code != tt.status {
			t.Errorf("%s: status = %d", tt.name, rec.Code)
		}
		if got := decodeJSON[map[string]string](t, rec.Body.String()); got["error"] != http.StatusText(tt.status) {
			t.Errorf("%s: body = %s", tt.name, rec.Body)
		}
	}
}

func TestPreview(t *testing.T) {
	tests := []struct {
		body   string
		status int
		html   string
	}{
		{`{"body": "~~old~~ *new*"}`, http.StatusOK, "<p><del>old</del> <em>new</em></p>\n"},
		{`{"body": ""}`, http.StatusOK, ""},
		{`{}`, http.StatusBadRequest, ""},
		{`not JSON`, http.StatusBadRequest, ""},
	}
	for _, tt := range tests {
		r := newRequest(http.MethodPost, "/5/edit/preview", strings.NewReader(tt.body), "token")
		r.Header.Set("Content-Type", "application/json")
		rec := serve(t, loggedInStore(), r)

		if rec.Code != tt.status {
			t.Errorf("%s: status = %d", tt.body, rec.Code)
		}
		if got := decodeJSON[map[string]string](t, rec.Body.String()); tt.status == http.StatusOK && got["html"] != tt.html {
			t.Errorf("%s: html = %q", tt.body, got["html"])
		}
	}
}

func TestEditPage(t *testing.T) {
	store := loggedInStore()
	store.tags = []db.TagCount{{Name: "go", Count: 3}}
	store.editPost = &db.EditablePost{
		ID:     7,
		Number: 5,
		Slug:   "a-slug",
		// The page must escape this, or the body would end the script element
		Body: "</script><script>alert(1)</script>",
	}

	rec := serve(t, store, newRequest(http.MethodGet, "/5/edit", nil, "token"))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	page := rec.Body.String()
	start := `<script id="editor-data" type="application/json">`
	i := strings.Index(page, start)
	if i < 0 {
		t.Fatal("the page has no editor data")
	}
	script, _, _ := strings.Cut(page[i+len(start):], "</script>")
	data := decodeJSON[editorData](t, script)

	if data.Post.ID != 7 || data.Post.Body != store.editPost.Body {
		t.Errorf("post = %+v", data.Post)
	}
	want := editorURLs{Post: "/5/a-slug", Edit: "/5/edit", Preview: "/5/edit/preview", Upload: "/5/edit/upload"}
	if data.Post.URLs != want {
		t.Errorf("URLs = %+v", data.Post.URLs)
	}
	if !slices.Equal(data.Tags, []string{"go"}) {
		t.Errorf("tags = %v", data.Tags)
	}
	if len(data.Categories) == 0 || data.Categories[0] != (editorCategory{"note", "Note"}) {
		t.Errorf("categories = %v", data.Categories)
	}
	if data.ImageBaseURL != "https://storage.example.com" {
		t.Errorf("image base URL = %q", data.ImageBaseURL)
	}
	assertContains(t, rec, `<script type="module" src="/assets/editor-789abc.js">`, "<title>Modifica #5</title>")
}

func TestEditPageNotFound(t *testing.T) {
	for _, path := range []string{"/5/edit", "/abc/edit"} {
		rec := serve(t, loggedInStore(), newRequest(http.MethodGet, path, nil, "token"))
		if rec.Code != http.StatusNotFound {
			t.Errorf("%s status = %d", path, rec.Code)
		}
	}
}

// TestEditRouting checks that /{number}/edit reaches the editor, while
// /archivio/edit stays the archive of the tag "edit".
func TestEditRouting(t *testing.T) {
	store := loggedInStore()
	store.editPost = &db.EditablePost{Number: 5}
	if rec := serve(t, store, newRequest(http.MethodGet, "/5/edit", nil, "token")); rec.Code != http.StatusOK {
		t.Errorf("/5/edit status = %d", rec.Code)
	}

	store = &fakeStore{posts: []db.Post{{Number: 1}}, total: 1}
	rec := get(t, store, "/archivio/edit")
	if rec.Code != http.StatusOK || store.lastFilter.Tag != "edit" {
		t.Errorf("/archivio/edit status = %d, filter %+v", rec.Code, store.lastFilter)
	}
}
