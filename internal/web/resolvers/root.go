package resolvers

import (
	"context"
	"net/http"

	"blog/framework"
	"blog/framework/metagen"
	"blog/internal/web/appcore"
	"blog/internal/web/seo"
)

func (Resolver) MetaGenRootLayout(
	context.Context,
	*appcore.Context,
	*http.Request,
) (metagen.Metadata, error) {
	return metagen.Metadata{
		DangerRawHead: []string{appcore.ChromaStyleTag()},
	}, nil
}

func (Resolver) MetaGenRootPage(
	ctx context.Context,
	appCtx *appcore.Context,
	r *http.Request,
	_ RootParams,
) (metagen.Metadata, error) {
	return seo.MetaGenRootPage(ctx, appCtx, r)
}

func (Resolver) ResolveRootPage(
	ctx context.Context,
	appCtx *appcore.Context,
	r *http.Request,
	_ RootParams,
) (appcore.NotesPageView, error) {
	return appcore.LoadNotesPage(ctx, appCtx, r, framework.EmptyParams{})
}
