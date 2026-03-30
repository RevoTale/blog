package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"blog/internal/cmsgraphql"
	"blog/internal/config"
	"blog/internal/imageloader"
	"blog/internal/notes"
	"blog/web/bootstrap"
	webi18n "blog/web/i18n"

	frameworki18n "github.com/RevoTale/no-js/framework/i18n"
)

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

	handler, err := bootstrap.NewHandler(bootstrap.Inputs{
		RootURL:             rootURL,
		Notes:               noteService,
		I18nConfig:          i18nConfig,
		I18nCatalog:         i18nCatalog,
		ImageLoader:         imageLoader,
		StaticManifestPath:  cfg.StaticManifestPath,
		StaticURLPrefix:     cfg.StaticURLPrefix,
		PublicDir:           cfg.PublicDir,
		LovelyEyeScriptURL:  cfg.LovelyEyeScriptURL,
		LovelyEyeSiteID:     cfg.LovelyEyeSiteID,
		CacheLiveNavigation: cfg.CacheLiveNavigation,
		CachePublicFiles:    cfg.CachePublicFiles,
		EnableResolverDebug: cfg.EnableResolverDebug,
		LogServerError: func(err error) {
			log.Printf("blog server error: %v", err)
		},
	})
	if err != nil {
		return fmt.Errorf("handler setup failed: %w", err)
	}

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
