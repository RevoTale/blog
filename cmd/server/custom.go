package main

import (
	"fmt"
	"net/http"
	"strings"

	"blog/internal/config"
	"blog/internal/discovery"
	"blog/internal/notes"
	"blog/internal/site"
	runtime "blog/web/view"
	"github.com/RevoTale/no-js/framework/httpserver"
	frameworki18n "github.com/RevoTale/no-js/framework/i18n"
)

const immutableStaticCachePolicy = "public, max-age=31536000, immutable"
const defaultStaticManifestPath = "web/assets-build/manifest.json"
const defaultStaticURLPrefix = "/_assets/"
const defaultPublicDir = "web/public"

func newCustomConfig(
	cfg config.Config,
	siteResolver site.Resolver,
	noteService *notes.Service,
	i18nConfig frameworki18n.Config,
	logServerError func(error),
) httpserver.CustomConfig {
	cachePolicies := httpserver.DefaultCachePolicies()
	if strings.TrimSpace(cfg.CacheLiveNavigation) != "" {
		cachePolicies.LiveNavigation = strings.TrimSpace(cfg.CacheLiveNavigation)
	}
	cachePolicies.Static = immutableStaticCachePolicy

	custom := httpserver.CustomConfig{
		ExtraRoutes:         buildExtraRoutesMount(siteResolver, i18nConfig, noteService, cachePolicies.HTML),
		MainMiddlewares:     []func(http.Handler) http.Handler{runtime.WithCanonicalNotesRedirects},
		CachePolicies:       cachePolicies,
		LogServerError:      logServerError,
		EnableResolverDebug: cfg.EnableResolverDebug,
	}

	if (strings.TrimSpace(cfg.StaticManifestPath) != "" && cfg.StaticManifestPath != defaultStaticManifestPath) ||
		(strings.TrimSpace(cfg.StaticURLPrefix) != "" && cfg.StaticURLPrefix != defaultStaticURLPrefix) {
		custom.StaticAssets = &httpserver.StaticAssetsConfig{
			ManifestPath: cfg.StaticManifestPath,
			URLPrefix:    cfg.StaticURLPrefix,
		}
	}

	if (strings.TrimSpace(cfg.PublicDir) != "" && cfg.PublicDir != defaultPublicDir) ||
		strings.TrimSpace(cfg.CachePublicFiles) != "" {
		custom.PublicFiles = &httpserver.PublicFilesConfig{
			Dir:         cfg.PublicDir,
			CachePolicy: strings.TrimSpace(cfg.CachePublicFiles),
		}
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
