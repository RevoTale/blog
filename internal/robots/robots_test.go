package robots

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildRobotsTXTIncludesSitemap(t *testing.T) {
	t.Parallel()

	robots := buildRobotsTXT("https://revotale.com/blog/notes")
	require.Contains(t, robots, "User-agent: *")
	require.Contains(t, robots, "Allow: /")
	require.Contains(t, robots, "Sitemap: https://revotale.com/blog/notes/sitemap-index")
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
	require.Equal(t, http.StatusOK, recRobots.Code)
	require.Contains(t, recRobots.Header().Get("Content-Type"), "text/plain")
	require.Equal(t, "public, max-age=60", recRobots.Header().Get("Cache-Control"))
	require.Contains(t, recRobots.Body.String(), "Sitemap: https://revotale.com/blog/notes/sitemap-index")

	recMethod := httptest.NewRecorder()
	handler.ServeHTTP(recMethod, httptest.NewRequest(http.MethodPost, "/robots.txt", nil))
	require.Equal(t, http.StatusMethodNotAllowed, recMethod.Code)

	recUnknown := httptest.NewRecorder()
	handler.ServeHTTP(recUnknown, httptest.NewRequest(http.MethodGet, "/unknown", nil))
	require.Equal(t, http.StatusNoContent, recUnknown.Code)
	require.True(t, nextCalled)
}
