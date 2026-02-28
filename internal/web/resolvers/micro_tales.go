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

func (Resolver) ParseMicroTalesLiveState(r *http.Request) (appcore.NotesSignalState, error) {
	return appcore.ParseNotesLiveState(r)
}

func (Resolver) ResolveMicroTalesLive(
	ctx context.Context,
	appCtx *appcore.Context,
	r *http.Request,
	_ MicroTalesParams,
	state appcore.NotesSignalState,
) (appcore.NotesPageView, error) {
	return appcore.LoadNotesMicroTalesLivePage(ctx, appCtx, r, framework.EmptyParams{}, state)
}
