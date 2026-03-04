package framework

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"
	goruntime "runtime"
	"strings"
	"sync"
	"time"

	frameworki18n "blog/framework/i18n"
	"blog/framework/metagen"
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

type PageMetaGen[C interface{}, P interface{}] func(
	ctx context.Context,
	appCtx C,
	r *http.Request,
	params P,
) (metagen.Metadata, error)

type PageRenderer[VM interface{}] func(view VM) templ.Component

type LayoutRenderer[VM interface{}] func(meta metagen.Metadata, view VM, child templ.Component) templ.Component

type PageModule[C interface{}, P interface{}, VM interface{}] struct {
	Pattern           string
	ParseParams       ParamsParser[P]
	MetaGen           PageMetaGen[C, P]
	MetaGenName       string
	MetaGenChain      []PageMetaGen[C, P]
	MetaGenChainNames []string
	Load              PageLoader[C, P, VM]
	LoadName          string
	Render            PageRenderer[VM]
	Layouts           []LayoutRenderer[VM]
	RootLayout        func(meta metagen.Metadata, locale string, child templ.Component) templ.Component
	ErrorPage         func(locale string, path string) templ.Component
}

type RuntimeContext[C interface{}] interface {
	AppContext() C
	IsPartialRequest(r *http.Request) bool
	RenderPage(r *http.Request, w http.ResponseWriter, component templ.Component, meta metagen.Metadata) error
	IsNotFound(err error) bool
	RespondNotFound(w http.ResponseWriter, r *http.Request, notFoundContext NotFoundContext)
	RespondServerError(w http.ResponseWriter, err error)
	LogServerError(err error)
	LogResolverTiming(event ResolverTiming)
}

type ResolverStage string

const (
	ResolverStageMetaGen ResolverStage = "meta_gen"
	ResolverStageLoad    ResolverStage = "load"
)

type ResolverTiming struct {
	RoutePattern string
	Stage        ResolverStage
	Method       string
	Duration     time.Duration
	Err          error
}

type NotFoundSource string

const (
	NotFoundSourcePageLoad       NotFoundSource = "page_load"
	NotFoundSourceMetaGen        NotFoundSource = "meta_gen"
	NotFoundSourceUnmatchedRoute NotFoundSource = "unmatched_route"
)

