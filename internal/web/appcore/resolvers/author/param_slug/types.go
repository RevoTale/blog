package param_slug

import (
	"context"
	"net/http"

	"blog/internal/web/appcore"
)

type PageView = appcore.AuthorPageView

type LiveState = appcore.AuthorSignalState

type Params struct {
	Slug string
}

type RouteResolver interface {
	ResolvePage(ctx context.Context, appCtx *appcore.Context, r *http.Request, params Params) (PageView, error)
	ParseLiveState(r *http.Request) (LiveState, error)
	ResolveLive(
		ctx context.Context,
		appCtx *appcore.Context,
		r *http.Request,
		params Params,
		state LiveState,
	) (PageView, error)
}

var _ RouteResolver = (*Resolver)(nil)
