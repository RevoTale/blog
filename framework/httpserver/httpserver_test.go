package httpserver

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"blog/framework"
	"github.com/a-h/templ"
)

type componentFunc func(ctx context.Context, w io.Writer) error

func (f componentFunc) Render(ctx context.Context, w io.Writer) error {
	return f(ctx, w)
}

func textComponent(value string) templ.Component {
	return componentFunc(func(_ context.Context, w io.Writer) error {
		_, err := io.WriteString(w, value)
		return err
	})
}

func wrapComponent(tag string, child templ.Component) templ.Component {
	return componentFunc(func(ctx context.Context, w io.Writer) error {
		if _, err := io.WriteString(w, "["+tag+"]"); err != nil {
			return err
		}
		if err := child.Render(ctx, w); err != nil {
			return err
		}
		_, err := io.WriteString(w, "[/"+tag+"]")
		return err
	})
}

func TestHTTPServerCachePoliciesAndHTMX(t *testing.T) {
	t.Parallel()

	staticDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(staticDir, "file.txt"), []byte("asset"), 0o644); err != nil {
		t.Fatalf("write static asset: %v", err)
	}

	handler, err := New(Config[*struct{}]{
		AppContext: &struct{}{},
		Handlers: []framework.RouteHandler[*struct{}]{
			framework.PageOnlyRouteHandler[*struct{}, framework.EmptyParams, string]{
				Page: framework.PageModule[*struct{}, framework.EmptyParams, string]{
					Pattern: "/notes",
					ParseParams: func(path string) (framework.EmptyParams, bool) {
						return framework.EmptyParams{}, path == "/notes"
					},
					Load: func(context.Context, *struct{}, *http.Request, framework.EmptyParams) (string, error) {
						return "page", nil
					},
					Render: func(view string) templ.Component { return textComponent(view) },
					Layouts: []framework.LayoutRenderer[string]{
						func(_ string, child templ.Component) templ.Component {
							return wrapComponent("layout", child)
						},
					},
				},
			},
		},
		Static: StaticMount{
			URLPrefix: "/.revotale/",
			Dir:       staticDir,
		},
		CachePolicies: CachePolicies{
			HTML:           "html-cache",
			Live:           "live-cache",
			LiveNavigation: "live-nav-cache",
			Static:         "static-cache",
			Health:         "health-cache",
			Error:          "error-cache",
		},
		NotFoundPage: func(framework.NotFoundContext) templ.Component {
			return textComponent("not-found")
		},
	})
	if err != nil {
		t.Fatalf("new http server: %v", err)
	}

	recPage := httptest.NewRecorder()
	handler.ServeHTTP(recPage, httptest.NewRequest(http.MethodGet, "/notes", nil))
	if recPage.Code != http.StatusOK {
		t.Fatalf("page status: expected %d, got %d", http.StatusOK, recPage.Code)
	}
	if got := recPage.Header().Get("Cache-Control"); got != "html-cache" {
		t.Fatalf("page cache policy: expected %q, got %q", "html-cache", got)
	}
	if got := recPage.Header().Get("Vary"); !strings.Contains(got, "HX-Request") {
		t.Fatalf("page vary header: expected HX-Request, got %q", got)
	}
	if body := strings.TrimSpace(recPage.Body.String()); body != "[layout]page[/layout]" {
		t.Fatalf("page body: expected layout-wrapped response, got %q", body)
	}

	reqHTMX := httptest.NewRequest(http.MethodGet, "/notes", nil)
	reqHTMX.Header.Set("HX-Request", "true")
	recHTMX := httptest.NewRecorder()
	handler.ServeHTTP(recHTMX, reqHTMX)
	if recHTMX.Code != http.StatusOK {
		t.Fatalf("htmx status: expected %d, got %d", http.StatusOK, recHTMX.Code)
	}
	if got := recHTMX.Header().Get("Cache-Control"); got != "live-cache" {
		t.Fatalf("htmx cache policy: expected %q, got %q", "live-cache", got)
	}
	if body := strings.TrimSpace(recHTMX.Body.String()); body != "page" {
		t.Fatalf("htmx body: expected partial response, got %q", body)
	}

	reqHTMXNav := httptest.NewRequest(http.MethodGet, "/notes?__live=navigation", nil)
	reqHTMXNav.Header.Set("HX-Request", "true")
	recHTMXNav := httptest.NewRecorder()
	handler.ServeHTTP(recHTMXNav, reqHTMXNav)
	if recHTMXNav.Code != http.StatusOK {
		t.Fatalf("htmx nav status: expected %d, got %d", http.StatusOK, recHTMXNav.Code)
	}
	if got := recHTMXNav.Header().Get("Cache-Control"); got != "live-nav-cache" {
		t.Fatalf("htmx nav cache policy: expected %q, got %q", "live-nav-cache", got)
	}

	recStatic := httptest.NewRecorder()
	handler.ServeHTTP(recStatic, httptest.NewRequest(http.MethodGet, "/.revotale/file.txt", nil))
	if recStatic.Code != http.StatusOK {
		t.Fatalf("static status: expected %d, got %d", http.StatusOK, recStatic.Code)
	}
	if got := recStatic.Header().Get("Cache-Control"); got != "static-cache" {
		t.Fatalf("static cache policy: expected %q, got %q", "static-cache", got)
	}

	recHealth := httptest.NewRecorder()
	handler.ServeHTTP(recHealth, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if recHealth.Code != http.StatusOK {
		t.Fatalf("health status: expected %d, got %d", http.StatusOK, recHealth.Code)
	}
	if got := recHealth.Header().Get("Cache-Control"); got != "health-cache" {
		t.Fatalf("health cache policy: expected %q, got %q", "health-cache", got)
	}
	if body := strings.TrimSpace(recHealth.Body.String()); body != "ok" {
		t.Fatalf("health body: expected %q, got %q", "ok", body)
	}
}

