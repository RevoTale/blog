package engine

import (
	"errors"
	"net/http"

	"blog/framework"
	"github.com/a-h/templ"
)

type Config[C interface{}] struct {
	AppContext C
	Handlers   []framework.RouteHandler[C]

	IsPartialRequest func(r *http.Request) bool
	RenderPage       func(r *http.Request, w http.ResponseWriter, component templ.Component) error

	IsNotFoundError   func(err error) bool
	HandleNotFound    func(w http.ResponseWriter, r *http.Request, notFoundContext framework.NotFoundContext)
	HandleServerError func(w http.ResponseWriter, err error)
}

type Engine[C interface{}] struct {
	appContext C
	handlers   []framework.RouteHandler[C]

	isPartialRequest func(r *http.Request) bool
	renderPage       func(r *http.Request, w http.ResponseWriter, component templ.Component) error

	isNotFound  func(err error) bool
	notFound    func(w http.ResponseWriter, r *http.Request, notFoundContext framework.NotFoundContext)
	serverError func(w http.ResponseWriter, err error)
}

func New[C interface{}](cfg Config[C]) (*Engine[C], error) {
	if cfg.RenderPage == nil {
		return nil, errors.New("render page callback is required")
	}

	isNotFound := cfg.IsNotFoundError
	if isNotFound == nil {
		isNotFound = func(error) bool { return false }
	}

	notFound := cfg.HandleNotFound
	if notFound == nil {
		notFound = func(w http.ResponseWriter, r *http.Request, _ framework.NotFoundContext) {
			http.NotFound(w, r)
		}
	}

	isPartialRequest := cfg.IsPartialRequest
	if isPartialRequest == nil {
		isPartialRequest = func(*http.Request) bool { return false }
	}

	serverError := cfg.HandleServerError
	if serverError == nil {
		serverError = func(w http.ResponseWriter, _ error) {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}

	return &Engine[C]{
		appContext:       cfg.AppContext,
		handlers:         cfg.Handlers,
		isPartialRequest: isPartialRequest,
		renderPage:       cfg.RenderPage,
		isNotFound:       isNotFound,
		notFound:         notFound,
		serverError:      serverError,
	}, nil
}

func (engine *Engine[C]) ServeRoute(w http.ResponseWriter, r *http.Request) bool {
	for _, handler := range engine.handlers {
		if handler.TryServe(engine, w, r) {
			return true
		}
	}

	return false
}

func (engine *Engine[C]) AppContext() C {
	return engine.appContext
}

func (engine *Engine[C]) IsPartialRequest(r *http.Request) bool {
	return engine.isPartialRequest(r)
}

func (engine *Engine[C]) RenderPage(
	r *http.Request,
	w http.ResponseWriter,
	component templ.Component,
) error {
	return engine.renderPage(r, w, component)
}

func (engine *Engine[C]) IsNotFound(err error) bool {
	return engine.isNotFound(err)
}

func (engine *Engine[C]) RespondNotFound(
	w http.ResponseWriter,
	r *http.Request,
	notFoundContext framework.NotFoundContext,
) {
	engine.notFound(w, r, notFoundContext)
}

func (engine *Engine[C]) RespondServerError(w http.ResponseWriter, err error) {
	engine.serverError(w, err)
}
