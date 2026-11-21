package db

import (
	"context"
	"errors"
	"slices"
	"testing"
)

func TestSavePost(t *testing.T) {
	db := openTestTx(t)
	ctx := context.Background()

	draft, err := db.CreateDraft(ctx)
	if err != nil {
		t.Fatal(err)
	}

	edit := PostEdit{
		Number:   draft,
		Category: "til",
		Title:    "Title",
		Body:     "Body",
		Tags:     []string{"zz-new-tag", "go"},
		HTML:     "<p>Body</p>\n",
		Intent:   Save,
	}
	post, err := db.SavePost(ctx, draft, edit)
	if err != nil {
		t.Fatal(err)
	}
	if post.Title != "Title" || post.Category != "til" || post.Slug != "" || post.UpdatedAt == nil || post.PublishedAt != nil {
		t.Errorf("saved %+v", post)
	}
	// Postgres can return the tags in any order
	if tags := slices.Sorted(slices.Values(post.Tags)); !slices.Equal(tags, []string{"go", "zz-new-tag"}) {
		t.Errorf("tags = %v", post.Tags)
	}

	// Each new body gets a new revision: the draft's empty body was revision 1.
	// Publishing shows the post, with the HTML cached for the new revision.
	edit.Intent = Publish
	edit.Body, edit.HTML = "New body", "<p>New body</p>\n"
	edit.Tags = []string{"go"}
	if post, err = db.SavePost(ctx, draft, edit); err != nil {
		t.Fatal(err)
	}
	if post.PublishedAt == nil || !slices.Equal(post.Tags, []string{"go"}) {
		t.Errorf("published %+v", post)
	}
	published, err := db.PostByNumber(ctx, draft)
	if err != nil || published == nil {
		t.Fatalf("PostByNumber = %v, %v", published, err)
	}
	if published.BodyRevision != 3 || published.CachedHTML == nil || *published.CachedHTML != edit.HTML {
		t.Errorf("revision %d, cached HTML %v", published.BodyRevision, published.CachedHTML)
	}

	// Saving the same body keeps the revision
	edit.Intent = Unpublish
	if post, err = db.SavePost(ctx, draft, edit); err != nil {
		t.Fatal(err)
	}
	if post.PublishedAt != nil {
		t.Error("post is still published")
	}
	drafts, _ := db.Drafts(ctx)
	if i := slices.IndexFunc(drafts, func(p Post) bool { return p.Number == draft }); i < 0 || drafts[i].BodyRevision != 3 {
		t.Errorf("draft missing or revision changed: %v", i)
	}

	// Renumbering
	edit.Number = draft + 1000
	if post, err = db.SavePost(ctx, draft, edit); err != nil || post.Number != draft+1000 {
		t.Fatalf("renumbered %+v, %v", post, err)
	}
	if old, _ := db.PostForEdit(ctx, draft); old != nil {
		t.Error("the old number still exists")
	}
}

func TestSavePostErrors(t *testing.T) {
	db := openTestTx(t)
	ctx := context.Background()

	first, _ := db.CreateDraft(ctx)
	second, err := db.CreateDraft(ctx)
	if err != nil {
		t.Fatal(err)
	}

	edit := PostEdit{Number: first, Category: "note", Body: "Body"}
	if _, err := db.SavePost(ctx, second, edit); !errors.Is(err, ErrNumberTaken) {
		t.Errorf("taken number: %v", err)
	}
	if post, err := db.SavePost(ctx, -1, edit); post != nil || err != nil {
		t.Errorf("missing post: %v, %v", post, err)
	}
}
