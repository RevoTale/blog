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
	meta framework.MetaContext[*runtime.Context],
	_ AuthorParamSlugParams,
) (metagen.Metadata, error) {
	_ = meta
	return metagen.Metadata{}, nil
}

func (Resolver) MetaGenAuthorParamSlugPage(
	meta framework.MetaContext[*runtime.Context],
	params AuthorParamSlugParams,
) (metagen.Metadata, error) {
	return seo.MetaGenAuthorPage(meta, params.Slug)
}

func (Resolver) ResolveAuthorParamSlugPage(
	ctx context.Context,
	appCtx *runtime.Context,
	r *http.Request,
	params AuthorParamSlugParams,
) (runtime.AuthorPageView, error) {
	return runtime.LoadAuthorPage(ctx, appCtx, r, framework.SlugParams{Slug: params.Slug})
}
