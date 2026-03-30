package resolvers

import (
	"context"
	"net/http"

	"blog/web/seo"
	"blog/web/view"
	"github.com/RevoTale/no-js/framework"
	"github.com/RevoTale/no-js/framework/metagen"
)

func (Resolver) MetaGenChannelsPage(
	ctx context.Context,
	appCtx *runtime.Context,
	r *http.Request,
	_ ChannelsParams,
) (metagen.Metadata, error) {
	return seo.MetaGenChannelsPage(ctx, appCtx, r)
}

func (Resolver) ResolveChannelsPage(
	ctx context.Context,
	appCtx *runtime.Context,
	r *http.Request,
	_ ChannelsParams,
) (runtime.NotesPageView, error) {
	return runtime.LoadChannelsPage(ctx, appCtx, r, framework.EmptyParams{})
}
