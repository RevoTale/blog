package param_slug

import (
	"context"
	"net/http"

	"blog/internal/web/appcore"
)

type PageView = appcore.NotesPageView

type Params struct {
	Slug string
}

type RouteResolver interface {
	ResolvePage(ctx context.Context, appCtx *appcore.Context, r *http.Request, params Params) (PageView, error)
}

var _ RouteResolver = (*Resolver)(nil)
