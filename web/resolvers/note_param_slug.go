package resolvers

import (
	"context"
	"net/http"

	"blog/web/seo"
	"blog/web/view"
	"github.com/RevoTale/no-js/framework"
	"github.com/RevoTale/no-js/framework/metagen"
)

func (Resolver) MetaGenNoteParamSlugPage(
	meta framework.MetaContext[*runtime.Context],
	params NoteParamSlugParams,
) (metagen.Metadata, error) {
	return seo.MetaGenNotePage(meta, params.Slug)
}

func (Resolver) ResolveNoteParamSlugPage(
	ctx context.Context,
	appCtx *runtime.Context,
	r *http.Request,
	params NoteParamSlugParams,
) (runtime.NotePageView, error) {
	return runtime.LoadNotePage(ctx, appCtx, r, framework.SlugParams{Slug: params.Slug})
}
