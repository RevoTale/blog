package framework

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/a-h/templ"
)

type EmptyParams struct{}

type SlugParams struct {
	Slug string
}

type ParamsParser[P interface{}] func(path string) (P, bool)
type StateParser[S interface{}] func(r *http.Request) (S, error)

type PageLoader[C interface{}, P interface{}, VM interface{}] func(
	ctx context.Context,
	appCtx C,
	r *http.Request,
	params P,
) (VM, error)

type LiveLoader[C interface{}, P interface{}, VM interface{}, S interface{}] func(
	ctx context.Context,
	appCtx C,
	r *http.Request,
	params P,
	state S,
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

type LiveModule[C interface{}, P interface{}, VM interface{}, S interface{}] struct {
	Pattern           string
	ParseParams       ParamsParser[P]
	ParseState        StateParser[S]
	Load              LiveLoader[C, P, VM, S]
	Render            PageRenderer[VM]
	SelectorID        string
	BadRequestMessage string
}

type RuntimeContext[C interface{}] interface {
	AppContext() C
	RenderPage(r *http.Request, w http.ResponseWriter, component templ.Component) error
	PatchLive(w http.ResponseWriter, r *http.Request, selectorID string, component templ.Component) error
	IsNotFound(err error) bool
	RespondNotFound(w http.ResponseWriter, r *http.Request)
	RespondBadRequest(w http.ResponseWriter, message string)
	RespondServerError(w http.ResponseWriter, err error)
}

type RouteHandler[C interface{}] interface {
	TryServePage(runtime RuntimeContext[C], w http.ResponseWriter, r *http.Request) bool
	TryServeLive(runtime RuntimeContext[C], w http.ResponseWriter, r *http.Request) bool
}

type PageOnlyRouteHandler[C interface{}, P interface{}, VM interface{}] struct {
	Page PageModule[C, P, VM]
}

type PageAndLiveRouteHandler[C interface{}, P interface{}, VM interface{}, S interface{}] struct {
	Page PageModule[C, P, VM]
	Live LiveModule[C, P, VM, S]
}

func (h PageOnlyRouteHandler[C, P, VM]) TryServePage(
	runtime RuntimeContext[C],
	w http.ResponseWriter,
	r *http.Request,
) bool {
	return servePageModule(runtime, w, r, h.Page)
}

func (h PageOnlyRouteHandler[C, P, VM]) TryServeLive(
	RuntimeContext[C],
	http.ResponseWriter,
	*http.Request,
) bool {
	return false
}

func (h PageAndLiveRouteHandler[C, P, VM, S]) TryServePage(
	runtime RuntimeContext[C],
	w http.ResponseWriter,
	r *http.Request,
) bool {
	return servePageModule(runtime, w, r, h.Page)
}

func (h PageAndLiveRouteHandler[C, P, VM, S]) TryServeLive(
	runtime RuntimeContext[C],
	w http.ResponseWriter,
	r *http.Request,
) bool {
	return serveLiveModule(runtime, w, r, h.Live)
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
		handleLoadError(runtime, w, r, err, module.Pattern)
		return true
	}

	component := module.Render(view)
	component = applyLayouts(module.Layouts, view, component)
	if err := runtime.RenderPage(r, w, component); err != nil {
		runtime.RespondServerError(w, fmt.Errorf("render route %q: %w", module.Pattern, err))
	}
	return true
}

func serveLiveModule[C interface{}, P interface{}, VM interface{}, S interface{}](
	runtime RuntimeContext[C],
	w http.ResponseWriter,
	r *http.Request,
	module LiveModule[C, P, VM, S],
) bool {
	params, ok := module.ParseParams(r.URL.Path)
	if !ok {
		return false
	}

	state, err := module.ParseState(r)
	if err != nil {
		message := strings.TrimSpace(module.BadRequestMessage)
		if message == "" {
			message = "invalid request payload"
		}
		runtime.RespondBadRequest(w, message)
		return true
	}

	view, err := module.Load(r.Context(), runtime.AppContext(), r, params, state)
	if err != nil {
		handleLoadError(runtime, w, r, err, module.Pattern)
		return true
	}

	if err := runtime.PatchLive(w, r, module.SelectorID, module.Render(view)); err != nil {
		runtime.RespondServerError(w, fmt.Errorf("patch route %q: %w", module.Pattern, err))
	}
	return true
}

func handleLoadError[C interface{}](
	runtime RuntimeContext[C],
	w http.ResponseWriter,
	r *http.Request,
	err error,
	routePattern string,
) {
	if runtime.IsNotFound(err) {
		runtime.RespondNotFound(w, r)
		return
	}

	runtime.RespondServerError(w, fmt.Errorf("load route %q: %w", routePattern, err))
}
