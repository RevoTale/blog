package resolvers

import (
	"context"
	"net/http"

	"blog/web/seo"
	"blog/web/view"
	"github.com/RevoTale/no-js/framework"
	"github.com/RevoTale/no-js/framework/metagen"
)

func (Resolver) MetaGenRootLayout(meta framework.MetaContext[*runtime.Context]) (metagen.Metadata, error) {
	_ = meta
	return metagen.Metadata{
		DangerRawHead: []string{runtime.ChromaStyleTag()},
	}, nil
}

func (Resolver) MetaGenRootPage(
	meta framework.MetaContext[*runtime.Context],
	_ RootParams,
) (metagen.Metadata, error) {
	return seo.MetaGenRootPage(meta)
}

func (Resolver) ResolveRootPage(
	ctx context.Context,
	appCtx *runtime.Context,
	r *http.Request,
	_ RootParams,
) (runtime.NotesPageView, error) {
	return runtime.LoadNotesPage(ctx, appCtx, r, framework.EmptyParams{})
}
