package robots

import (
	"blog/internal/req"
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
		if !req.IsReadMethod(r.Method) {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		req.SetCacheControl(w, cachePolicy)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(buildRobotsTXT(rootURL)))
	})
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



