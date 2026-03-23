package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"blog/internal/config"
	"blog/internal/gql"
	"blog/internal/imageloader"
	"blog/internal/notes"
	"blog/internal/robots"
	"blog/internal/seo"
	webgen "blog/internal/web/gen"
	webi18n "blog/internal/web/i18n"
	"blog/internal/web/runtime"

	"github.com/RevoTale/no-js/framework/httpserver"
	frameworki18n "github.com/RevoTale/no-js/framework/i18n"
)

const immutableStaticCachePolicy = "public, max-age=31536000, immutable"

func main() {
	if err := run(); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}

func run() error {
	cfg := config.Load()
	rootURL, err := validateRootURL(cfg.RootURL)
	if err != nil {
		return err
	}

	i18nConfig, err := frameworki18n.NormalizeConfig(webi18n.Config())
	if err != nil {
		return fmt.Errorf("normalize i18n config: %w", err)
	}
	i18nCatalog, err := webi18n.LoadCatalog()
	if err != nil {
		return fmt.Errorf("load i18n catalog: %w", err)
	}
	imageLoader := imageloader.New(cfg.EnableImageLoader)

	graphqlClient := gql.NewClient(cfg)
	noteService := notes.NewService(
		graphqlClient,
		cfg.PageSize,
		rootURL,
		imageLoader,
	)
	cachePolicies := httpserver.DefaultCachePolicies()
	if strings.TrimSpace(cfg.CacheLiveNavigation) != "" {
		cachePolicies.LiveNavigation = cfg.CacheLiveNavigation
	}
	cachePolicies.Static = immutableStaticCachePolicy

	handler, err := webgen.NewHandler(webgen.ServerConfig{
		AppContext: runtime.NewContext(
			noteService,
			i18nConfig,
			i18nCatalog,
			rootURL,
			cfg.LovelyEyeScriptURL,
			cfg.LovelyEyeSiteID,
		),
		Runtime: runtime.BootstrapConfig{
			LocalizationConfig: i18nConfig,
			ImageLoader:        imageLoader,
			LovelyEyeScriptURL: cfg.LovelyEyeScriptURL,
			LovelyEyeSiteID:    cfg.LovelyEyeSiteID,
		},
		StaticManifestPath:     cfg.StaticManifestPath,
		PublicDir:              cfg.PublicDir,
		PublicFilesCachePolicy: cfg.CachePublicFiles,
		CachePolicies:          cachePolicies,
		LogServerError: func(err error) {
			log.Printf("blog server error: %v", err)
		},
		EnableResolverDebug: cfg.EnableResolverDebug,
	})
	if err != nil {
		return fmt.Errorf("handler setup failed: %w", err)
	}
	handler = runtime.WithCanonicalNotesRedirects(handler)
	handler = seo.WithFeedAndSitemapEndpoints(handler, seo.FeedAndSitemapConfig{
		RootURL:    rootURL,
		I18nConfig: i18nConfig,
		Notes:      noteService,
	})
	handler = robots.WithRobotsEndpoint(handler, rootURL, cachePolicies.HTML)

	log.Printf("blog server listening on %s", cfg.ListenAddr)
	if err := http.ListenAndServe(cfg.ListenAddr, handler); err != nil {
		return err
	}

	return nil
}

func validateRootURL(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("BLOG_ROOT_URL is required and must be an absolute URL")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("parse BLOG_ROOT_URL %q: %w", trimmed, err)
	}
	if !parsed.IsAbs() || strings.TrimSpace(parsed.Host) == "" {
		return "", fmt.Errorf("BLOG_ROOT_URL %q must be absolute", trimmed)
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}
