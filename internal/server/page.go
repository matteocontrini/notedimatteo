package server

import (
	"net/http"

	"github.com/matteocontrini/notedimatteo/internal/db"
	"github.com/matteocontrini/notedimatteo/internal/urls"
	"github.com/matteocontrini/notedimatteo/internal/views"
)

const siteTitle = "Note di Matteo"

// newPage loads the data every page needs: metadata, the header's categories,
// and, for logged-in users, the number of drafts. It fills empty meta fields
// with site defaults.
func (s *Server) newPage(r *http.Request, meta views.Meta) (views.Page, error) {
	categories, err := s.store.Categories(r.Context())
	if err != nil {
		return views.Page{}, err
	}

	loggedIn := isLoggedIn(r)
	drafts := 0
	if loggedIn {
		if drafts, err = s.store.CountDrafts(r.Context()); err != nil {
			return views.Page{}, err
		}
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

	return views.Page{
		Meta:        meta,
		Canonical:   s.siteURL + r.URL.EscapedPath(),
		SiteHost:    s.siteHost,
		SearchQuery: r.URL.Query().Get("q"),
		Stylesheet:  s.stylesheet,
		Script:      s.script,
		Categories:  categoryLinks(categories, r.URL.EscapedPath()),

		IsLoggedIn:       loggedIn,
		UnpublishedCount: drafts,
	}, nil
}

// newSidebarPage is newPage plus the sidebar's archive and tags.
func (s *Server) newSidebarPage(r *http.Request, meta views.Meta) (views.Page, error) {
	page, err := s.newPage(r, meta)
	if err != nil {
		return views.Page{}, err
	}

	archive, err := s.store.Archive(r.Context(), views.Rome)
	if err != nil {
		return views.Page{}, err
	}
	tags, err := s.store.Tags(r.Context())
	if err != nil {
		return views.Page{}, err
	}

	page.Archive = archiveYears(archive)
	page.Tags = tags
	return page, nil
}

func categoryLinks(categories []db.CategoryCount, currentPath string) []views.CategoryLink {
	links := make([]views.CategoryLink, 0, len(categories))
	for _, c := range categories {
		url := urls.Category(c.Slug)
		links = append(links, views.CategoryLink{
			Label:   views.CategoryLabel(c.Slug),
			Count:   c.Count,
			URL:     url,
			Current: url == currentPath,
		})
	}
	return links
}

// archiveYears groups months, already sorted newest first, by year.
func archiveYears(months []db.MonthCount) []views.ArchiveYear {
	var years []views.ArchiveYear
	for _, m := range months {
		if len(years) == 0 || years[len(years)-1].Year != m.Year {
			years = append(years, views.ArchiveYear{Year: m.Year})
		}
		last := &years[len(years)-1]
		last.Months = append(last.Months, m)
	}
	return years
}

func newPagination(path string, current, total int) views.Pagination {
	p := views.Pagination{Current: current, Total: total}
	if current > 1 {
		p.Previous = urls.Page(path, current-1)
	}
	if current < total {
		p.Next = urls.Page(path, current+1)
	}
	return p
}
