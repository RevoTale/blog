package resolvers

import (
	"context"
	"net/http"

	"blog/web/seo"
	"blog/web/view"
	"github.com/RevoTale/no-js/framework"
	"github.com/RevoTale/no-js/framework/metagen"
)

func (Resolver) MetaGenTagParamSlugPage(
	meta framework.MetaContext[*runtime.Context],
	params TagParamSlugParams,
) (metagen.Metadata, error) {
	return seo.MetaGenTagPage(meta, params.Slug)
}

func (Resolver) ResolveTagParamSlugPage(
	ctx context.Context,
	appCtx *runtime.Context,
	r *http.Request,
	params TagParamSlugParams,
) (runtime.NotesPageView, error) {
	return runtime.LoadTagPage(ctx, appCtx, r, framework.SlugParams{Slug: params.Slug})
}
