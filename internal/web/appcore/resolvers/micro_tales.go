package resolvers

import (
	"context"
	"net/http"

	"blog/framework"
	"blog/internal/web/appcore"
)

func (Resolver) ResolveMicroTalesPage(
	ctx context.Context,
	appCtx *appcore.Context,
	r *http.Request,
	_ MicroTalesParams,
) (appcore.NotesPageView, error) {
	return appcore.LoadNotesMicroTalesPage(ctx, appCtx, r, framework.EmptyParams{})
}
