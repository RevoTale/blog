package engine

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"blog/framework"
	"blog/framework/metagen"
	"github.com/a-h/templ"
)

type testAppContext struct{}

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

func TestServeRoutePageOnly(t *testing.T) {
	var rendered string

	routeEngine, err := New(Config[*testAppContext]{
		AppContext: &testAppContext{},
		Handlers: []framework.RouteHandler[*testAppContext]{
			framework.PageOnlyRouteHandler[*testAppContext, framework.EmptyParams, string]{
				Page: framework.PageModule[*testAppContext, framework.EmptyParams, string]{
					Pattern: "/notes",
					ParseParams: func(path string) (framework.EmptyParams, bool) {
						return framework.EmptyParams{}, path == "/notes"
					},
					Load: func(context.Context, *testAppContext, *http.Request, framework.EmptyParams) (string, error) {
						return "page", nil
					},
					Render: func(view string) templ.Component { return textComponent(view) },
				},
			},
		},
		RenderPage: func(_ *http.Request, _ http.ResponseWriter, component templ.Component, _ metagen.Metadata) error {
			var b bytes.Buffer
			if err := component.Render(context.Background(), &b); err != nil {
				return err
			}
			rendered = b.String()
			return nil
		},
	})
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}

	if !routeEngine.ServeRoute(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/notes", nil)) {
		t.Fatal("expected route to match")
	}
	if rendered != "page" {
		t.Fatalf("expected page content, got %q", rendered)
	}

	if routeEngine.ServeRoute(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/missing", nil)) {
		t.Fatal("did not expect missing route to match")
	}
}

func TestServeRouteSkipsLayoutsForPartialRequests(t *testing.T) {
	var rendered string

	routeEngine, err := New(Config[*testAppContext]{
		AppContext: &testAppContext{},
		Handlers: []framework.RouteHandler[*testAppContext]{
			framework.PageOnlyRouteHandler[*testAppContext, framework.EmptyParams, string]{
				Page: framework.PageModule[*testAppContext, framework.EmptyParams, string]{
					Pattern: "/notes",
					ParseParams: func(path string) (framework.EmptyParams, bool) {
						return framework.EmptyParams{}, path == "/notes"
					},
					Load: func(context.Context, *testAppContext, *http.Request, framework.EmptyParams) (string, error) {
						return "body", nil
					},
					Render: func(view string) templ.Component { return textComponent(view) },
					Layouts: []framework.LayoutRenderer[string]{
						func(_ metagen.Metadata, _ string, child templ.Component) templ.Component {
							return wrapComponent("layout", child)
						},
					},
				},
			},
		},
		IsPartialRequest: func(_ *http.Request) bool { return true },
		RenderPage: func(_ *http.Request, _ http.ResponseWriter, component templ.Component, _ metagen.Metadata) error {
			var b bytes.Buffer
			if err := component.Render(context.Background(), &b); err != nil {
				return err
			}
			rendered = b.String()
			return nil
		},
	})
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}

	if !routeEngine.ServeRoute(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/notes", nil)) {
		t.Fatal("expected route to match")
	}
	if rendered != "body" {
		t.Fatalf("expected partial body without layout, got %q", rendered)
	}
}

