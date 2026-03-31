package routes

import (
	"net/http"

	blogdiscovery "blog/internal/discovery"
	runtimeview "blog/web/view"
	"github.com/RevoTale/no-js/framework"
	frameworkdiscovery "github.com/RevoTale/no-js/framework/discovery"
)

func Robots(
	runtime framework.RuntimeContext[*runtimeview.Context],
	r *http.Request,
) (frameworkdiscovery.Robots, error) {
	return blogdiscovery.BuildRobots(resolveDiscoveryRootURL(runtime, r)), nil
}
