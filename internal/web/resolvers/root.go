package resolvers

import (
	"context"
	"net/http"

	"blog/framework"
	"blog/internal/web/appcore"
)

func (Resolver) ResolveRootPage(
	ctx context.Context,
	appCtx *appcore.Context,
	r *http.Request,
	_ RootParams,
) (appcore.NotesPageView, error) {
	return appcore.LoadNotesPage(ctx, appCtx, r, framework.EmptyParams{})
}

func (Resolver) ParseRootLiveState(r *http.Request) (appcore.NotesSignalState, error) {
	return appcore.ParseNotesLiveState(r)
}

func (Resolver) ResolveRootLive(
	ctx context.Context,
	appCtx *appcore.Context,
	r *http.Request,
	_ RootParams,
	state appcore.NotesSignalState,
) (appcore.NotesPageView, error) {
	return appcore.LoadNotesLivePage(ctx, appCtx, r, framework.EmptyParams{}, state)
}
