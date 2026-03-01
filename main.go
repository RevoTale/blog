package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"blog/framework/httpserver"
	"blog/framework/staticassets"
	"blog/internal/config"
	"blog/internal/gql"
	"blog/internal/notes"
	"blog/internal/web/appcore"
	webgen "blog/internal/web/gen"
)

const immutableStaticCachePolicy = "public, max-age=31536000, immutable"

func main() {
	if err := run(); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}

func run() error {
	cfg := config.Load()

	manifest, err := staticassets.ReadManifest(cfg.StaticManifestPath)
	if err != nil {
		return fmt.Errorf(
			"load static manifest %q: %w (run staticassetsgen during build)",
			cfg.StaticManifestPath,
			err,
		)
	}

	appcore.SetStaticAssetBasePath(manifest.URLPrefix)

	graphqlClient := gql.NewClient(cfg)
	noteService := notes.NewService(graphqlClient, cfg.PageSize, cfg.RootURL)
	cachePolicies := httpserver.DefaultCachePolicies()
	if strings.TrimSpace(cfg.CacheLiveNavigation) != "" {
		cachePolicies.LiveNavigation = cfg.CacheLiveNavigation
	}
	cachePolicies.Static = immutableStaticCachePolicy

	handler, err := httpserver.New(httpserver.Config[*appcore.Context]{
		AppContext:      appcore.NewContext(noteService),
		Handlers:        webgen.Handlers(webgen.NewRouteResolvers()),
		IsNotFoundError: appcore.IsNotFoundError,
		NotFoundPage:    webgen.NotFoundPage,
		Static: httpserver.StaticMount{
			URLPrefix: manifest.URLPrefix,
			Dir:       cfg.StaticBuildDir,
		},
		CachePolicies: cachePolicies,
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
