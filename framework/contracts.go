package framework

import (
	"context"
	"fmt"
	"net/http"

	"github.com/a-h/templ"
)

type EmptyParams struct{}

type SlugParams struct {
	Slug string
}

type ParamsParser[P interface{}] func(path string) (P, bool)

type PageLoader[C interface{}, P interface{}, VM interface{}] func(
	ctx context.Context,
	appCtx C,
	r *http.Request,
	params P,
) (VM, error)

type PageRenderer[VM interface{}] func(view VM) templ.Component

type LayoutRenderer[VM interface{}] func(view VM, child templ.Component) templ.Component

type PageModule[C interface{}, P interface{}, VM interface{}] struct {
	Pattern     string
	ParseParams ParamsParser[P]
	Load        PageLoader[C, P, VM]
	Render      PageRenderer[VM]
	Layouts     []LayoutRenderer[VM]
}

type RuntimeContext[C interface{}] interface {
	AppContext() C
	IsPartialRequest(r *http.Request) bool
	RenderPage(r *http.Request, w http.ResponseWriter, component templ.Component) error
	IsNotFound(err error) bool
	RespondNotFound(w http.ResponseWriter, r *http.Request, notFoundContext NotFoundContext)
	RespondServerError(w http.ResponseWriter, err error)
}

type NotFoundSource string

const (
	NotFoundSourcePageLoad       NotFoundSource = "page_load"
	NotFoundSourceUnmatchedRoute NotFoundSource = "unmatched_route"
)

type NotFoundContext struct {
	RequestPath         string
	MatchedRoutePattern string
	Source              NotFoundSource
}

type RouteHandler[C interface{}] interface {
	TryServe(runtime RuntimeContext[C], w http.ResponseWriter, r *http.Request) bool
}

type PageOnlyRouteHandler[C interface{}, P interface{}, VM interface{}] struct {
	Page PageModule[C, P, VM]
}

func (h PageOnlyRouteHandler[C, P, VM]) TryServe(
	runtime RuntimeContext[C],
	w http.ResponseWriter,
	r *http.Request,
) bool {
	return servePageModule(runtime, w, r, h.Page)
}

func applyLayouts[VM interface{}](
	layouts []LayoutRenderer[VM],
	view VM,
	child templ.Component,
) templ.Component {
	wrapped := child
	for idx := len(layouts) - 1; idx >= 0; idx-- {
		wrapped = layouts[idx](view, wrapped)
	}
	return wrapped
}

func servePageModule[C interface{}, P interface{}, VM interface{}](
	runtime RuntimeContext[C],
	w http.ResponseWriter,
	r *http.Request,
	module PageModule[C, P, VM],
) bool {
	params, ok := module.ParseParams(r.URL.Path)
	if !ok {
		return false
	}

	view, err := module.Load(r.Context(), runtime.AppContext(), r, params)
	if err != nil {
		handleLoadError(runtime, w, r, err, module.Pattern, NotFoundSourcePageLoad)
		return true
	}

	component := module.Render(view)
	if !runtime.IsPartialRequest(r) {
		component = applyLayouts(module.Layouts, view, component)
	}
	if err := runtime.RenderPage(r, w, component); err != nil {
		runtime.RespondServerError(w, fmt.Errorf("render route %q: %w", module.Pattern, err))
	}
	return true
}

func handleLoadError[C interface{}](
	runtime RuntimeContext[C],
	w http.ResponseWriter,
	r *http.Request,
	err error,
	routePattern string,
	source NotFoundSource,
) {
	if runtime.IsNotFound(err) {
		runtime.RespondNotFound(w, r, NotFoundContext{
			RequestPath:         r.URL.Path,
			MatchedRoutePattern: routePattern,
			Source:              source,
		})
		return
	}

	runtime.RespondServerError(w, fmt.Errorf("load route %q: %w", routePattern, err))
}
