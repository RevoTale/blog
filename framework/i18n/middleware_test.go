package i18n

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddlewareRewriteAndContext(t *testing.T) {
	t.Parallel()

	resolver, err := NewResolver(Config{
		Locales:       []string{"en", "uk"},
		DefaultLocale: "en",
		PrefixMode:    PrefixAsNeeded,
	})
	if err != nil {
		t.Fatalf("new resolver: %v", err)
	}

	var gotPath string
	var gotLocale string
	handler := Middleware(MiddlewareConfig{
		Resolver: resolver,
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotLocale = LocaleFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/uk/note/hello?x=1", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: expected %d, got %d", http.StatusOK, rec.Code)
	}
	if gotPath != "/note/hello" {
		t.Fatalf("rewritten path: expected %q, got %q", "/note/hello", gotPath)
	}
	if gotLocale != "uk" {
		t.Fatalf("locale: expected %q, got %q", "uk", gotLocale)
	}
}

func TestMiddlewareCanonicalRedirectAndUnknownLocale(t *testing.T) {
	t.Parallel()

	resolver, err := NewResolver(Config{
		Locales:       []string{"en", "uk"},
		DefaultLocale: "en",
		PrefixMode:    PrefixAsNeeded,
	})
	if err != nil {
		t.Fatalf("new resolver: %v", err)
	}

	handler := Middleware(MiddlewareConfig{
		Resolver: resolver,
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	recRedirect := httptest.NewRecorder()
	reqRedirect := httptest.NewRequest(http.MethodGet, "/en/note/hello?x=1", nil)
	handler.ServeHTTP(recRedirect, reqRedirect)
	if recRedirect.Code != http.StatusPermanentRedirect {
		t.Fatalf("redirect status: expected %d, got %d", http.StatusPermanentRedirect, recRedirect.Code)
	}
	if location := recRedirect.Header().Get("Location"); location != "/note/hello?x=1" {
		t.Fatalf("redirect location: expected %q, got %q", "/note/hello?x=1", location)
	}

	recNotFound := httptest.NewRecorder()
	reqNotFound := httptest.NewRequest(http.MethodGet, "/it/note/hello", nil)
	handler.ServeHTTP(recNotFound, reqNotFound)
	if recNotFound.Code != http.StatusNotFound {
		t.Fatalf("unknown locale status: expected %d, got %d", http.StatusNotFound, recNotFound.Code)
	}
}
