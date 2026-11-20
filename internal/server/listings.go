package server

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/matteocontrini/notedimatteo/internal/db"
	"github.com/matteocontrini/notedimatteo/internal/views"
)

const postsPerPage = 20

// pageNumber reads ?page=, which must be a positive integer if present.
func pageNumber(r *http.Request) (int, error) {
	value := r.URL.Query().Get("page")
	if value == "" {
		return 1, nil
	}
	page, err := strconv.Atoi(value)
	if err != nil || page < 1 || page > math.MaxInt32/postsPerPage {
		return 0, errBadRequest
	}
	return page, nil
}

// paginatedPosts loads one page of the posts matching filter, of count in total.
func (s *Server) paginatedPosts(r *http.Request, filter db.PostFilter, count int) ([]views.PostItem, views.Pagination, error) {
	current, err := pageNumber(r)
	if err != nil {
		return nil, views.Pagination{}, err
	}

	posts, err := s.store.Posts(r.Context(), filter, postsPerPage, (current-1)*postsPerPage)
	if err != nil {
		return nil, views.Pagination{}, err
	}
	items, err := s.postItems(r.Context(), posts)
	if err != nil {
		return nil, views.Pagination{}, err
	}

	totalPages := (count + postsPerPage - 1) / postsPerPage
	return items, newPagination(r.URL.Path, current, totalPages), nil
}

func (s *Server) home(w http.ResponseWriter, r *http.Request) error {
	page, err := s.newSidebarPage(r, views.Meta{})
	if err != nil {
		return err
	}

	count, err := s.store.CountPosts(r.Context(), db.PostFilter{})
	if err != nil {
		return err
	}
	items, pagination, err := s.paginatedPosts(r, db.PostFilter{}, count)
	if err != nil {
		return err
	}

	return s.render(w, r, http.StatusOK, views.Home(page, views.GroupByDay(items), pagination))
}

func (s *Server) category(w http.ResponseWriter, r *http.Request, slug string) error {
	if !views.IsCategory(slug) {
		return errNotFound
	}

	label := views.CategoryLabel(slug)
	page, err := s.newSidebarPage(r, views.Meta{Title: label + " in Note di Matteo"})
	if err != nil {
		return err
	}

	filter := db.PostFilter{Category: slug}
	count, err := s.store.CountPosts(r.Context(), filter)
	if err != nil {
		return err
	}
	items, pagination, err := s.paginatedPosts(r, filter, count)
	if err != nil {
		return err
	}

	return s.render(w, r, http.StatusOK,
		views.Listing(page, label, "Nessun post pubblicato in questa categoria.", items, pagination))
}

func (s *Server) tag(w http.ResponseWriter, r *http.Request) error {
	tag := strings.TrimSpace(r.PathValue("tag"))
	if tag == "" {
		return errNotFound
	}

	filter := db.PostFilter{Tag: tag}
	count, err := s.store.CountPosts(r.Context(), filter)
	if err != nil {
		return err
	}
	if count == 0 {
		return errNotFound
	}

	page, err := s.newSidebarPage(r, views.Meta{Title: "#" + tag})
	if err != nil {
		return err
	}
	items, pagination, err := s.paginatedPosts(r, filter, count)
	if err != nil {
		return err
	}

	return s.render(w, r, http.StatusOK,
		views.Listing(page, "#"+tag, "Nessun post con questo tag.", items, pagination))
}

// month serves /archivio/{year}/{month}, with the month written with two
// digits; /archivio/2026/4 redirects to /archivio/2026/04.
//
// TODO: after the migration, select the month in Europe/Rome rather than UTC.
func (s *Server) month(w http.ResponseWriter, r *http.Request) error {
	yearText, monthText := r.PathValue("year"), r.PathValue("month")
	year, yearErr := strconv.Atoi(yearText)
	month, monthErr := strconv.Atoi(monthText)
	if yearErr != nil || monthErr != nil || year < 1 || year > 9999 || month < 1 || month > 12 {
		return errNotFound
	}

	if canonical := fmt.Sprintf("%d/%02d", year, month); yearText+"/"+monthText != canonical {
		target := "/archivio/" + canonical
		if r.URL.RawQuery != "" {
			target += "?" + r.URL.RawQuery
		}
		redirect(w, target, http.StatusTemporaryRedirect)
		return nil
	}

	from := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	posts, err := s.store.Posts(r.Context(), db.PostFilter{From: from, Until: from.AddDate(0, 1, 0)}, 0, 0)
	if err != nil {
		return err
	}
	if len(posts) == 0 {
		return errNotFound
	}

	items, err := s.postItems(r.Context(), posts)
	if err != nil {
		return err
	}
	page, err := s.newSidebarPage(r, views.Meta{Title: "Archivio " + views.MonthName(month) + " " + yearText})
	if err != nil {
		return err
	}

	heading := views.CapitalizedMonthName(month) + " " + yearText
	return s.render(w, r, http.StatusOK,
		views.Listing(page, heading, "Nessun post pubblicato in questo mese.", items, views.Pagination{}))
}
