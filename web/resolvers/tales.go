package resolvers

import (
	"context"
	"net/http"

	"blog/web/seo"
	"blog/web/view"
	"github.com/RevoTale/no-js/framework"
	"github.com/RevoTale/no-js/framework/metagen"
)

func (Resolver) MetaGenTalesPage(
	ctx context.Context,
	appCtx *runtime.Context,
	meta framework.MetadataContext,
	_ TalesParams,
) (metagen.Metadata, error) {
	return seo.MetaGenTalesPage(ctx, appCtx, meta)
}

func (Resolver) ResolveTalesPage(
	ctx context.Context,
	appCtx *runtime.Context,
	r *http.Request,
	_ TalesParams,
) (runtime.NotesPageView, error) {
	return runtime.LoadNotesTalesPage(ctx, appCtx, r, framework.EmptyParams{})
}
