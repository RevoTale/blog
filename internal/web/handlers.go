package web

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"blog/framework/engine"
	"blog/internal/config"
	"blog/internal/notes"
	"blog/internal/web/appcore"
	"blog/internal/web/components"
	webgen "blog/internal/web/gen"
	r_layout_root "blog/internal/web/gen/r_layout_root"
	"github.com/a-h/templ"
	"github.com/starfederation/datastar-go/datastar"
)

type Handler struct {
	cfg         config.Config
	routeEngine *engine.Engine[*appcore.Context]
}

func NewHandler(cfg config.Config, service *notes.Service) (*Handler, error) {
	handler := &Handler{cfg: cfg}
	appCtx := appcore.NewContext(service)

	routeEngine, err := engine.New(engine.Config[*appcore.Context]{
		AppContext:        appCtx,
		Handlers:          webgen.Handlers(webgen.NewRouteResolvers()),
		RenderPage:        renderComponent,
		PatchLive:         patchComponent,
		IsNotFoundError:   appcore.IsNotFoundError,
		HandleNotFound:    handler.notFound,
		HandleServerError: handler.serverError,
	})
	if err != nil {
		return nil, fmt.Errorf("create route engine: %w", err)
	}

	handler.routeEngine = routeEngine
	return handler, nil
}

func (h *Handler) Register(mux *http.ServeMux) {
	fs := http.FileServer(http.Dir(h.cfg.StaticDir))
	mux.Handle("/static/", withCacheControlPublicHour(http.StripPrefix("/static/", fs)))
	mux.HandleFunc("/", h.handleRoute)
}

func (h *Handler) handleRoute(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/healthz" {
		h.handleHealth(w)
		return
	}

	if h.routeEngine.ServeRoute(w, r) {
		return
	}

	h.notFound(w, r)
}

func (h *Handler) handleHealth(w http.ResponseWriter) {
	setCacheControlPublicHour(w)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("ok"))
}

func (h *Handler) serverError(w http.ResponseWriter, err error) {
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	log.Printf("blog server error: %v", err)
}

func (h *Handler) notFound(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSpace(r.URL.Path)
	if path == "" {
		path = "/"
	}

	view := appcore.NotesPageView{
		PageTitle:   "404 Not Found",
		SidebarMode: appcore.SidebarModeRoot,
		Filter: notes.ListFilter{
			Type: notes.NoteTypeAll,
		},
	}
	component := r_layout_root.Layout(view, components.NotFound(path))
	if err := renderComponentWithStatus(r, w, component, http.StatusNotFound); err != nil {
		h.serverError(w, fmt.Errorf("render not found page: %w", err))
	}
}

func renderComponent(r *http.Request, w http.ResponseWriter, component templ.Component) error {
	return renderComponentWithStatus(r, w, component, 0)
}

func renderComponentWithStatus(
	r *http.Request,
	w http.ResponseWriter,
	component templ.Component,
	statusCode int,
) error {
	setCacheControlPublicHour(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if statusCode > 0 {
		w.WriteHeader(statusCode)
	}
	return component.Render(r.Context(), w)
}

func patchComponent(w http.ResponseWriter, r *http.Request, selectorID string, component templ.Component) error {
	sse := datastar.NewSSE(w, r)
	return sse.PatchElementTempl(component, datastar.WithSelectorID(selectorID))
}
