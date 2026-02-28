package engine

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"blog/framework"
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

func TestServeRouteLiveFirst(t *testing.T) {
	var rendered string
	var patched string

	handlers := []framework.RouteHandler[*testAppContext]{
		framework.PageAndLiveRouteHandler[*testAppContext, framework.EmptyParams, string, string]{
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
			Live: framework.LiveModule[*testAppContext, framework.EmptyParams, string, string]{
				Pattern: "/notes/live",
				ParseParams: func(path string) (framework.EmptyParams, bool) {
					return framework.EmptyParams{}, path == "/notes/live"
				},
				ParseState: func(*http.Request) (string, error) { return "live", nil },
				Load: func(context.Context, *testAppContext, *http.Request, framework.EmptyParams, string) (string, error) {
					return "live", nil
				},
				Render:            func(view string) templ.Component { return textComponent(view) },
				SelectorID:        "notes-content",
				BadRequestMessage: "bad payload",
			},
		},
	}

	routeEngine, err := New(Config[*testAppContext]{
		AppContext: &testAppContext{},
		Handlers:   handlers,
		RenderPage: func(_ *http.Request, _ http.ResponseWriter, component templ.Component) error {
			var b bytes.Buffer
			if err := component.Render(context.Background(), &b); err != nil {
				return err
			}
			rendered = b.String()
			return nil
		},
		PatchLive: func(_ http.ResponseWriter, _ *http.Request, _ string, component templ.Component) error {
			var b bytes.Buffer
			if err := component.Render(context.Background(), &b); err != nil {
				return err
			}
			patched = b.String()
			return nil
		},
	})
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}

	if !routeEngine.ServeRoute(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/notes/live", nil)) {
		t.Fatal("expected live route to match")
	}
	if patched != "live" {
		t.Fatalf("expected live content, got %q", patched)
	}

	if !routeEngine.ServeRoute(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/notes", nil)) {
		t.Fatal("expected page route to match")
	}
	if rendered != "page" {
		t.Fatalf("expected page content, got %q", rendered)
	}
}

func TestLiveStateParseError(t *testing.T) {
	var badRequestMessage string

	routeEngine, err := New(Config[*testAppContext]{
		AppContext: &testAppContext{},
		Handlers: []framework.RouteHandler[*testAppContext]{
			framework.PageAndLiveRouteHandler[*testAppContext, framework.EmptyParams, string, string]{
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
				Live: framework.LiveModule[*testAppContext, framework.EmptyParams, string, string]{
					Pattern: "/notes/live",
					ParseParams: func(path string) (framework.EmptyParams, bool) {
						return framework.EmptyParams{}, path == "/notes/live"
					},
					ParseState: func(*http.Request) (string, error) {
						return "", errors.New("invalid")
					},
					Load: func(context.Context, *testAppContext, *http.Request, framework.EmptyParams, string) (string, error) {
						return "", nil
					},
					Render:            func(view string) templ.Component { return textComponent(view) },
					SelectorID:        "notes-content",
					BadRequestMessage: "invalid datastar signal payload",
				},
			},
		},
		RenderPage: func(*http.Request, http.ResponseWriter, templ.Component) error { return nil },
		PatchLive:  func(http.ResponseWriter, *http.Request, string, templ.Component) error { return nil },
		HandleBadRequest: func(_ http.ResponseWriter, message string) {
			badRequestMessage = message
		},
	})
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}

	if !routeEngine.ServeRoute(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/notes/live", nil)) {
		t.Fatal("expected live route to match")
	}
	if badRequestMessage != "invalid datastar signal payload" {
		t.Fatalf("expected bad request message to be propagated, got %q", badRequestMessage)
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
			RenderPage:      func(*http.Request, http.ResponseWriter, templ.Component) error { return nil },
			PatchLive:       func(http.ResponseWriter, *http.Request, string, templ.Component) error { return nil },
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
			RenderPage:      func(*http.Request, http.ResponseWriter, templ.Component) error { return nil },
			PatchLive:       func(http.ResponseWriter, *http.Request, string, templ.Component) error { return nil },
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
						func(_ string, child templ.Component) templ.Component {
							return wrapComponent("outer", child)
						},
						func(_ string, child templ.Component) templ.Component {
							return wrapComponent("inner", child)
						},
					},
				},
			},
		},
		RenderPage: func(_ *http.Request, _ http.ResponseWriter, component templ.Component) error {
			var b bytes.Buffer
			if err := component.Render(context.Background(), &b); err != nil {
				return err
			}
			rendered = b.String()
			return nil
		},
		PatchLive: func(http.ResponseWriter, *http.Request, string, templ.Component) error { return nil },
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
