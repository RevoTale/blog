package httpserver

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"blog/framework"
	"blog/framework/engine"
	"github.com/a-h/templ"
	"github.com/starfederation/datastar-go/datastar"
)

const defaultCacheControlPolicy = "public, max-age=3600, s-maxage=3600"
const defaultHealthPath = "/healthz"
const defaultHealthBody = "ok"
const defaultStaticPrefix = "/.revotale/"
const liveNavigationMarkerKey = "__live"
const liveNavigationMarkerValue = "navigation"

type StaticMount struct {
	URLPrefix string
	Dir       string
}

type CachePolicies struct {
	HTML           string
	Live           string
	LiveNavigation string
	Static         string
	Health         string
	Error          string
}

func DefaultCachePolicies() CachePolicies {
	return CachePolicies{
		HTML:   defaultCacheControlPolicy,
		Live:   defaultCacheControlPolicy,
		Static: defaultCacheControlPolicy,
		Health: defaultCacheControlPolicy,
		Error:  defaultCacheControlPolicy,
	}
}

type Config[C interface{}] struct {
	AppContext C
	Handlers   []framework.RouteHandler[C]

	Static StaticMount

	CachePolicies CachePolicies

	IsNotFoundError func(err error) bool
	NotFoundPage    func(notFoundContext framework.NotFoundContext) templ.Component
	LogServerError  func(err error)

	HealthPath string
	HealthBody string
}

type server[C interface{}] struct {
	cachePolicies CachePolicies
	notFoundPage  func(notFoundContext framework.NotFoundContext) templ.Component
	logServerErr  func(err error)
	healthPath    string
	healthBody    string

	routeEngine *engine.Engine[C]
}

func New[C interface{}](cfg Config[C]) (http.Handler, error) {
	cachePolicies := withDefaultPolicies(cfg.CachePolicies)
	healthPath := normalizeHealthPath(cfg.HealthPath)
	healthBody := strings.TrimSpace(cfg.HealthBody)
	if healthBody == "" {
		healthBody = defaultHealthBody
	}

	srv := &server[C]{
		cachePolicies: cachePolicies,
		notFoundPage:  cfg.NotFoundPage,
		logServerErr:  cfg.LogServerError,
		healthPath:    healthPath,
		healthBody:    healthBody,
	}

	routeEngine, err := engine.New(engine.Config[C]{
		AppContext:        cfg.AppContext,
		Handlers:          cfg.Handlers,
		RenderPage:        srv.renderPage,
		PatchLive:         srv.patchLive,
		IsNotFoundError:   cfg.IsNotFoundError,
		HandleNotFound:    srv.handleNotFound,
		HandleBadRequest:  srv.handleBadRequest,
		HandleServerError: srv.handleServerError,
	})
	if err != nil {
		return nil, fmt.Errorf("create route engine: %w", err)
	}
	srv.routeEngine = routeEngine

	mux := http.NewServeMux()
	if strings.TrimSpace(cfg.Static.Dir) != "" {
		prefix := normalizeStaticPrefix(cfg.Static.URLPrefix)
		fs := http.FileServer(http.Dir(cfg.Static.Dir))
		mux.Handle(prefix, withCachePolicy(cachePolicies.Static, http.StripPrefix(prefix, fs)))
	}

	mux.HandleFunc("/", srv.handleRoute)
	return mux, nil
}

func (s *server[C]) handleRoute(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == s.healthPath {
		s.handleHealth(w)
		return
	}

	if s.routeEngine.ServeRoute(w, r) {
		return
	}

	s.handleNotFound(w, r, framework.NotFoundContext{
		RequestPath: r.URL.Path,
		Source:      framework.NotFoundSourceUnmatchedRoute,
	})
}

func (s *server[C]) renderPage(r *http.Request, w http.ResponseWriter, component templ.Component) error {
	return s.renderPageWithStatus(r, w, component, 0, s.cachePolicies.HTML)
}

func (s *server[C]) renderPageWithStatus(
	r *http.Request,
	w http.ResponseWriter,
	component templ.Component,
	statusCode int,
	cachePolicy string,
) error {
	setCachePolicy(w, cachePolicy)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if statusCode > 0 {
		w.WriteHeader(statusCode)
	}
	return component.Render(r.Context(), w)
}

func (s *server[C]) patchLive(
	w http.ResponseWriter,
	r *http.Request,
	selectorID string,
	component templ.Component,
) error {
	sse := datastar.NewSSE(w, r)
	setCachePolicy(w, s.liveCachePolicyFor(r))
	return sse.PatchElementTempl(component, datastar.WithSelectorID(selectorID))
}

func (s *server[C]) liveCachePolicyFor(r *http.Request) string {
	if r != nil &&
		strings.TrimSpace(r.URL.Query().Get(liveNavigationMarkerKey)) == liveNavigationMarkerValue &&
		strings.TrimSpace(s.cachePolicies.LiveNavigation) != "" {
		return s.cachePolicies.LiveNavigation
	}

	return s.cachePolicies.Live
}

func (s *server[C]) handleNotFound(
	w http.ResponseWriter,
	r *http.Request,
	notFoundContext framework.NotFoundContext,
) {
	if s.notFoundPage == nil {
		setCachePolicy(w, s.cachePolicies.Error)
		http.NotFound(w, r)
		return
	}

	component := s.notFoundPage(notFoundContext)
	if component == nil {
		setCachePolicy(w, s.cachePolicies.Error)
		http.NotFound(w, r)
		return
	}
	if err := s.renderPageWithStatus(r, w, component, http.StatusNotFound, s.cachePolicies.Error); err != nil {
		s.handleServerError(w, fmt.Errorf("render not found page: %w", err))
	}
}

func (s *server[C]) handleBadRequest(w http.ResponseWriter, message string) {
	setCachePolicy(w, s.cachePolicies.Error)
	http.Error(w, message, http.StatusBadRequest)
}

func (s *server[C]) handleServerError(w http.ResponseWriter, err error) {
	setCachePolicy(w, s.cachePolicies.Error)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	if s.logServerErr != nil {
		s.logServerErr(err)
		return
	}

	log.Printf("framework server error: %v", err)
}

func (s *server[C]) handleHealth(w http.ResponseWriter) {
	setCachePolicy(w, s.cachePolicies.Health)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(s.healthBody))
}

func normalizeStaticPrefix(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return defaultStaticPrefix
	}
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return prefix
}

func normalizeHealthPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return defaultHealthPath
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

func withDefaultPolicies(policies CachePolicies) CachePolicies {
	defaults := DefaultCachePolicies()
	if strings.TrimSpace(policies.HTML) == "" {
		policies.HTML = defaults.HTML
	}
	if strings.TrimSpace(policies.Live) == "" {
		policies.Live = defaults.Live
	}
	if strings.TrimSpace(policies.Static) == "" {
		policies.Static = defaults.Static
	}
	if strings.TrimSpace(policies.Health) == "" {
		policies.Health = defaults.Health
	}
	if strings.TrimSpace(policies.Error) == "" {
		policies.Error = defaults.Error
	}
	return policies
}

func setCachePolicy(w http.ResponseWriter, policy string) {
	policy = strings.TrimSpace(policy)
	if policy == "" {
		return
	}
	w.Header().Set("Cache-Control", policy)
}

func withCachePolicy(policy string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setCachePolicy(w, policy)
		next.ServeHTTP(w, r)
	})
}
