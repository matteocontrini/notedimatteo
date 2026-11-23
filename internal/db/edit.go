package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/matteocontrini/notedimatteo/internal/markdown"
)

// EditablePost is a post as the editor shows it, published or not.
type EditablePost = PostForEditRow

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
	post, err := db.queries.PostForEdit(ctx, number)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find post: %w", err)
	}
	return &post, nil
}

// SavePost saves the editor's fields to the post with number, and returns the
// saved post, or nil if there's no such post. One transaction updates the
// post, its cached HTML, and its tags. The post's number can change.
func (db *DB) SavePost(ctx context.Context, number int, edit PostEdit) (*EditablePost, error) {
	var saved *EditablePost
	err := pgx.BeginFunc(ctx, db.conn, func(tx pgx.Tx) error {
		q := db.queries.WithTx(tx)

		post, err := q.LockPost(ctx, number)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("lock post: %w", err)
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
			UpdatedAt:    now,
			PublishedAt:  publishedAt,
		})
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.ConstraintName == "posts_number_key" {
			return ErrNumberTaken
		}
		if err != nil {
			return fmt.Errorf("update post: %w", err)
		}

		err = q.SavePostHTML(ctx, SavePostHTMLParams{
			PostID:          post.ID,
			RendererVersion: markdown.Version,
			BodyRevision:    revision,
			HTML:            edit.HTML,
			RenderedAt:      now,
		})
		if err != nil {
			return fmt.Errorf("save post HTML: %w", err)
		}

		if err := q.DeletePostTags(ctx, post.ID); err != nil {
			return fmt.Errorf("delete post tags: %w", err)
		}
		if len(edit.Tags) > 0 {
			if err := q.CreateTags(ctx, edit.Tags); err != nil {
				return fmt.Errorf("create tags: %w", err)
			}
			if err := q.AddPostTags(ctx, AddPostTagsParams{PostID: post.ID, Names: edit.Tags}); err != nil {
				return fmt.Errorf("add post tags: %w", err)
			}
		}

		row, err := q.PostForEdit(ctx, edit.Number)
		if err != nil {
			return fmt.Errorf("reload post: %w", err)
		}
		saved = &row
		return nil
	})
	if err != nil {
		return nil, err
	}
	return saved, nil
}
