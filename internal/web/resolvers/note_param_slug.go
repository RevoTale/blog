package resolvers

import (
	"context"
	"net/http"

	"blog/internal/web/appcore"
	"blog/internal/web/seo"
	"github.com/RevoTale/no-js/framework"
	"github.com/RevoTale/no-js/framework/metagen"
)

func (Resolver) MetaGenNoteParamSlugPage(
	ctx context.Context,
	appCtx *appcore.Context,
	r *http.Request,
	params NoteParamSlugParams,
) (metagen.Metadata, error) {
	return seo.MetaGenNotePage(ctx, appCtx, r, params.Slug)
}

func (Resolver) ResolveNoteParamSlugPage(
	ctx context.Context,
	appCtx *appcore.Context,
	r *http.Request,
	params NoteParamSlugParams,
) (appcore.NotePageView, error) {
	return appcore.LoadNotePage(ctx, appCtx, r, framework.SlugParams{Slug: params.Slug})
}
