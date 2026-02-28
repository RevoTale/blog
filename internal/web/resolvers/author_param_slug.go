package resolvers

import (
	"context"
	"net/http"

	"blog/framework"
	"blog/internal/web/appcore"
)

func (Resolver) ResolveAuthorParamSlugPage(
	ctx context.Context,
	appCtx *appcore.Context,
	r *http.Request,
	params AuthorParamSlugParams,
) (appcore.AuthorPageView, error) {
	return appcore.LoadAuthorPage(ctx, appCtx, r, framework.SlugParams{Slug: params.Slug})
}

func (Resolver) ParseAuthorParamSlugLiveState(r *http.Request) (appcore.AuthorSignalState, error) {
	return appcore.ParseAuthorLiveState(r)
}

func (Resolver) ResolveAuthorParamSlugLive(
	ctx context.Context,
	appCtx *appcore.Context,
	r *http.Request,
	params AuthorParamSlugParams,
	state appcore.AuthorSignalState,
) (appcore.AuthorPageView, error) {
	return appcore.LoadAuthorLivePage(ctx, appCtx, r, framework.SlugParams{Slug: params.Slug}, state)
}