func TestNotFoundAndServerErrorClassification(t *testing.T) {
	errNotFound := errors.New("not found")
	errBoom := errors.New("boom")

	t.Run("not found", func(t *testing.T) {
		notFoundCalled := false
		serverErrorCalled := false
		var notFoundContext framework.NotFoundContext

		routeEngine, err := New(Config[*testAppContext]{
			AppContext: &testAppContext{},
			Handlers: []framework.RouteHandler[*testAppContext]{
				framework.PageOnlyRouteHandler[*testAppContext, framework.EmptyParams, string]{
					Page: framework.PageModule[*testAppContext, framework.EmptyParams, string]{
						Pattern: "/notes",
						ParseParams: func(path string) (framework.EmptyParams, bool) {
							return framework.EmptyParams{}, path == "/notes"
						},
						Load: func(context.Context, *testAppContext, *http.Request, framework.EmptyParams) (string, error) {
							return "", errNotFound
						},
						Render: func(view string) templ.Component { return textComponent(view) },
					},
				},
			},
			RenderPage:      func(*http.Request, http.ResponseWriter, templ.Component, metagen.Metadata) error { return nil },
			IsNotFoundError: func(err error) bool { return errors.Is(err, errNotFound) },
			HandleNotFound: func(_ http.ResponseWriter, _ *http.Request, ctx framework.NotFoundContext) {
				notFoundCalled = true
				notFoundContext = ctx
			},
			HandleServerError: func(http.ResponseWriter, error) {
				serverErrorCalled = true
			},
		})
		if err != nil {
			t.Fatalf("new engine: %v", err)
		}

		if !routeEngine.ServeRoute(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/notes", nil)) {
			t.Fatal("expected route to match")
		}
		if !notFoundCalled {
			t.Fatal("expected not found callback")
		}
		if notFoundContext.Source != framework.NotFoundSourcePageLoad {
			t.Fatalf("expected not-found source %q, got %q", framework.NotFoundSourcePageLoad, notFoundContext.Source)
		}
		if notFoundContext.MatchedRoutePattern != "/notes" {
			t.Fatalf("expected matched route pattern /notes, got %q", notFoundContext.MatchedRoutePattern)
		}
		if notFoundContext.RequestPath != "/notes" {
			t.Fatalf("expected request path /notes, got %q", notFoundContext.RequestPath)
		}
		if serverErrorCalled {
			t.Fatal("did not expect server error callback")
		}
	})

	t.Run("server error", func(t *testing.T) {
		notFoundCalled := false
		serverErrorCalled := false

		routeEngine, err := New(Config[*testAppContext]{
			AppContext: &testAppContext{},
			Handlers: []framework.RouteHandler[*testAppContext]{
				framework.PageOnlyRouteHandler[*testAppContext, framework.EmptyParams, string]{
					Page: framework.PageModule[*testAppContext, framework.EmptyParams, string]{
						Pattern: "/notes",
						ParseParams: func(path string) (framework.EmptyParams, bool) {
							return framework.EmptyParams{}, path == "/notes"
						},
						Load: func(context.Context, *testAppContext, *http.Request, framework.EmptyParams) (string, error) {
							return "", errBoom
						},
						Render: func(view string) templ.Component { return textComponent(view) },
					},
				},
			},
			RenderPage:      func(*http.Request, http.ResponseWriter, templ.Component, metagen.Metadata) error { return nil },
			IsNotFoundError: func(error) bool { return false },
			HandleNotFound: func(http.ResponseWriter, *http.Request, framework.NotFoundContext) {
				notFoundCalled = true
			},
			HandleServerError: func(http.ResponseWriter, error) {
				serverErrorCalled = true
			},
		})
		if err != nil {
			t.Fatalf("new engine: %v", err)
		}

		if !routeEngine.ServeRoute(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/notes", nil)) {
			t.Fatal("expected route to match")
		}
		if notFoundCalled {
			t.Fatal("did not expect not found callback")
		}
		if !serverErrorCalled {
			t.Fatal("expected server error callback")
		}
	})
}

