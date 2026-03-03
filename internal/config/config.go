package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	ListenAddr string

	StaticDir          string
	StaticBuildDir     string
	StaticManifestPath string
	PublicDir          string

	RootURL string

	CacheLiveNavigation string
	CachePublicFiles    string
	EnableImageLoader   bool

	GraphQLEndpoint  string
	GraphQLAuthToken string

	PageSize int
}

func Load() Config {
	staticDir := getEnv("BLOG_STATIC_DIR", "internal/web/static")
	staticBuildDir := getEnv("BLOG_STATIC_BUILD_DIR", "internal/web/static-build")
	staticManifestPath := getEnv(
		"BLOG_STATIC_MANIFEST_PATH",
		staticBuildDir+"/manifest.json",
	)
	publicDir := getEnv("BLOG_PUBLIC_DIR", "internal/web/public")

	return Config{
		ListenAddr:         getEnv("BLOG_LISTEN_ADDR", ":8080"),
		StaticDir:          staticDir,
		StaticBuildDir:     staticBuildDir,
		StaticManifestPath: staticManifestPath,
		PublicDir:          publicDir,
		RootURL:            getEnv("BLOG_ROOT_URL", ""),
		CacheLiveNavigation: strings.TrimSpace(
			os.Getenv("BLOG_CACHE_LIVE_NAV"),
		),
		CachePublicFiles: strings.TrimSpace(
			os.Getenv("BLOG_CACHE_PUBLIC_FILES"),
		),
		EnableImageLoader: getEnvBool("BLOG_ENABLE_IMAGE_LOADER", false),
		GraphQLEndpoint:   getEnv("BLOG_GRAPHQL_ENDPOINT", "http://localhost:3000/api/graphql"),
		GraphQLAuthToken:  os.Getenv("BLOG_GRAPHQL_AUTH_TOKEN"),
		PageSize:          getEnvInt("BLOG_NOTES_PAGE_SIZE", 12),
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

func getEnvBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}

	return parsed
}
