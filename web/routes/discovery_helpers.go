package routes

import (
	"net/http"

	runtimeview "blog/web/view"
	"github.com/RevoTale/no-js/framework"
)

func resolveDiscoveryRootURL(
	runtime framework.RuntimeContext[*runtimeview.Context],
	r *http.Request,
) string {
	appCtx := runtime.AppContext()
	return appCtx.ResolveRootURL(r)
}
