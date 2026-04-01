package main

import (
	"fmt"
	"log"
	"net/http"

	"blog/internal/cmsgraphql"
	"blog/internal/config"
	"blog/internal/imageloader"
	"blog/internal/notes"
	"blog/internal/site"
	generated "blog/web/generated"
	runtime "blog/web/view"
	"github.com/RevoTale/no-js/framework/httpserver"
)

const immutableStaticCachePolicy = "public, max-age=31536000, immutable"
const blogLiveNavigationCachePolicy = "public, max-age=3600, s-maxage=3600"

func main() {
	if err := run(); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}

func run() error {
	cfg := config.Load()
	siteResolver, err := site.NewResolver(cfg)
	if err != nil {
		return err
	}

	imageLoader := imageloader.New(cfg.EnableImageLoader)

	graphqlClient := gql.NewClient(cfg)
	noteService := notes.NewService(
		graphqlClient,
		cfg.PageSize,
		imageLoader,
	)

	appContext, err := runtime.NewContext(runtime.Config{
		Notes:              noteService,
		SiteResolver:       siteResolver,
		ImageLoader:        imageLoader,
		LovelyEyeScriptURL: cfg.LovelyEyeScriptURL,
		LovelyEyeSiteID:    cfg.LovelyEyeSiteID,
	})
	if err != nil {
		return fmt.Errorf("build app context: %w", err)
	}

	cachePolicies := httpserver.DefaultCachePolicies()
	cachePolicies.Static = immutableStaticCachePolicy
	cachePolicies.LiveNavigation = blogLiveNavigationCachePolicy

	handler, err := httpserver.NewApp(httpserver.Config[*runtime.Context]{
		App: generated.Bundle(appContext),
		Custom: httpserver.CustomConfig{
			MainMiddlewares: []func(http.Handler) http.Handler{
				runtime.WithCanonicalNotesRedirects,
			},
			CachePolicies: cachePolicies,
			LogServerError: func(err error) {
				log.Printf("blog server error: %v", err)
			},
			EnableResolverDebug: cfg.EnableResolverDebug,
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
