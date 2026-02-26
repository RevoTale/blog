package web

import "net/http"

const cacheControlPublicHour = "public, max-age=3600, s-maxage=3600"

func setCacheControlPublicHour(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", cacheControlPublicHour)
}

func withCacheControlPublicHour(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setCacheControlPublicHour(w)
		next.ServeHTTP(w, r)
	})
}
