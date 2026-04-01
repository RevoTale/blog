package resolvers

import (
	"context"
	"net/http"

	"blog/web/seo"
	"blog/web/view"
	"github.com/RevoTale/no-js/framework"
	"github.com/RevoTale/no-js/framework/metagen"
)

func (Resolver) MetaGenAuthorParamSlugLayout(
	context.Context,
	*runtime.Context,
	framework.MetadataContext,
	AuthorParamSlugParams,
) (metagen.Metadata, error) {
	return metagen.Metadata{}, nil
}

func (Resolver) MetaGenAuthorParamSlugPage(
	ctx context.Context,
	appCtx *runtime.Context,
	meta framework.MetadataContext,
	params AuthorParamSlugParams,
) (metagen.Metadata, error) {
	return seo.MetaGenAuthorPage(ctx, appCtx, meta, params.Slug)
}

func (Resolver) ResolveAuthorParamSlugPage(
	ctx context.Context,
	appCtx *runtime.Context,
	r *http.Request,
	params AuthorParamSlugParams,
) (runtime.AuthorPageView, error) {
	return runtime.LoadAuthorPage(ctx, appCtx, r, framework.SlugParams{Slug: params.Slug})
}
