package discovery

import (
	"blog/internal/requestcache"
	"fmt"
	"net/http"
	"strings"
)

func WithRobotsEndpoint(
	next http.Handler,
	rootURL string,
	cachePolicy string,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if next == nil {
			return
		}
		if r == nil || r.URL == nil {
			next.ServeHTTP(w, r)
			return
		}
		if r.URL.Path != "/robots.txt" {
			next.ServeHTTP(w, r)
			return
		}
		if !requestcache.IsReadMethod(r.Method) {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		requestcache.SetCacheControl(w, cachePolicy)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(buildRobotsTXT(rootURL)))
	})
}

func MountRobotsEndpoint(mux *http.ServeMux, rootURL string, cachePolicy string) error {
	if mux == nil {
		return fmt.Errorf("mux is required")
	}

	mux.Handle("/robots.txt", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r == nil || r.URL == nil {
			return
		}
		if !requestcache.IsReadMethod(r.Method) {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		requestcache.SetCacheControl(w, cachePolicy)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(buildRobotsTXT(rootURL)))
	}))

	return nil
}

func buildRobotsTXT(rootURL string) string {
	out := []string{
		"User-agent: *",
		"Allow: /",
	}
	trimmedRoot := strings.TrimSuffix(strings.TrimSpace(rootURL), "/")
	if trimmedRoot != "" {
		out = append(out, "Sitemap: "+trimmedRoot+"/sitemap-index")
	}
	return strings.Join(out, "\n") + "\n"
}
