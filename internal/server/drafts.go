package server

import (
	"net/http"

	"github.com/matteocontrini/notedimatteo/internal/urls"
	"github.com/matteocontrini/notedimatteo/internal/views"
)

// unpublished lists the drafts, with links to edit or delete them.
func (s *Server) unpublished(w http.ResponseWriter, r *http.Request) error {
	posts, err := s.store.Drafts(r.Context())
	if err != nil {
		return err
	}
	items, err := s.postItems(r.Context(), posts)
	if err != nil {
		return err
	}
	page, err := s.newSidebarPage(r, views.Meta{})
	if err != nil {
		return err
	}
	return s.render(w, r, http.StatusOK, views.Unpublished(page, items))
}

// newDraft creates an empty draft and redirects to its editor. SvelteKit used
// a GET link; this is a POST, so other sites can't create drafts.
func (s *Server) newDraft(w http.ResponseWriter, r *http.Request) error {
	number, err := s.store.CreateDraft(r.Context())
	if err != nil {
		return err
	}
	redirect(w, urls.Edit(number), http.StatusSeeOther)
	return nil
}

// deleteDraft deletes an unpublished post. Published posts get a 404.
func (s *Server) deleteDraft(w http.ResponseWriter, r *http.Request) error {
	number, ok := postNumber(r.PathValue("number"))
	if !ok {
		return errNotFound
	}
	deleted, err := s.store.DeleteDraft(r.Context(), number)
	if err != nil {
		return err
	}
	if !deleted {
		return errNotFound
	}
	redirect(w, "/unpublished", http.StatusSeeOther)
	return nil
}
