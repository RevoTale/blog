package resolvers

import (
	"context"
	"net/http"

	"blog/web/seo"
	"blog/web/view"
	"github.com/RevoTale/no-js/framework"
	"github.com/RevoTale/no-js/framework/metagen"
)

func (Resolver) MetaGenRootLayout(
	_ context.Context,
	_ *runtime.Context,
	_ framework.MetadataContext,
) (metagen.Metadata, error) {
	return metagen.Metadata{
		DangerRawHead: []string{runtime.ChromaStyleTag()},
	}, nil
}

func (Resolver) MetaGenRootPage(
	ctx context.Context,
	appCtx *runtime.Context,
	meta framework.MetadataContext,
	_ RootParams,
) (metagen.Metadata, error) {
	return seo.MetaGenRootPage(ctx, appCtx, meta)
}

func (Resolver) ResolveRootPage(
	ctx context.Context,
	appCtx *runtime.Context,
	r *http.Request,
	_ RootParams,
) (runtime.NotesPageView, error) {
	return runtime.LoadNotesPage(ctx, appCtx, r, framework.EmptyParams{})
}
