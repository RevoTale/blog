package resolvers

import (
	"context"
	"net/http"

	"blog/framework"
	"blog/internal/web/appcore"
)

func (Resolver) ResolveTalesPage(
	ctx context.Context,
	appCtx *appcore.Context,
	r *http.Request,
	_ TalesParams,
) (appcore.NotesPageView, error) {
	return appcore.LoadNotesTalesPage(ctx, appCtx, r, framework.EmptyParams{})
}
