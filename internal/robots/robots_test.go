package robots

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)
func TestBuildRobotsTXTIncludesSitemap(t *testing.T) {
	t.Parallel()

	robots := buildRobotsTXT("https://revotale.com/blog/notes")
	if !strings.Contains(robots, "User-agent: *") {
		t.Fatalf("robots.txt should include user-agent directive")
	}
	if !strings.Contains(robots, "Allow: /") {
		t.Fatalf("robots.txt should include allow directive")
	}
	if !strings.Contains(robots, "Sitemap: https://revotale.com/blog/notes/sitemap-index") {
		t.Fatalf("robots.txt should include sitemap reference")
	}
}
func TestWithRobotsEndpoint(t *testing.T) {
	t.Parallel()

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusNoContent)
	})

	handler := WithRobotsEndpoint(next, "https://revotale.com/blog/notes", "public, max-age=60")

	recRobots := httptest.NewRecorder()
	handler.ServeHTTP(recRobots, httptest.NewRequest(http.MethodGet, "/robots.txt", nil))
	if recRobots.Code != http.StatusOK {
		t.Fatalf("robots status: expected %d, got %d", http.StatusOK, recRobots.Code)
	}
	if contentType := recRobots.Header().Get("Content-Type"); !strings.Contains(contentType, "text/plain") {
		t.Fatalf("robots content-type: expected text/plain, got %q", contentType)
	}
	if got := recRobots.Header().Get("Cache-Control"); got != "public, max-age=60" {
		t.Fatalf("robots cache-control: expected %q, got %q", "public, max-age=60", got)
	}
	if body := recRobots.Body.String(); !strings.Contains(body, "Sitemap: https://revotale.com/blog/notes/sitemap-index") {
		t.Fatalf("robots body should include sitemap reference")
	}

	recMethod := httptest.NewRecorder()
	handler.ServeHTTP(recMethod, httptest.NewRequest(http.MethodPost, "/robots.txt", nil))
	if recMethod.Code != http.StatusMethodNotAllowed {
		t.Fatalf("robots method status: expected %d, got %d", http.StatusMethodNotAllowed, recMethod.Code)
	}

	recUnknown := httptest.NewRecorder()
	handler.ServeHTTP(recUnknown, httptest.NewRequest(http.MethodGet, "/unknown", nil))
	if recUnknown.Code != http.StatusNoContent {
		t.Fatalf("unknown route should delegate to next handler: status=%d", recUnknown.Code)
	}
	if !nextCalled {
		t.Fatalf("unknown route should delegate to next handler")
	}
}
