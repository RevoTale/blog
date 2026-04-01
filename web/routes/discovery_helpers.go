package routes

import (
	"net/http"
	"strings"

	runtimeview "blog/web/view"
	"github.com/RevoTale/no-js/framework"
)

func resolveDiscoveryRootURL(
	runtime framework.RuntimeContext[*runtimeview.Context],
	r *http.Request,
) string {
	appCtx := runtime.AppContext()
	resolver := appCtx.SiteResolver()
	if resolver == nil {
		
		return strings.TrimSpace(appCtx.RootURL())
	}

	if resolved := strings.TrimSpace(resolver.Resolve(r)); resolved != "" {
		return resolved
	}

	return strings.TrimSpace(resolver.CanonicalURL())
}
