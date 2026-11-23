package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ncruces/go-sqlite3"

	"github.com/matteocontrini/notedimatteo/internal/markdown"
)

// EditablePost is a post as the editor shows it, published or not.
type EditablePost struct {
	ID          int
	Number      int
	Category    string
	Title       string
	Slug        string
	HideTitle   bool
	Body        string
	CreatedAt   time.Time
	UpdatedAt   *time.Time
	PublishedAt *time.Time
	// Tags are in the order they were typed.
	Tags []string
}

func newEditablePost(row PostForEditRow) (*EditablePost, error) {
	tags, err := decodeTags(row.Tags)
	if err != nil {
		return nil, err
	}
	return &EditablePost{
		ID:          row.ID,
		Number:      row.Number,
		Category:    row.Category,
		Title:       row.Title,
		Slug:        row.Slug,
		HideTitle:   row.HideTitle,
		Body:        row.Body,
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   timePointer(row.UpdatedAt),
		PublishedAt: timePointer(row.PublishedAt),
		Tags:        tags,
	}, nil
}

// Intent says what saving does to a post's publication.
type Intent int

const (
	// Save keeps the post published or unpublished.
	Save Intent = iota
	// Publish sets the publication time to now, even if the post was published.
	Publish
	// Unpublish turns the post back into a draft.
	Unpublish
)

// PostEdit holds the editor's fields.
type PostEdit struct {
	Number    int
	Category  string
	Title     string
	Slug      string
	HideTitle bool
	Body      string
	Tags      []string
	// HTML is Body rendered by the current Markdown renderer.
	HTML   string
	Intent Intent
}

// ErrNumberTaken means another post already has the number.
var ErrNumberTaken = errors.New("post number already taken")

// PostForEdit returns a post, published or not, or nil if there's none.
func (db *DB) PostForEdit(ctx context.Context, number int) (*EditablePost, error) {
	row, err := db.queries.PostForEdit(ctx, number)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find post: %w", err)
	}
	return newEditablePost(row)
}

// SavePost saves the editor's fields to the post with number, and returns the
// saved post, or nil if there's no such post. One transaction updates the
// post, its cached HTML, and its tags. The post's number can change.
func (db *DB) SavePost(ctx context.Context, number int, edit PostEdit) (*EditablePost, error) {
	tx, err := db.sql.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin save: %w", err)
	}
	defer tx.Rollback()

	saved, err := savePost(ctx, db.queries.WithTx(tx), number, edit)
	if saved == nil || err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit save: %w", err)
	}
	return saved, nil
}

// savePost runs SavePost's queries in its transaction.
func savePost(ctx context.Context, q *Queries, number int, edit PostEdit) (*EditablePost, error) {
	post, err := q.PostForSave(ctx, number)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find post: %w", err)
	}

	now := currentTime()
	revision := post.BodyRevision
	if edit.Body != post.Body {
		revision++
	}
	publishedAt := post.PublishedAt
	switch edit.Intent {
	case Publish:
		publishedAt = &now
	case Unpublish:
		publishedAt = nil
	}

	err = q.UpdatePost(ctx, UpdatePostParams{
		ID:           post.ID,
		Number:       edit.Number,
		Category:     edit.Category,
		Title:        edit.Title,
		Slug:         edit.Slug,
		HideTitle:    edit.HideTitle,
		Body:         edit.Body,
		BodyRevision: revision,
		UpdatedAt:    &now,
		PublishedAt:  publishedAt,
	})
	// number is the only unique column that the update can change
	if errors.Is(err, sqlite3.CONSTRAINT_UNIQUE) {
		return nil, ErrNumberTaken
	}
	if err != nil {
		return nil, fmt.Errorf("update post: %w", err)
	}

	err = q.SavePostHTML(ctx, SavePostHTMLParams{
		PostID:          post.ID,
		RendererVersion: markdown.Version,
		BodyRevision:    revision,
		HTML:            edit.HTML,
		RenderedAt:      now,
	})
	if err != nil {
		return nil, fmt.Errorf("save post HTML: %w", err)
	}

	if err := q.DeletePostTags(ctx, post.ID); err != nil {
		return nil, fmt.Errorf("delete post tags: %w", err)
	}
	if len(edit.Tags) > 0 {
		names, err := json.Marshal(edit.Tags)
		if err != nil {
			return nil, err
		}
		if err := q.CreateTags(ctx, string(names)); err != nil {
			return nil, fmt.Errorf("create tags: %w", err)
		}
		if err := q.AddPostTags(ctx, AddPostTagsParams{PostID: post.ID, Names: string(names)}); err != nil {
			return nil, fmt.Errorf("add post tags: %w", err)
		}
	}

	row, err := q.PostForEdit(ctx, edit.Number)
	if err != nil {
		return nil, fmt.Errorf("reload post: %w", err)
	}
	return newEditablePost(row)
}
