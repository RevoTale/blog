package resolvers

import (
	"context"
	"net/http"

	"blog/internal/web/runtime"
	"blog/internal/web/seo"
	"github.com/RevoTale/no-js/framework"
	"github.com/RevoTale/no-js/framework/metagen"
)

func (Resolver) MetaGenRootLayout(
	_ context.Context,
	_ *runtime.Context,
	_ *http.Request,
) (metagen.Metadata, error) {
	return metagen.Metadata{
		DangerRawHead: []string{runtime.ChromaStyleTag()},
	}, nil
}

func (Resolver) MetaGenRootPage(
	ctx context.Context,
	appCtx *runtime.Context,
	r *http.Request,
	_ RootParams,
) (metagen.Metadata, error) {
	return seo.MetaGenRootPage(ctx, appCtx, r)
}

func (Resolver) ResolveRootPage(
	ctx context.Context,
	appCtx *runtime.Context,
	r *http.Request,
	_ RootParams,
) (runtime.NotesPageView, error) {
	return runtime.LoadNotesPage(ctx, appCtx, r, framework.EmptyParams{})
}
