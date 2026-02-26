package web

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"blog/internal/config"
	"blog/internal/notes"
	"github.com/a-h/templ"
	"github.com/starfederation/datastar-go/datastar"
)

const (
	pageRouteNotes        = "notes"
	pageRouteNoteBySlug   = "note/[slug]"
	pageRouteAuthorBySlug = "author/[slug]"
	routeParamSlug        = "slug"
)

var slugPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

type Handler struct {
	cfg        config.Config
	service    *notes.Service
	pageRouter *AppRouter
}

func NewHandler(cfg config.Config, service *notes.Service) (*Handler, error) {
	pageRouter, err := NewAppRouter(embeddedAppFS, "app")
	if err != nil {
		return nil, fmt.Errorf("create app router: %w", err)
	}

	return &Handler{cfg: cfg, service: service, pageRouter: pageRouter}, nil
}

func (h *Handler) Register(mux *http.ServeMux) {
	fs := http.FileServer(http.Dir(h.cfg.StaticDir))
	mux.Handle("/static/", withCacheControlPublicHour(http.StripPrefix("/static/", fs)))
	mux.HandleFunc("/", h.handleRoute)
}

func (h *Handler) handleRoute(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		h.handleHome(w, r)
		return
	}

	if r.URL.Path == "/healthz" {
		h.handleHealth(w)
		return
	}

	if r.URL.Path == "/notes/live" {
		h.handleNotesLive(w, r)
		return
	}

	if params, ok := matchPathPattern("/author/[slug]/live", r.URL.Path); ok {
		slug := strings.TrimSpace(params[routeParamSlug])
		if !slugPattern.MatchString(slug) {
			http.NotFound(w, r)
			return
		}
		h.handleAuthorBySlugLive(w, r, slug)
		return
	}

	match, ok := h.pageRouter.Match(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}

	switch match.ID {
	case pageRouteNotes:
		h.handleNotes(w, r)
	case pageRouteNoteBySlug:
		slug, ok := match.Param(routeParamSlug)
		if !ok || !slugPattern.MatchString(slug) {
			http.NotFound(w, r)
			return
		}
		h.handleNoteBySlug(w, r, slug)
	case pageRouteAuthorBySlug:
		slug, ok := match.Param(routeParamSlug)
		if !ok || !slugPattern.MatchString(slug) {
			http.NotFound(w, r)
			return
		}
		h.handleAuthorBySlug(w, r, slug)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) handleHome(w http.ResponseWriter, r *http.Request) {
	setCacheControlPublicHour(w)
	http.Redirect(w, r, "/notes", http.StatusFound)
}

func (h *Handler) handleHealth(w http.ResponseWriter) {
	setCacheControlPublicHour(w)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("ok"))
}

func (h *Handler) handleNotes(w http.ResponseWriter, r *http.Request) {
	state := notesSignalState{
		Page: parsePage(r.URL.Query().Get("page")),
		Tag:  strings.TrimSpace(r.URL.Query().Get("tag")),
	}

	result, err := h.service.ListNotes(r.Context(), state.Page, state.Tag)
	if err != nil {
		h.serverError(w, fmt.Errorf("list notes: %w", err))
		return
	}

	view := newNotesPageView(result)
	if err := renderComponent(r, w, NotesPage(view)); err != nil {
		h.serverError(w, fmt.Errorf("render notes: %w", err))
		return
	}
}

func (h *Handler) handleNotesLive(w http.ResponseWriter, r *http.Request) {
	state, err := parseNotesState(r)
	if err != nil {
		http.Error(w, "invalid datastar signal payload", http.StatusBadRequest)
		return
	}

	result, err := h.service.ListNotes(r.Context(), state.Page, state.Tag)
	if err != nil {
		h.serverError(w, fmt.Errorf("list notes live: %w", err))
		return
	}

	view := newNotesPageView(result)
	if err := patchComponent(w, r, "notes-content", NotesContent(view)); err != nil {
		h.serverError(w, fmt.Errorf("patch notes: %w", err))
		return
	}
}

