package resolvers

import (
	"context"
	"net/http"

	"blog/framework"
	"blog/internal/web/appcore"
)

func (Resolver) ResolveNoteParamSlugPage(
	ctx context.Context,
	appCtx *appcore.Context,
	r *http.Request,
	params NoteParamSlugParams,
) (appcore.NotePageView, error) {
	return appcore.LoadNotePage(ctx, appCtx, r, framework.SlugParams{Slug: params.Slug})
}
