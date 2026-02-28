package main

import (
	"log"
	"net/http"
	"strings"

	"blog/framework/httpserver"
	"blog/internal/config"
	"blog/internal/gql"
	"blog/internal/notes"
	"blog/internal/web/appcore"
	webgen "blog/internal/web/gen"
)

func main() {
	cfg := config.Load()

	graphqlClient := gql.NewClient(cfg)
	noteService := notes.NewService(graphqlClient, cfg.PageSize, cfg.RootURL)
	cachePolicies := httpserver.DefaultCachePolicies()
	if strings.TrimSpace(cfg.CacheLiveNavigation) != "" {
		cachePolicies.LiveNavigation = cfg.CacheLiveNavigation
	}
	handler, err := httpserver.New(httpserver.Config[*appcore.Context]{
		AppContext:      appcore.NewContext(noteService),
		Handlers:        webgen.Handlers(webgen.NewRouteResolvers()),
		IsNotFoundError: appcore.IsNotFoundError,
		NotFoundPage:    webgen.NotFoundPage,
		Static: httpserver.StaticMount{
			URLPrefix: "/.revotale/",
			Dir:       cfg.StaticDir,
		},
		CachePolicies: cachePolicies,
		LogServerError: func(err error) {
			log.Printf("blog server error: %v", err)
		},
	})
	if err != nil {
		log.Fatalf("handler setup failed: %v", err)
	}

	log.Printf("blog server listening on %s", cfg.ListenAddr)
	if err := http.ListenAndServe(cfg.ListenAddr, handler); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
