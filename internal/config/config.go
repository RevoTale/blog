package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	ListenAddr string
	StaticDir  string

	RootURL string

	CacheLiveNavigation string

	GraphQLEndpoint  string
	GraphQLAuthToken string

	PageSize int
}

func Load() Config {
	return Config{
		ListenAddr: getEnv("BLOG_LISTEN_ADDR", ":8080"),
		StaticDir:  getEnv("BLOG_STATIC_DIR", "internal/web/static"),
		RootURL:    getEnv("BLOG_ROOT_URL", ""),
		CacheLiveNavigation: strings.TrimSpace(
			os.Getenv("BLOG_CACHE_LIVE_NAV"),
		),
		GraphQLEndpoint:  getEnv("BLOG_GRAPHQL_ENDPOINT", "http://localhost:3000/api/graphql"),
		GraphQLAuthToken: os.Getenv("BLOG_GRAPHQL_AUTH_TOKEN"),
		PageSize:         getEnvInt("BLOG_NOTES_PAGE_SIZE", 12),
	}
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return fallback
	}

	return parsed
}