func TestLayoutOrder(t *testing.T) {
	var rendered string

	routeEngine, err := New(Config[*testAppContext]{
		AppContext: &testAppContext{},
		Handlers: []framework.RouteHandler[*testAppContext]{
			framework.PageOnlyRouteHandler[*testAppContext, framework.EmptyParams, string]{
				Page: framework.PageModule[*testAppContext, framework.EmptyParams, string]{
					Pattern: "/notes",
					ParseParams: func(path string) (framework.EmptyParams, bool) {
						return framework.EmptyParams{}, path == "/notes"
					},
					Load: func(context.Context, *testAppContext, *http.Request, framework.EmptyParams) (string, error) {
						return "body", nil
					},
					Render: func(view string) templ.Component { return textComponent(view) },
					Layouts: []framework.LayoutRenderer[string]{
						func(_ metagen.Metadata, _ string, child templ.Component) templ.Component {
							return wrapComponent("outer", child)
						},
						func(_ metagen.Metadata, _ string, child templ.Component) templ.Component {
							return wrapComponent("inner", child)
						},
					},
				},
			},
		},
		RenderPage: func(_ *http.Request, _ http.ResponseWriter, component templ.Component, _ metagen.Metadata) error {
			var b bytes.Buffer
			if err := component.Render(context.Background(), &b); err != nil {
				return err
			}
			rendered = b.String()
			return nil
		},
	})
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}

	if !routeEngine.ServeRoute(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/notes", nil)) {
		t.Fatal("expected route to match")
	}
	if rendered != "[outer][inner]body[/inner][/outer]" {
		t.Fatalf("unexpected render output: %q", rendered)
	}
}

func TestMetaGenRunsConcurrentlyWithLoad(t *testing.T) {
	t.Parallel()

	loadStarted := make(chan struct{})
	metaStarted := make(chan struct{})
	errConcurrent := errors.New("meta did not observe concurrent load start")
	serverErrCalled := false

	routeEngine, err := New(Config[*testAppContext]{
		AppContext: &testAppContext{},
		Handlers: []framework.RouteHandler[*testAppContext]{
			framework.PageOnlyRouteHandler[*testAppContext, framework.EmptyParams, string]{
				Page: framework.PageModule[*testAppContext, framework.EmptyParams, string]{
					Pattern: "/notes",
					ParseParams: func(path string) (framework.EmptyParams, bool) {
						return framework.EmptyParams{}, path == "/notes"
					},
					MetaGen: func(context.Context, *testAppContext, *http.Request, framework.EmptyParams) (metagen.Metadata, error) {
						close(metaStarted)
						select {
						case <-loadStarted:
							return metagen.Metadata{Title: "Notes"}, nil
						case <-time.After(500 * time.Millisecond):
							return metagen.Metadata{}, errConcurrent
						}
					},
					Load: func(context.Context, *testAppContext, *http.Request, framework.EmptyParams) (string, error) {
						close(loadStarted)
						select {
						case <-metaStarted:
						case <-time.After(500 * time.Millisecond):
							return "", errors.New("load did not observe metagen start")
						}
						return "body", nil
					},
					Render: func(view string) templ.Component {
						return textComponent(view)
					},
				},
			},
		},
		RenderPage: func(_ *http.Request, _ http.ResponseWriter, component templ.Component, _ metagen.Metadata) error {
			var b bytes.Buffer
			if err := component.Render(context.Background(), &b); err != nil {
				return err
			}
			if b.String() != "body" {
				return errors.New("unexpected body render")
			}
			return nil
		},
		HandleServerError: func(_ http.ResponseWriter, err error) {
			if errors.Is(err, errConcurrent) {
				serverErrCalled = true
			}
		},
	})
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}

	if !routeEngine.ServeRoute(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/notes", nil)) {
		t.Fatal("expected route to match")
	}
	if serverErrCalled {
		t.Fatal("metagen/load should run concurrently without server error")
	}
}

