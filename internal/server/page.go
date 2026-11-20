package server

import (
	"net/http"

	"github.com/matteocontrini/notedimatteo/internal/db"
)

const siteTitle = "Note di Matteo"

// Meta holds a page's title and social metadata. Empty fields use site defaults.
type Meta struct {
	Title       string
	Description string
	OGTitle     string
	OGType      string
}

// Page is the data shared by every page: metadata, header, and sidebar.
type Page struct {
	Meta             Meta
	Canonical        string
	SiteHost         string
	SearchQuery      string
	IsLoggedIn       bool
	UnpublishedCount int
	Categories       []categoryLink
	Archive          []archiveYear
	Tags             []db.TagCount
}

type categoryLink struct {
	Label   string
	Count   int
	URL     string
	Current bool
}

type archiveYear struct {
	Year   int
	Months []db.MonthCount
}

func (s *Server) newPage(r *http.Request, meta Meta) (Page, error) {
	ctx := r.Context()

	categories, err := s.store.Categories(ctx)
	if err != nil {
		return Page{}, err
	}
	archive, err := s.store.Archive(ctx)
	if err != nil {
		return Page{}, err
	}
	tags, err := s.store.Tags(ctx)
	if err != nil {
		return Page{}, err
	}

	if meta.Title == "" {
		meta.Title = siteTitle
	}
	if meta.Description == "" {
		meta.Description = siteTitle
	}
	if meta.OGTitle == "" {
		meta.OGTitle = meta.Title
	}
	if meta.OGType == "" {
		meta.OGType = "website"
	}

	return Page{
		Meta:        meta,
		Canonical:   s.siteURL + r.URL.Path,
		SiteHost:    s.siteHost,
		SearchQuery: r.URL.Query().Get("q"),
		Categories:  categoryLinks(categories, r.URL.Path),
		Archive:     archiveYears(archive),
		Tags:        tags,
	}, nil
}

func categoryLinks(categories []db.CategoryCount, currentPath string) []categoryLink {
	links := make([]categoryLink, 0, len(categories))
	for _, c := range categories {
		url := categoryURL(c.Slug)
		links = append(links, categoryLink{
			Label:   categoryLabel(c.Slug),
			Count:   c.Count,
			URL:     url,
			Current: url == currentPath,
		})
	}
	return links
}

// archiveYears groups months, already sorted newest first, by year.
func archiveYears(months []db.MonthCount) []archiveYear {
	var years []archiveYear
	for _, m := range months {
		if len(years) == 0 || years[len(years)-1].Year != m.Year {
			years = append(years, archiveYear{Year: m.Year})
		}
		last := &years[len(years)-1]
		last.Months = append(last.Months, m)
	}
	return years
}

type pagination struct {
	Current  int
	Total    int
	Previous string
	Next     string
}

func newPagination(path string, current, total int) pagination {
	p := pagination{Current: current, Total: total}
	if current > 1 {
		p.Previous = pageURL(path, current-1)
	}
	if current < total {
		p.Next = pageURL(path, current+1)
	}
	return p
}
