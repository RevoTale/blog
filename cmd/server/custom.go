package main

import (
	"fmt"
	"net/http"

	"blog/internal/config"
	"blog/internal/discovery"
	"blog/internal/notes"
	"blog/internal/site"
	runtime "blog/web/view"
	"github.com/RevoTale/no-js/framework/httpserver"
	frameworki18n "github.com/RevoTale/no-js/framework/i18n"
)

const immutableStaticCachePolicy = "public, max-age=31536000, immutable"
const blogLiveNavigationCachePolicy = "public, max-age=3600, s-maxage=3600"

func newCustomConfig(
	cfg config.Config,
	siteResolver site.Resolver,
	noteService *notes.Service,
	i18nConfig frameworki18n.Config,
	logServerError func(error),
) httpserver.CustomConfig {
	cachePolicies := httpserver.DefaultCachePolicies()
	cachePolicies.Static = immutableStaticCachePolicy
	cachePolicies.LiveNavigation = blogLiveNavigationCachePolicy

	custom := httpserver.CustomConfig{
		ExtraRoutes:         buildExtraRoutesMount(siteResolver, i18nConfig, noteService, cachePolicies.HTML),
		MainMiddlewares:     []func(http.Handler) http.Handler{runtime.WithCanonicalNotesRedirects},
		CachePolicies:       cachePolicies,
		LogServerError:      logServerError,
		EnableResolverDebug: cfg.EnableResolverDebug,
	}

	return custom
}

func buildExtraRoutesMount(
	siteResolver site.Resolver,
	i18nConfig frameworki18n.Config,
	noteService *notes.Service,
	htmlCachePolicy string,
) func(*http.ServeMux) error {
	return func(mux *http.ServeMux) error {
		if err := discovery.MountFeedAndSitemapEndpoints(mux, discovery.FeedAndSitemapConfig{
			RootURL:    siteResolver.CanonicalURL(),
			I18nConfig: i18nConfig,
			Notes:      noteService,
		}); err != nil {
			return fmt.Errorf("mount feed and sitemap endpoints: %w", err)
		}
		if err := discovery.MountRobotsEndpoint(mux, siteResolver.CanonicalURL(), htmlCachePolicy); err != nil {
			return fmt.Errorf("mount robots endpoint: %w", err)
		}

		return nil
	}
}
