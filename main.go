package main

import (
	"log"
	"net/http"

	"blog/internal/config"
	"blog/internal/gql"
	"blog/internal/notes"
	"blog/internal/web"
)

func main() {
	cfg := config.Load()

	graphqlClient := gql.NewClient(cfg)
	noteService := notes.NewService(graphqlClient, cfg.PageSize, cfg.RootURL)
	handler, err := web.NewHandler(cfg, noteService)
	if err != nil {
		log.Fatalf("handler setup failed: %v", err)
	}
	mux := http.NewServeMux()
	handler.Register(mux)

	log.Printf("blog server listening on %s", cfg.ListenAddr)
	if err := http.ListenAndServe(cfg.ListenAddr, mux); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
