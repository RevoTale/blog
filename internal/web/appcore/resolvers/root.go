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
