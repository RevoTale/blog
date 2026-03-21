package resolvers

import (
	"context"
	"net/http"

	"blog/internal/web/appcore"
	"blog/internal/web/seo"
	"github.com/RevoTale/no-js/framework"
	"github.com/RevoTale/no-js/framework/metagen"
)

func (Resolver) MetaGenTalesPage(
	ctx context.Context,
	appCtx *appcore.Context,
	r *http.Request,
	_ TalesParams,
) (metagen.Metadata, error) {
	return seo.MetaGenTalesPage(ctx, appCtx, r)
}

func (Resolver) ResolveTalesPage(
	ctx context.Context,
	appCtx *appcore.Context,
	r *http.Request,
	_ TalesParams,
) (appcore.NotesPageView, error) {
	return appcore.LoadNotesTalesPage(ctx, appCtx, r, framework.EmptyParams{})
}
