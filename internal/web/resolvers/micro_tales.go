package resolvers

import (
	"context"
	"net/http"

	"blog/internal/web/appcore"
	"blog/internal/web/seo"
	"github.com/RevoTale/no-js/framework"
	"github.com/RevoTale/no-js/framework/metagen"
)

func (Resolver) MetaGenMicroTalesPage(
	ctx context.Context,
	appCtx *appcore.Context,
	r *http.Request,
	_ MicroTalesParams,
) (metagen.Metadata, error) {
	return seo.MetaGenMicroTalesPage(ctx, appCtx, r)
}

func (Resolver) ResolveMicroTalesPage(
	ctx context.Context,
	appCtx *appcore.Context,
	r *http.Request,
	_ MicroTalesParams,
) (appcore.NotesPageView, error) {
	return appcore.LoadNotesMicroTalesPage(ctx, appCtx, r, framework.EmptyParams{})
}
