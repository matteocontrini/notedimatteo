// Package views contains the site's templ components and the data they render.
package views

import "github.com/matteocontrini/notedimatteo/internal/db"

// Meta holds a page's title and social metadata.
type Meta struct {
	Title       string
	Description string
	OGTitle     string
	OGType      string
}

// Page is the data shared by every page: metadata, assets, header, and sidebar.
type Page struct {
	Meta        Meta
	Canonical   string
	SiteHost    string
	SearchQuery string
	// Stylesheet and Script are the URLs of the built assets.
	Stylesheet       string
	Script           string
	IsLoggedIn       bool
	UnpublishedCount int
	Categories       []CategoryLink
	Archive          []ArchiveYear
	Tags             []db.TagCount
}

type CategoryLink struct {
	Label   string
	Count   int
	URL     string
	Current bool
}

type ArchiveYear struct {
	Year   int
	Months []db.MonthCount
}

type Pagination struct {
	Current  int
	Total    int
	Previous string
	Next     string
}

// PostItem is a post with its rendered body. HTML must come from the Markdown
// renderer, because it's output without escaping.
type PostItem struct {
	db.Post
	HTML string
}

type PostGroup struct {
	Heading string
	Posts   []PostItem
}

// GroupByDay groups posts, already sorted, by their day in Rome's time zone.
func GroupByDay(posts []PostItem) []PostGroup {
	var groups []PostGroup
	lastKey := ""
	for _, post := range posts {
		key := dateKey(post.Timestamp())
		if len(groups) == 0 || key != lastKey {
			groups = append(groups, PostGroup{Heading: formatDate(post.Timestamp())})
			lastKey = key
		}
		last := &groups[len(groups)-1]
		last.Posts = append(last.Posts, post)
	}
	return groups
}
