package web

import (
	"fmt"
	"log"
	"net/http"

	"blog/framework/engine"
	"blog/internal/config"
	"blog/internal/notes"
	"blog/internal/web/appcore"
	webgen "blog/internal/web/gen"
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

	http.NotFound(w, r)
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

func renderComponent(r *http.Request, w http.ResponseWriter, component templ.Component) error {
	setCacheControlPublicHour(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return component.Render(r.Context(), w)
}

func patchComponent(w http.ResponseWriter, r *http.Request, selectorID string, component templ.Component) error {
	sse := datastar.NewSSE(w, r)
	return sse.PatchElementTempl(component, datastar.WithSelectorID(selectorID))
}
