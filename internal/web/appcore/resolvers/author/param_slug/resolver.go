package param_slug

import (
	"context"
	"net/http"

	"blog/framework"
	"blog/internal/web/appcore"
)

type Resolver struct{}

func (Resolver) ResolvePage(
	ctx context.Context,
	appCtx *appcore.Context,
	r *http.Request,
	params Params,
) (PageView, error) {
	return appcore.LoadAuthorPage(ctx, appCtx, r, framework.SlugParams{Slug: params.Slug})
}

func (Resolver) ParseLiveState(r *http.Request) (LiveState, error) {
	return appcore.ParseAuthorLiveState(r)
}

func (Resolver) ResolveLive(
	ctx context.Context,
	appCtx *appcore.Context,
	r *http.Request,
	params Params,
	state LiveState,
) (PageView, error) {
	return appcore.LoadAuthorLivePage(ctx, appCtx, r, framework.SlugParams{Slug: params.Slug}, state)
}
