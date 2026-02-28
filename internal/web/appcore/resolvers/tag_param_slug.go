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
