package gen

import (
	runtime "blog/web/view"
	"github.com/RevoTale/no-js/framework/httpserver"
	frameworki18n "github.com/RevoTale/no-js/framework/i18n"
)

func Bundle(appContext *runtime.Context) httpserver.AppBundle[*runtime.Context] {
	var i18nConfig *frameworki18n.Config
	if appContext != nil {
		cfg := appContext.I18nConfig()
		i18nConfig = &cfg
	}

	return httpserver.AppBundle[*runtime.Context]{
		Context:                       appContext,
		Handlers:                      Handlers(NewRouteResolvers()),
		I18n:                          i18nConfig,
		NotFoundPage:                  NotFoundPage,
		OnStaticAssetBasePathResolved: runtime.SetStaticAssetBasePath,
	}
}
