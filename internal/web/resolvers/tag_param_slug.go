package resolvers

import (
	"context"
	"net/http"

	"blog/internal/web/runtime"
	"blog/internal/web/seo"
	"github.com/RevoTale/no-js/framework"
	"github.com/RevoTale/no-js/framework/metagen"
)

func (Resolver) MetaGenTagParamSlugPage(
	ctx context.Context,
	appCtx *runtime.Context,
	r *http.Request,
	params TagParamSlugParams,
) (metagen.Metadata, error) {
	return seo.MetaGenTagPage(ctx, appCtx, r, params.Slug)
}

func (Resolver) ResolveTagParamSlugPage(
	ctx context.Context,
	appCtx *runtime.Context,
	r *http.Request,
	params TagParamSlugParams,
) (runtime.NotesPageView, error) {
	return runtime.LoadTagPage(ctx, appCtx, r, framework.SlugParams{Slug: params.Slug})
}
