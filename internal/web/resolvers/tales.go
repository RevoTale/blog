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

func (Resolver) ParseTalesLiveState(r *http.Request) (appcore.NotesSignalState, error) {
	return appcore.ParseNotesLiveState(r)
}

func (Resolver) ResolveTalesLive(
	ctx context.Context,
	appCtx *appcore.Context,
	r *http.Request,
	_ TalesParams,
	state appcore.NotesSignalState,
) (appcore.NotesPageView, error) {
	return appcore.LoadNotesTalesLivePage(ctx, appCtx, r, framework.EmptyParams{}, state)
}
