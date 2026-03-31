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
		siteResolver.CanonicalURL(),
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

	handler, err := httpserver.NewApp(httpserver.Config[*runtime.Context]{
		App: generated.Bundle(appContext),
		Custom: newCustomConfig(
			cfg,
			siteResolver,
			noteService,
			appContext.I18nConfig(),
			func(err error) {
				log.Printf("blog server error: %v", err)
			},
		),
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
