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

	RenderPage func(r *http.Request, w http.ResponseWriter, component templ.Component) error
	PatchLive  func(w http.ResponseWriter, r *http.Request, selectorID string, component templ.Component) error

	IsNotFoundError   func(err error) bool
	HandleNotFound    func(w http.ResponseWriter, r *http.Request, notFoundContext framework.NotFoundContext)
	HandleBadRequest  func(w http.ResponseWriter, message string)
	HandleServerError func(w http.ResponseWriter, err error)
}

type Engine[C interface{}] struct {
	appContext C
	handlers   []framework.RouteHandler[C]

	renderPage func(r *http.Request, w http.ResponseWriter, component templ.Component) error
	patchLive  func(w http.ResponseWriter, r *http.Request, selectorID string, component templ.Component) error

	isNotFound  func(err error) bool
	notFound    func(w http.ResponseWriter, r *http.Request, notFoundContext framework.NotFoundContext)
	badRequest  func(w http.ResponseWriter, message string)
	serverError func(w http.ResponseWriter, err error)
}

func New[C interface{}](cfg Config[C]) (*Engine[C], error) {
	if cfg.RenderPage == nil {
		return nil, errors.New("render page callback is required")
	}
	if cfg.PatchLive == nil {
		return nil, errors.New("patch live callback is required")
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

	badRequest := cfg.HandleBadRequest
	if badRequest == nil {
		badRequest = func(w http.ResponseWriter, message string) {
			http.Error(w, message, http.StatusBadRequest)
		}
	}

	serverError := cfg.HandleServerError
	if serverError == nil {
		serverError = func(w http.ResponseWriter, _ error) {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}

	return &Engine[C]{
		appContext:  cfg.AppContext,
		handlers:    cfg.Handlers,
		renderPage:  cfg.RenderPage,
		patchLive:   cfg.PatchLive,
		isNotFound:  isNotFound,
		notFound:    notFound,
		badRequest:  badRequest,
		serverError: serverError,
	}, nil
}

func (engine *Engine[C]) ServeRoute(w http.ResponseWriter, r *http.Request) bool {
	for _, handler := range engine.handlers {
		if handler.TryServeLive(engine, w, r) {
			return true
		}
	}

	for _, handler := range engine.handlers {
		if handler.TryServePage(engine, w, r) {
			return true
		}
	}

	return false
}

func (engine *Engine[C]) AppContext() C {
	return engine.appContext
}

func (engine *Engine[C]) RenderPage(
	r *http.Request,
	w http.ResponseWriter,
	component templ.Component,
) error {
	return engine.renderPage(r, w, component)
}

func (engine *Engine[C]) PatchLive(
	w http.ResponseWriter,
	r *http.Request,
	selectorID string,
	component templ.Component,
) error {
	return engine.patchLive(w, r, selectorID, component)
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

func (engine *Engine[C]) RespondBadRequest(w http.ResponseWriter, message string) {
	engine.badRequest(w, message)
}

func (engine *Engine[C]) RespondServerError(w http.ResponseWriter, err error) {
	engine.serverError(w, err)
}
