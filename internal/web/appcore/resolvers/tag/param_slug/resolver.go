package param_slug

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
	params Params,
) (PageView, error) {
	return appcore.LoadTagPage(ctx, appCtx, r, framework.SlugParams{Slug: params.Slug})
}