func TestMetaGenErrorPrefersMetadataClassification(t *testing.T) {
	t.Parallel()

	errNotFound := errors.New("not found")
	loadCanceled := make(chan struct{})
	renderCalled := false
	notFoundCalled := false

	routeEngine, err := New(Config[*testAppContext]{
		AppContext: &testAppContext{},
		Handlers: []framework.RouteHandler[*testAppContext]{
			framework.PageOnlyRouteHandler[*testAppContext, framework.EmptyParams, string]{
				Page: framework.PageModule[*testAppContext, framework.EmptyParams, string]{
					Pattern: "/notes",
					ParseParams: func(path string) (framework.EmptyParams, bool) {
						return framework.EmptyParams{}, path == "/notes"
					},
					MetaGen: func(context.Context, *testAppContext, *http.Request, framework.EmptyParams) (metagen.Metadata, error) {
						return metagen.Metadata{}, errNotFound
					},
					Load: func(ctx context.Context, _ *testAppContext, _ *http.Request, _ framework.EmptyParams) (string, error) {
						defer close(loadCanceled)
						<-ctx.Done()
						return "", ctx.Err()
					},
					Render: func(view string) templ.Component {
						return textComponent(view)
					},
				},
			},
		},
		RenderPage: func(_ *http.Request, _ http.ResponseWriter, _ templ.Component, _ metagen.Metadata) error {
			renderCalled = true
			return nil
		},
		IsNotFoundError: func(err error) bool {
			return errors.Is(err, errNotFound)
		},
		HandleNotFound: func(_ http.ResponseWriter, _ *http.Request, ctx framework.NotFoundContext) {
			notFoundCalled = true
			if ctx.Source != framework.NotFoundSourceMetaGen {
				t.Fatalf("expected not-found source %q, got %q", framework.NotFoundSourceMetaGen, ctx.Source)
			}
		},
	})
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}

	if !routeEngine.ServeRoute(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/notes", nil)) {
		t.Fatal("expected route to match")
	}
	if renderCalled {
		t.Fatal("render callback should not run when metagen fails")
	}
	if !notFoundCalled {
		t.Fatal("expected not-found callback for metagen not found")
	}
	select {
	case <-loadCanceled:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected load context cancellation after metagen failure")
	}
}

func TestLoadFailureAfterRootRenderUsesErrorPage(t *testing.T) {
	t.Parallel()

	errBoom := errors.New("boom")
	renderCalled := false
	loggedError := ""
	var rendered string

	routeEngine, err := New(Config[*testAppContext]{
		AppContext: &testAppContext{},
		Handlers: []framework.RouteHandler[*testAppContext]{
			framework.PageOnlyRouteHandler[*testAppContext, framework.EmptyParams, string]{
				Page: framework.PageModule[*testAppContext, framework.EmptyParams, string]{
					Pattern: "/notes",
					ParseParams: func(path string) (framework.EmptyParams, bool) {
						return framework.EmptyParams{}, path == "/notes"
					},
					MetaGen: func(context.Context, *testAppContext, *http.Request, framework.EmptyParams) (metagen.Metadata, error) {
						return metagen.Metadata{Title: "Notes"}, nil
					},
					Load: func(context.Context, *testAppContext, *http.Request, framework.EmptyParams) (string, error) {
						return "", errBoom
					},
					Render: func(view string) templ.Component {
						renderCalled = true
						return textComponent(view)
					},
					RootLayout: func(meta metagen.Metadata, _ string, child templ.Component) templ.Component {
						return componentFunc(func(ctx context.Context, w io.Writer) error {
							if _, err := io.WriteString(w, "<html><head>"+meta.Title+"</head><body>"); err != nil {
								return err
							}
							if err := child.Render(ctx, w); err != nil {
								return err
							}
							_, err := io.WriteString(w, "</body></html>")
							return err
						})
					},
					ErrorPage: func(_ string, path string) templ.Component {
						return textComponent("error:" + path)
					},
				},
			},
		},
		RenderPage: func(_ *http.Request, _ http.ResponseWriter, component templ.Component, _ metagen.Metadata) error {
			var b bytes.Buffer
			if err := component.Render(context.Background(), &b); err != nil {
				return err
			}
			rendered = b.String()
			return nil
		},
		LogServerError: func(err error) {
			loggedError = err.Error()
		},
	})
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}

	if !routeEngine.ServeRoute(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/notes", nil)) {
		t.Fatal("expected route to match")
	}
	if renderCalled {
		t.Fatal("page renderer should not be called when load fails after stream start")
	}
	if !strings.Contains(rendered, "<html><head>Notes</head><body>error:/notes</body></html>") {
		t.Fatalf("unexpected streamed error output: %q", rendered)
	}
	if !strings.Contains(loggedError, "after stream start") {
		t.Fatalf("expected post-stream load error to be logged, got %q", loggedError)
	}
}
