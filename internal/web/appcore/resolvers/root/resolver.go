package root

import (
	"context"
	"net/http"

	"blog/framework"
	"blog/internal/web/appcore"
)

type Resolver struct{}

func (Resolver) ResolvePage(
	ctx context.Context,
	appCtx *appcore.Context,
	r *http.Request,
	_ Params,
) (PageView, error) {
	return appcore.LoadNotesPage(ctx, appCtx, r, framework.EmptyParams{})
}