type NotFoundContext struct {
	RequestPath         string
	MatchedRoutePattern string
	Locale              string
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
	meta metagen.Metadata,
	view VM,
	child templ.Component,
) templ.Component {
	wrapped := child
	for idx := len(layouts) - 1; idx >= 0; idx-- {
		wrapped = layouts[idx](meta, view, wrapped)
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

	type metadataResult struct {
		meta metagen.Metadata
		err  error
	}
	type pageLoadResult struct {
		view VM
		err  error
	}

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	appCtx := runtime.AppContext()

	metaCh := make(chan metadataResult, 1)
	go func() {
		meta, err := resolveMetadata(runtime, ctx, appCtx, r, params, module)
		metaCh <- metadataResult{meta: meta, err: err}
	}()

	loadCh := make(chan pageLoadResult, 1)
	go func() {
		startedAt := time.Now()
		view, err := module.Load(ctx, appCtx, r, params)
		runtime.LogResolverTiming(ResolverTiming{
			RoutePattern: module.Pattern,
			Stage:        ResolverStageLoad,
			Method:       loadMethodName(module),
			Duration:     time.Since(startedAt),
			Err:          err,
		})
		loadCh <- pageLoadResult{view: view, err: err}
	}()

	metaResult := <-metaCh
	if metaResult.err != nil {
		handleModuleError(runtime, w, r, metaResult.err, module.Pattern, NotFoundSourceMetaGen, "meta")
		return true
	}
	meta := metagen.Normalize(metaResult.meta)

	var loadOnce sync.Once
	var loadResult pageLoadResult
	awaitLoad := func() pageLoadResult {
		loadOnce.Do(func() {
			loadResult = <-loadCh
		})
		return loadResult
	}

	partial := runtime.IsPartialRequest(r)
	if partial || module.RootLayout == nil {
		result := awaitLoad()
		if result.err != nil {
			handleModuleError(runtime, w, r, result.err, module.Pattern, NotFoundSourcePageLoad, "load")
			return true
		}

		component := module.Render(result.view)
		if !partial {
			component = applyLayouts(module.Layouts, meta, result.view, component)
		}
		if err := runtime.RenderPage(r, w, component, meta); err != nil {
			runtime.RespondServerError(w, fmt.Errorf("render route %q: %w", module.Pattern, err))
		}
		return true
	}

	locale := frameworki18n.LocaleFromContext(r.Context())
	streamedBody := templ.ComponentFunc(func(renderCtx context.Context, writer io.Writer) error {
		result := awaitLoad()
		if result.err != nil {
			runtime.LogServerError(fmt.Errorf("load route %q after stream start: %w", module.Pattern, result.err))
			if module.ErrorPage == nil {
				return nil
			}
			errorComponent := module.ErrorPage(locale, r.URL.Path)
			if errorComponent == nil {
				return nil
			}
			return errorComponent.Render(renderCtx, writer)
		}

		component := module.Render(result.view)
		component = applyLayouts(module.Layouts, meta, result.view, component)
		return component.Render(renderCtx, writer)
	})

	component := module.RootLayout(meta, locale, streamedBody)
	if component == nil {
		component = streamedBody
	}
	if err := runtime.RenderPage(r, w, component, meta); err != nil {
		runtime.RespondServerError(w, fmt.Errorf("render route %q: %w", module.Pattern, err))
	}
	return true
}

func resolveMetadata[C interface{}, P interface{}, VM interface{}](
	runtime RuntimeContext[C],
	ctx context.Context,
	appCtx C,
	r *http.Request,
	params P,
	module PageModule[C, P, VM],
) (metagen.Metadata, error) {
	chain := module.MetaGenChain
	if len(chain) == 0 && module.MetaGen != nil {
		chain = append(chain, module.MetaGen)
	}
	if len(chain) == 0 {
		return metagen.Metadata{}, nil
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := make([]metagen.Metadata, len(chain))
	errs := make([]error, len(chain))

	var wg sync.WaitGroup
	for idx, fn := range chain {
		wg.Add(1)
		go func(i int, run PageMetaGen[C, P]) {
			defer wg.Done()
			if run == nil {
				return
			}
			startedAt := time.Now()
			meta, err := run(ctx, appCtx, r, params)
			runtime.LogResolverTiming(ResolverTiming{
				RoutePattern: module.Pattern,
				Stage:        ResolverStageMetaGen,
				Method:       metaGenMethodName(module, i, run),
				Duration:     time.Since(startedAt),
				Err:          err,
			})
			if err != nil {
				errs[i] = err
				cancel()
				return
			}
			results[i] = meta
		}(idx, fn)
	}
	wg.Wait()

	for _, err := range errs {
		if err != nil {
			return metagen.Metadata{}, err
		}
	}
	return metagen.MergeAll(results...), nil
}

func handleModuleError[C interface{}](
	runtime RuntimeContext[C],
	w http.ResponseWriter,
	r *http.Request,
	err error,
	routePattern string,
	source NotFoundSource,
	stage string,
) {
	if runtime.IsNotFound(err) {
		runtime.RespondNotFound(w, r, NotFoundContext{
			RequestPath:         r.URL.Path,
			MatchedRoutePattern: routePattern,
			Locale:              frameworki18n.LocaleFromContext(r.Context()),
			Source:              source,
		})
		return
	}

	runtime.RespondServerError(w, fmt.Errorf("%s route %q: %w", stage, routePattern, err))
}

func loadMethodName[C interface{}, P interface{}, VM interface{}](module PageModule[C, P, VM]) string {
	if name := strings.TrimSpace(module.LoadName); name != "" {
		return name
	}
	return resolverFuncName(module.Load)
}

func metaGenMethodName[C interface{}, P interface{}, VM interface{}](
	module PageModule[C, P, VM],
	index int,
	run PageMetaGen[C, P],
) string {
	if index >= 0 && index < len(module.MetaGenChainNames) {
		if name := strings.TrimSpace(module.MetaGenChainNames[index]); name != "" {
			return name
		}
	}
	if len(module.MetaGenChain) == 0 {
		if name := strings.TrimSpace(module.MetaGenName); name != "" {
			return name
		}
	}
	return resolverFuncName(run)
}

func resolverFuncName(fn interface{}) string {
	if fn == nil {
		return "unknown"
	}
	value := reflect.ValueOf(fn)
	if !value.IsValid() || value.Kind() != reflect.Func {
		return "unknown"
	}
	ptr := value.Pointer()
	if ptr == 0 {
		return "unknown"
	}
	resolved := goruntime.FuncForPC(ptr)
	if resolved == nil {
		return "unknown"
	}
	return resolved.Name()
}
