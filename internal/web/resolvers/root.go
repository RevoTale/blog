package resolvers

import (
	"context"
	"net/http"

	"blog/internal/web/appcore"
	"blog/internal/web/seo"
	"github.com/RevoTale/no-js/framework"
	"github.com/RevoTale/no-js/framework/metagen"
)

func (Resolver) MetaGenRootLayout(
	_ context.Context,
	_ *appcore.Context,
	_ *http.Request,
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
