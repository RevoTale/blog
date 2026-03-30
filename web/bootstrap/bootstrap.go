package bootstrap

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"blog/internal/discovery"
	"blog/internal/imageloader"
	"blog/internal/notes"
	generated "blog/web/generated"
	"blog/web/view"
	"github.com/RevoTale/no-js/framework"
	"github.com/RevoTale/no-js/framework/httpserver"
	frameworki18n "github.com/RevoTale/no-js/framework/i18n"
	"github.com/RevoTale/no-js/framework/staticassets"
)

const immutableStaticCachePolicy = "public, max-age=31536000, immutable"
const defaultHealthPath = "/healthz"

type Inputs struct {
	RootURL string

	Notes       *notes.Service
	I18nConfig  frameworki18n.Config
	I18nCatalog *frameworki18n.Catalog
	ImageLoader imageloader.Loader

	StaticManifestPath      string
	StaticURLPrefix         string
	PublicDir               string
	PublicRequestPathPrefix string

	LovelyEyeScriptURL string
	LovelyEyeSiteID    string

	CacheLiveNavigation string
	CachePublicFiles    string
	EnableResolverDebug bool

	LogServerError    func(error)
	LogResolverTiming func(event framework.ResolverTiming)
}

func NewHandler(in Inputs) (http.Handler, error) {
	mux := http.NewServeMux()
	if err := MountRoutes(mux, in); err != nil {
		return nil, err
	}
	return mux, nil
}

func MountRoutes(mux *http.ServeMux, in Inputs) error {
	if mux == nil {
		return fmt.Errorf("mux is required")
	}

	cachePolicies := defaultCachePolicies(in)
	appContext := runtime.NewContext(
		in.Notes,
		in.I18nConfig,
		in.I18nCatalog,
		in.RootURL,
		in.LovelyEyeScriptURL,
		in.LovelyEyeSiteID,
	)

	runtimeCfg := runtime.BootstrapConfig{
		LocalizationConfig:  in.I18nConfig,
		StaticAssetBasePath: in.StaticURLPrefix,
		ImageLoader:         in.ImageLoader,
		LovelyEyeScriptURL:  in.LovelyEyeScriptURL,
		LovelyEyeSiteID:     in.LovelyEyeSiteID,
	}

	staticMount, err := loadStaticMount(in.StaticManifestPath, in.StaticURLPrefix)
	if err != nil {
		return err
	}
	if strings.TrimSpace(staticMount.URLPrefix) != "" {
		runtimeCfg.StaticAssetBasePath = staticMount.URLPrefix
	}
	runtime.Initialize(runtimeCfg)

	handler, err := httpserver.New(httpserver.Config[*runtime.Context]{
		AppContext:          appContext,
		Handlers:            generated.Handlers(generated.NewRouteResolvers()),
		IsNotFoundError:     runtime.IsNotFoundError,
		NotFoundPage:        generated.NotFoundPage,
		Static:              staticMount,
		CachePolicies:       cachePolicies,
		LogServerError:      in.LogServerError,
		LogResolverTiming:   in.LogResolverTiming,
		EnableResolverDebug: in.EnableResolverDebug,
	})
	if err != nil {
		return fmt.Errorf("build blog handler: %w", err)
	}

	handler = runtime.WithCanonicalNotesRedirects(handler)

	i18nResolver, err := frameworki18n.NewResolver(in.I18nConfig)
	if err != nil {
		return fmt.Errorf("create i18n resolver: %w", err)
	}

	bypassPrefixes := []string{}
	if strings.TrimSpace(staticMount.URLPrefix) != "" {
		bypassPrefixes = append(bypassPrefixes, staticMount.URLPrefix)
	}
	handler = frameworki18n.Middleware(frameworki18n.MiddlewareConfig{
		Resolver:       i18nResolver,
		BypassPrefixes: bypassPrefixes,
		BypassExact:    []string{defaultHealthPath},
	})(handler)

	publicDir := strings.TrimSpace(in.PublicDir)
	if publicDir != "" {
		publicMiddleware, err := httpserver.WithPublicFiles(httpserver.PublicFilesConfig{
			Dir:               publicDir,
			RequestPathPrefix: normalizePublicPrefix(in.PublicRequestPathPrefix),
			CachePolicy:       strings.TrimSpace(in.CachePublicFiles),
		})
		if err != nil {
			return fmt.Errorf("build public files middleware: %w", err)
		}
		handler = publicMiddleware(handler)
	}

	mux.Handle("/", handler)

	if err := discovery.MountFeedAndSitemapEndpoints(mux, discovery.FeedAndSitemapConfig{
		RootURL:    in.RootURL,
		I18nConfig: in.I18nConfig,
		Notes:      in.Notes,
	}); err != nil {
		return fmt.Errorf("mount feed and sitemap endpoints: %w", err)
	}
	if err := discovery.MountRobotsEndpoint(mux, in.RootURL, cachePolicies.HTML); err != nil {
		return fmt.Errorf("mount robots endpoint: %w", err)
	}

	return nil
}

func defaultCachePolicies(in Inputs) httpserver.CachePolicies {
	cachePolicies := httpserver.DefaultCachePolicies()
	if strings.TrimSpace(in.CacheLiveNavigation) != "" {
		cachePolicies.LiveNavigation = strings.TrimSpace(in.CacheLiveNavigation)
	}
	cachePolicies.Static = immutableStaticCachePolicy
	return cachePolicies
}

func normalizePublicPrefix(prefix string) string {
	trimmed := strings.TrimSpace(prefix)
	if trimmed == "" {
		return "/"
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	if trimmed != "/" && strings.HasSuffix(trimmed, "/") {
		trimmed = strings.TrimRight(trimmed, "/")
	}
	return trimmed
}

func loadStaticMount(manifestPath string, basePrefix string) (httpserver.StaticMount, error) {
	trimmedManifestPath := strings.TrimSpace(manifestPath)
	if trimmedManifestPath == "" {
		return httpserver.StaticMount{}, nil
	}

	manifest, err := staticassets.ReadManifest(trimmedManifestPath)
	if err != nil {
		return httpserver.StaticMount{}, fmt.Errorf("load static manifest %q: %w", trimmedManifestPath, err)
	}

	staticDir := filepath.Clean(filepath.Dir(trimmedManifestPath))
	info, statErr := os.Stat(staticDir)
	if statErr != nil {
		return httpserver.StaticMount{}, fmt.Errorf("stat static build dir %q: %w", staticDir, statErr)
	}
	if !info.IsDir() {
		return httpserver.StaticMount{}, fmt.Errorf("static build dir %q is not a directory", staticDir)
	}

	versionedPrefix := manifest.VersionedURLPrefix(basePrefix)
	return httpserver.StaticMount{URLPrefix: versionedPrefix, Dir: staticDir}, nil
}
