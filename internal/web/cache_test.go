package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithCacheControlPublicHour(t *testing.T) {
	handler := withCacheControlPublicHour(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/static/tui.css", nil)
	handler.ServeHTTP(recorder, request)

	if got := recorder.Header().Get("Cache-Control"); got != cacheControlPublicHour {
		t.Fatalf("expected cache-control %q, got %q", cacheControlPublicHour, got)
	}
}
