package routes

import (
	"fmt"
	"net/http"

	blogdiscovery "blog/internal/discovery"
	"blog/internal/notes"
	runtimeview "blog/web/view"
	"github.com/RevoTale/no-js/framework"
	frameworkdiscovery "github.com/RevoTale/no-js/framework/discovery"
)

func Feed(
	runtime framework.RuntimeContext[*runtimeview.Context],
	r *http.Request,
) (frameworkdiscovery.FeedDocument, error) {
	appCtx := runtime.AppContext()
	service := appCtx.Notes()
	if service == nil {
		return frameworkdiscovery.FeedDocument{}, fmt.Errorf("notes service unavailable")
	}

	locale := appCtx.LocaleFromRequest(r.URL.Query().Get("locale"))
	listResult, err := service.ListNotes(
		r.Context(),
		locale,
		blogdiscovery.FeedListFilterFromQuery(r.URL.Query()),
		notes.ListOptions{},
	)
	if err != nil {
		return frameworkdiscovery.FeedDocument{}, err
	}

	return blogdiscovery.BuildFeedDocument(
		resolveDiscoveryRootURL(runtime, r),
		resolveDiscoveryI18nConfig(runtime),
		locale,
		listResult.Notes,
	), nil
}