func (h *Handler) handleNoteBySlug(w http.ResponseWriter, r *http.Request, slug string) {
	note, err := h.service.GetNoteBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, notes.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		h.serverError(w, fmt.Errorf("get note %q: %w", slug, err))
		return
	}

	view := NotePageView{
		PageTitle: note.Title,
		Note:      *note,
	}
	if err := renderComponent(r, w, NotePage(view)); err != nil {
		h.serverError(w, fmt.Errorf("render note: %w", err))
		return
	}
}

func (h *Handler) handleAuthorBySlug(w http.ResponseWriter, r *http.Request, slug string) {
	page := parsePage(r.URL.Query().Get("page"))
	result, err := h.service.GetAuthorPage(r.Context(), slug, page)
	if err != nil {
		if errors.Is(err, notes.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		h.serverError(w, fmt.Errorf("get author %q: %w", slug, err))
		return
	}

	view := newAuthorPageView(result)
	if err := renderComponent(r, w, AuthorPage(view)); err != nil {
		h.serverError(w, fmt.Errorf("render author page: %w", err))
		return
	}
}

func (h *Handler) handleAuthorBySlugLive(w http.ResponseWriter, r *http.Request, slug string) {
	state, err := parseAuthorState(r)
	if err != nil {
		http.Error(w, "invalid datastar signal payload", http.StatusBadRequest)
		return
	}

	result, err := h.service.GetAuthorPage(r.Context(), slug, state.Page)
	if err != nil {
		if errors.Is(err, notes.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		h.serverError(w, fmt.Errorf("get author live %q: %w", slug, err))
		return
	}

	view := newAuthorPageView(result)
	if err := patchComponent(w, r, "author-content", AuthorContent(view)); err != nil {
		h.serverError(w, fmt.Errorf("patch author: %w", err))
		return
	}
}

func (h *Handler) serverError(w http.ResponseWriter, err error) {
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	log.Printf("blog server error: %v", err)
}

func renderComponent(r *http.Request, w http.ResponseWriter, component templ.Component) error {
	setCacheControlPublicHour(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return component.Render(r.Context(), w)
}

func patchComponent(w http.ResponseWriter, r *http.Request, selectorID string, component templ.Component) error {
	sse := datastar.NewSSE(w, r)
	return sse.PatchElementTempl(component, datastar.WithSelectorID(selectorID))
}

func parseNotesState(r *http.Request) (notesSignalState, error) {
	fallback := notesSignalState{
		Page: parsePage(r.URL.Query().Get("page")),
		Tag:  strings.TrimSpace(r.URL.Query().Get("tag")),
	}

	state, err := readDatastarState(r, fallback)
	if err != nil {
		return notesSignalState{}, err
	}
	state.Page = sanitizePage(state.Page)
	state.Tag = strings.TrimSpace(state.Tag)

	return state, nil
}

func parseAuthorState(r *http.Request) (authorSignalState, error) {
	fallback := authorSignalState{Page: parsePage(r.URL.Query().Get("page"))}
	state, err := readDatastarState(r, fallback)
	if err != nil {
		return authorSignalState{}, err
	}
	state.Page = sanitizePage(state.Page)

	return state, nil
}

func readDatastarState[T interface{}](r *http.Request, fallback T) (T, error) {
	if r.Method == http.MethodGet && strings.TrimSpace(r.URL.Query().Get(datastar.DatastarKey)) == "" {
		return fallback, nil
	}

	parsed := fallback
	if err := datastar.ReadSignals(r, &parsed); err != nil {
		return fallback, err
	}

	return parsed, nil
}

func parsePage(value string) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return 1
	}
	return parsed
}

func buildNotesURL(page int, tag string) string {
	if page < 1 {
		page = 1
	}

	q := make(url.Values)
	if page > 1 {
		q.Set("page", strconv.Itoa(page))
	}
	if strings.TrimSpace(tag) != "" {
		q.Set("tag", tag)
	}

	encoded := q.Encode()
	if encoded == "" {
		return "/notes"
	}
	return "/notes?" + encoded
}

func buildAuthorURL(slug string, page int) string {
	if page < 1 {
		page = 1
	}

	if page == 1 {
		return "/author/" + slug
	}

	q := make(url.Values)
	q.Set("page", strconv.Itoa(page))
	return "/author/" + slug + "?" + q.Encode()
}

func buildAuthorLiveURL(slug string) string {
	return "/author/" + slug + "/live"
}
