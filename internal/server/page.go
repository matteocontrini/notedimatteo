package server

import (
	"net/http"

	"github.com/matteocontrini/notedimatteo/internal/db"
	"github.com/matteocontrini/notedimatteo/internal/urls"
	"github.com/matteocontrini/notedimatteo/internal/views"
)

const siteTitle = "Note di Matteo"

// newPage loads the data shared by every page. Empty meta fields use site defaults.
func (s *Server) newPage(r *http.Request, meta views.Meta) (views.Page, error) {
	ctx := r.Context()

	categories, err := s.store.Categories(ctx)
	if err != nil {
		return views.Page{}, err
	}
	archive, err := s.store.Archive(ctx)
	if err != nil {
		return views.Page{}, err
	}
	tags, err := s.store.Tags(ctx)
	if err != nil {
		return views.Page{}, err
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
		Canonical:   s.siteURL + r.URL.Path,
		SiteHost:    s.siteHost,
		SearchQuery: r.URL.Query().Get("q"),
		Stylesheet:  s.stylesheet,
		Script:      s.script,
		Categories:  categoryLinks(categories, r.URL.Path),
		Archive:     archiveYears(archive),
		Tags:        tags,
	}, nil
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
