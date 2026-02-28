package resolvers

import (
	"context"
	"net/http"

	"blog/framework"
	"blog/internal/web/appcore"
)

func (Resolver) ResolveTagParamSlugPage(
	ctx context.Context,
	appCtx *appcore.Context,
	r *http.Request,
	params TagParamSlugParams,
) (appcore.NotesPageView, error) {
	return appcore.LoadTagPage(ctx, appCtx, r, framework.SlugParams{Slug: params.Slug})
}

func (Resolver) ParseTagParamSlugLiveState(r *http.Request) (appcore.NotesSignalState, error) {
	return appcore.ParseNotesLiveState(r)
}

func (Resolver) ResolveTagParamSlugLive(
	ctx context.Context,
	appCtx *appcore.Context,
	r *http.Request,
	params TagParamSlugParams,
	state appcore.NotesSignalState,
) (appcore.NotesPageView, error) {
	return appcore.LoadTagLivePage(ctx, appCtx, r, framework.SlugParams{Slug: params.Slug}, state)
}
