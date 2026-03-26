package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	ListenAddr string

	StaticManifestPath string
	StaticURLPrefix    string
	PublicDir          string

	RootURL string

	LovelyEyeScriptURL string
	LovelyEyeSiteID    string

	CacheLiveNavigation string
	CachePublicFiles    string
	EnableImageLoader   bool
	EnableResolverDebug bool

	GraphQLEndpoint  string
	GraphQLAuthToken string

	PageSize int
}

func Load() Config {
	staticManifestPath := getEnv("BLOG_STATIC_MANIFEST_PATH", "internal/web/static-build/manifest.json")
	publicDir := getEnv("BLOG_PUBLIC_DIR", "internal/web/public")

	return Config{
		ListenAddr:         getEnv("BLOG_LISTEN_ADDR", ":8080"),
		StaticManifestPath: staticManifestPath,
		StaticURLPrefix:    getEnv("BLOG_STATIC_URL_PREFIX", "/_assets/"),
		PublicDir:          publicDir,
		RootURL:            getEnv("BLOG_ROOT_URL", ""),
		LovelyEyeScriptURL: strings.TrimSpace(os.Getenv("LOVELY_EYE_SCRIPT_URL")),
		LovelyEyeSiteID:    strings.TrimSpace(os.Getenv("LOVELY_EYE_SITE_ID")),
		CacheLiveNavigation: strings.TrimSpace(
			os.Getenv("BLOG_CACHE_LIVE_NAV"),
		),
		CachePublicFiles: strings.TrimSpace(
			os.Getenv("BLOG_CACHE_PUBLIC_FILES"),
		),
		EnableImageLoader:   getEnvBool("BLOG_ENABLE_IMAGE_LOADER", false),
		EnableResolverDebug: getEnvBool("BLOG_ENABLE_RESOLVER_DEBUG", false),
		GraphQLEndpoint:     getEnv("BLOG_GRAPHQL_ENDPOINT", "http://localhost:3000/api/graphql"),
		GraphQLAuthToken:    os.Getenv("BLOG_GRAPHQL_AUTH_TOKEN"),
		PageSize:            getEnvInt("BLOG_NOTES_PAGE_SIZE", 12),
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
