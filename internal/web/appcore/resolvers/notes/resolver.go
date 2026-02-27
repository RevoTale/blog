package notes

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

func (Resolver) ParseLiveState(r *http.Request) (LiveState, error) {
	return appcore.ParseNotesLiveState(r)
}

func (Resolver) ResolveLive(
	ctx context.Context,
	appCtx *appcore.Context,
	r *http.Request,
	_ Params,
	state LiveState,
) (PageView, error) {
	return appcore.LoadNotesLivePage(ctx, appCtx, r, framework.EmptyParams{}, state)
}
