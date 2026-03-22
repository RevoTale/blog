package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"blog/internal/config"
	"blog/internal/gql"
	"blog/internal/imageloader"
	"blog/internal/notes"
	"blog/internal/robots"
	"blog/internal/seo"
	"blog/internal/web/appcore"
	webgen "blog/internal/web/gen"
	webi18n "blog/internal/web/i18n"

	"github.com/RevoTale/no-js/framework/httpserver"
	frameworki18n "github.com/RevoTale/no-js/framework/i18n"
	"github.com/RevoTale/no-js/framework/staticassets"
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

	manifest, err := staticassets.ReadManifest(cfg.StaticManifestPath)
	if err != nil {
		return fmt.Errorf(
			"load static manifest %q: %w (run staticassetsgen during build)",
			cfg.StaticManifestPath,
			err,
		)
	}

	i18nConfig, err := frameworki18n.NormalizeConfig(webi18n.Config())
	if err != nil {
		return fmt.Errorf("normalize i18n config: %w", err)
	}
	i18nCatalog, err := webi18n.LoadCatalog()
	if err != nil {
		return fmt.Errorf("load i18n catalog: %w", err)
	}
	i18nResolver, err := frameworki18n.NewResolver(i18nConfig)
	if err != nil {
		return fmt.Errorf("create i18n resolver: %w", err)
	}

	appcore.SetStaticAssetBasePath(manifest.URLPrefix)
	appcore.SetLocalizationConfig(i18nConfig)
	imageLoader := imageloader.New(cfg.EnableImageLoader)
	appcore.SetImageLoader(imageLoader)
	appcore.SetLovelyEye(cfg.LovelyEyeScriptURL, cfg.LovelyEyeSiteID)
	log.Printf(
		"lovely eye startup: enabled=%t script_url=%q site_id=%q",
		appcore.LovelyEyeEnabled(),
		appcore.LovelyEyeScriptURL(),
		appcore.LovelyEyeSiteID(),
	)

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
	publicMiddleware, err := httpserver.WithPublicFiles(
		httpserver.PublicFilesConfig{Dir: cfg.PublicDir}.WithPublicFileCachePolicy(cfg.CachePublicFiles),
	)
	if err != nil {
		return fmt.Errorf("setup public file serving: %w", err)
	}

	// Serve static files from the manifest directory so routing cannot drift
	// to an unprocessed source dir if env configuration is stale.
	staticDir := filepath.Clean(filepath.Dir(cfg.StaticManifestPath))
	if info, statErr := os.Stat(staticDir); statErr != nil {
		return fmt.Errorf("stat static build dir %q: %w", staticDir, statErr)
	} else if !info.IsDir() {
		return fmt.Errorf("static build dir %q is not a directory", staticDir)
	}
	log.Printf(
		"static assets bundle loaded: hash=%s prefix=%s dir=%s manifest=%s",
		manifest.Hash,
		manifest.URLPrefix,
		staticDir,
		cfg.StaticManifestPath,
	)

	handler, err := httpserver.New(httpserver.Config[*appcore.Context]{
		AppContext: appcore.NewContext(
			noteService,
			i18nConfig,
			i18nCatalog,
			rootURL,
			cfg.LovelyEyeScriptURL,
			cfg.LovelyEyeSiteID,
		),
		Handlers:        webgen.Handlers(webgen.NewRouteResolvers()),
		IsNotFoundError: appcore.IsNotFoundError,
		NotFoundPage:    webgen.NotFoundPage,
		Static: httpserver.StaticMount{
			URLPrefix: manifest.URLPrefix,
			Dir:       staticDir,
		},
		CachePolicies: cachePolicies,
		LogServerError: func(err error) {
			log.Printf("blog server error: %v", err)
		},
		EnableResolverDebug: cfg.EnableResolverDebug,
	})
	if err != nil {
		return fmt.Errorf("handler setup failed: %w", err)
	}
	handler = appcore.WithCanonicalNotesRedirects(handler)
	handler = frameworki18n.Middleware(frameworki18n.MiddlewareConfig{
		Resolver: i18nResolver,
		BypassPrefixes: []string{
			manifest.URLPrefix,
		},
		BypassExact: []string{
			"/healthz",
		},
	})(handler)
	handler = publicMiddleware(handler)
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