func TestHTTPServerNotFoundContextForLoadAndUnmatched(t *testing.T) {
	t.Parallel()

	errNotFound := errors.New("not found")
	ctxs := make([]framework.NotFoundContext, 0, 2)

	handler, err := New(Config[*struct{}]{
		AppContext: &struct{}{},
		Handlers: []framework.RouteHandler[*struct{}]{
			framework.PageOnlyRouteHandler[*struct{}, framework.EmptyParams, string]{
				Page: framework.PageModule[*struct{}, framework.EmptyParams, string]{
					Pattern: "/notes",
					ParseParams: func(path string) (framework.EmptyParams, bool) {
						return framework.EmptyParams{}, path == "/notes"
					},
					Load: func(context.Context, *struct{}, *http.Request, framework.EmptyParams) (string, error) {
						return "", errNotFound
					},
					Render: func(view string) templ.Component { return textComponent(view) },
				},
			},
		},
		IsNotFoundError: func(err error) bool { return errors.Is(err, errNotFound) },
		NotFoundPage: func(notFoundContext framework.NotFoundContext) templ.Component {
			ctxs = append(ctxs, notFoundContext)
			return textComponent("missing")
		},
		CachePolicies: CachePolicies{
			Error: "error-cache",
		},
	})
	if err != nil {
		t.Fatalf("new http server: %v", err)
	}

	recLoadNotFound := httptest.NewRecorder()
	handler.ServeHTTP(recLoadNotFound, httptest.NewRequest(http.MethodGet, "/notes", nil))
	if recLoadNotFound.Code != http.StatusNotFound {
		t.Fatalf("load not found status: expected %d, got %d", http.StatusNotFound, recLoadNotFound.Code)
	}
	if got := recLoadNotFound.Header().Get("Cache-Control"); got != "error-cache" {
		t.Fatalf("load not found cache policy: expected %q, got %q", "error-cache", got)
	}

	recUnmatched := httptest.NewRecorder()
	handler.ServeHTTP(recUnmatched, httptest.NewRequest(http.MethodGet, "/missing", nil))
	if recUnmatched.Code != http.StatusNotFound {
		t.Fatalf("unmatched status: expected %d, got %d", http.StatusNotFound, recUnmatched.Code)
	}

	if len(ctxs) != 2 {
		t.Fatalf("expected 2 not-found contexts, got %d", len(ctxs))
	}
	if ctxs[0].Source != framework.NotFoundSourcePageLoad {
		t.Fatalf("expected first source %q, got %q", framework.NotFoundSourcePageLoad, ctxs[0].Source)
	}
	if ctxs[0].MatchedRoutePattern != "/notes" {
		t.Fatalf("expected first matched pattern /notes, got %q", ctxs[0].MatchedRoutePattern)
	}
	if ctxs[1].Source != framework.NotFoundSourceUnmatchedRoute {
		t.Fatalf("expected second source %q, got %q", framework.NotFoundSourceUnmatchedRoute, ctxs[1].Source)
	}
	if ctxs[1].RequestPath != "/missing" {
		t.Fatalf("expected second request path /missing, got %q", ctxs[1].RequestPath)
	}
}
