package httpserver

import (
	"compress/gzip"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"blog/framework"
	"blog/framework/engine"
	frameworki18n "blog/framework/i18n"
	"blog/framework/metagen"
	"github.com/a-h/templ"
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
		IsPartialRequest:  srv.isHTMXRequest,
		RenderPage:        srv.renderPage,
		IsNotFoundError:   cfg.IsNotFoundError,
		HandleNotFound:    srv.handleNotFound,
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
	return withGzipCompression(mux), nil
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
		Locale:      frameworki18n.LocaleFromContext(r.Context()),
		Source:      framework.NotFoundSourceUnmatchedRoute,
	})
}

func (s *server[C]) renderPage(
	r *http.Request,
	w http.ResponseWriter,
	component templ.Component,
	meta metagen.Metadata,
) error {
	cachePolicy := s.cachePolicies.HTML
	if s.isHTMXRequest(r) {
		cachePolicy = s.liveCachePolicyFor(r)
		patch, err := metagen.BuildHTMXPatch(meta)
		if err != nil {
			return fmt.Errorf("build htmx metadata patch: %w", err)
		}
		if err := metagen.WriteHTMXHeaders(w, patch); err != nil {
			return fmt.Errorf("write htmx metadata patch: %w", err)
		}
	}

	return s.renderPageWithStatus(r, w, component, 0, cachePolicy)
}

func (s *server[C]) renderPageWithStatus(
	r *http.Request,
	w http.ResponseWriter,
	component templ.Component,
	statusCode int,
	cachePolicy string,
) error {
	setCachePolicy(w, cachePolicy)
	setVaryHeader(w, "HX-Request")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if statusCode > 0 {
		w.WriteHeader(statusCode)
	}
	return component.Render(r.Context(), w)
}

func (s *server[C]) isHTMXRequest(r *http.Request) bool {
	if r == nil {
		return false
	}

	return strings.EqualFold(strings.TrimSpace(r.Header.Get("HX-Request")), "true")
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

func setVaryHeader(w http.ResponseWriter, header string) {
	if w == nil {
		return
	}
	header = strings.TrimSpace(header)
	if header == "" {
		return
	}

	current := strings.TrimSpace(w.Header().Get("Vary"))
	if current == "" {
		w.Header().Set("Vary", header)
		return
	}

	for _, existing := range strings.Split(current, ",") {
		if strings.EqualFold(strings.TrimSpace(existing), header) {
			return
		}
	}

	parts := strings.Split(current, ",")
	parts = append(parts, header)
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	parts = slices.DeleteFunc(parts, func(value string) bool {
		return value == ""
	})
	w.Header().Set("Vary", strings.Join(parts, ", "))
}

func withGzipCompression(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if next == nil {
			return
		}
		setVaryHeader(w, "Accept-Encoding")
		if r == nil || r.Method == http.MethodHead || !acceptsGzip(r.Header.Get("Accept-Encoding")) {
			next.ServeHTTP(w, r)
			return
		}

		gzipWriter := &gzipResponseWriter{
			ResponseWriter: w,
			compress:       true,
		}
		defer func() {
			_ = gzipWriter.Close()
		}()

		next.ServeHTTP(gzipWriter, r)
	})
}

type gzipResponseWriter struct {
	http.ResponseWriter
	writer      *gzip.Writer
	compress    bool
	wroteHeader bool
}

func (w *gzipResponseWriter) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true

	if !w.compress || !isBodyAllowedForStatus(statusCode) {
		w.compress = false
		w.ResponseWriter.WriteHeader(statusCode)
		return
	}

	header := w.Header()
	if strings.TrimSpace(header.Get("Content-Encoding")) != "" {
		w.compress = false
		w.ResponseWriter.WriteHeader(statusCode)
		return
	}

	header.Del("Content-Length")
	header.Set("Content-Encoding", "gzip")
	w.ResponseWriter.WriteHeader(statusCode)
	w.writer = gzip.NewWriter(w.ResponseWriter)
}

func (w *gzipResponseWriter) Write(content []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	if !w.compress {
		return w.ResponseWriter.Write(content)
	}
	return w.writer.Write(content)
}

func (w *gzipResponseWriter) Flush() {
	if !w.compress {
		if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
			flusher.Flush()
		}
		return
	}

	if w.writer != nil {
		_ = w.writer.Flush()
	}
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *gzipResponseWriter) Close() error {
	if !w.compress || w.writer == nil {
		return nil
	}
	return w.writer.Close()
}

func isBodyAllowedForStatus(statusCode int) bool {
	if statusCode >= 100 && statusCode < 200 {
		return false
	}
	return statusCode != http.StatusNoContent && statusCode != http.StatusNotModified
}

func acceptsGzip(headerValue string) bool {
	for _, part := range strings.Split(headerValue, ",") {
		token := strings.TrimSpace(part)
		if token == "" {
			continue
		}

		encodingToken := token
		quality := 1.0
		if semicolon := strings.Index(token, ";"); semicolon >= 0 {
			encodingToken = strings.TrimSpace(token[:semicolon])
			params := strings.Split(token[semicolon+1:], ";")
			for _, param := range params {
				param = strings.TrimSpace(param)
				if !strings.HasPrefix(strings.ToLower(param), "q=") {
					continue
				}
				value := strings.TrimSpace(param[2:])
				if parsed, err := strconv.ParseFloat(value, 64); err == nil {
					quality = parsed
				}
			}
		}

		if strings.EqualFold(encodingToken, "gzip") && quality > 0 {
			return true
		}
	}

	return false
}
