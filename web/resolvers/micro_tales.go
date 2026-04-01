package resolvers

import (
	"context"
	"net/http"

	"blog/web/seo"
	"blog/web/view"
	"github.com/RevoTale/no-js/framework"
	"github.com/RevoTale/no-js/framework/metagen"
)

func (Resolver) MetaGenMicroTalesPage(
	ctx context.Context,
	appCtx *runtime.Context,
	meta framework.MetadataContext,
	_ MicroTalesParams,
) (metagen.Metadata, error) {
	return seo.MetaGenMicroTalesPage(ctx, appCtx, meta)
}

func (Resolver) ResolveMicroTalesPage(
	ctx context.Context,
	appCtx *runtime.Context,
	r *http.Request,
	_ MicroTalesParams,
) (runtime.NotesPageView, error) {
	return runtime.LoadNotesMicroTalesPage(ctx, appCtx, r, framework.EmptyParams{})
}
