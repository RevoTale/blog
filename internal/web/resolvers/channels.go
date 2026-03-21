package resolvers

import (
	"context"
	"net/http"

	"blog/internal/web/appcore"
	"blog/internal/web/seo"
	"github.com/RevoTale/no-js/framework"
	"github.com/RevoTale/no-js/framework/metagen"
)

func (Resolver) MetaGenChannelsPage(
	ctx context.Context,
	appCtx *appcore.Context,
	r *http.Request,
	_ ChannelsParams,
) (metagen.Metadata, error) {
	return seo.MetaGenChannelsPage(ctx, appCtx, r)
}

func (Resolver) ResolveChannelsPage(
	ctx context.Context,
	appCtx *appcore.Context,
	r *http.Request,
	_ ChannelsParams,
) (appcore.NotesPageView, error) {
	return appcore.LoadChannelsPage(ctx, appCtx, r, framework.EmptyParams{})
}
