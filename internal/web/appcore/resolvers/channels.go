package resolvers

import (
	"context"
	"net/http"

	"blog/framework"
	"blog/internal/web/appcore"
)

func (Resolver) ResolveChannelsPage(
	ctx context.Context,
	appCtx *appcore.Context,
	r *http.Request,
	_ ChannelsParams,
) (appcore.NotesPageView, error) {
	return appcore.LoadChannelsPage(ctx, appCtx, r, framework.EmptyParams{})
}
