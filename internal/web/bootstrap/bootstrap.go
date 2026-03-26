package bootstrap

import (
	"net/http"
	"strings"

	"blog/internal/imageloader"
	"blog/internal/notes"
	"blog/internal/robots"
	"blog/internal/seo"
	webgen "blog/internal/web/gen"
	"blog/internal/web/runtime"
	"github.com/RevoTale/no-js/framework"
	"github.com/RevoTale/no-js/framework/httpserver"
	frameworki18n "github.com/RevoTale/no-js/framework/i18n"
)

const immutableStaticCachePolicy = "public, max-age=31536000, immutable"

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

func ServerConfig(in Inputs) webgen.ServerConfig {
	cachePolicies := httpserver.DefaultCachePolicies()
	if strings.TrimSpace(in.CacheLiveNavigation) != "" {
		cachePolicies.LiveNavigation = strings.TrimSpace(in.CacheLiveNavigation)
	}
	cachePolicies.Static = immutableStaticCachePolicy

	return webgen.ServerConfig{
		Runtime: webgen.RuntimeConfig{
			AppContext: runtime.NewContext(
				in.Notes,
				in.I18nConfig,
				in.I18nCatalog,
				in.RootURL,
				in.LovelyEyeScriptURL,
				in.LovelyEyeSiteID,
			),
			Bootstrap: runtime.BootstrapConfig{
				LocalizationConfig:  in.I18nConfig,
				StaticAssetBasePath: in.StaticURLPrefix,
				ImageLoader:         in.ImageLoader,
				LovelyEyeScriptURL:  in.LovelyEyeScriptURL,
				LovelyEyeSiteID:     in.LovelyEyeSiteID,
			},
		},
		Features: webgen.FeatureConfig{
			StaticAssets: webgen.StaticAssetsConfig{
				ManifestPath: in.StaticManifestPath,
				URLPrefix:    in.StaticURLPrefix,
			},
			PublicFiles: webgen.PublicFilesConfig{
				Dir:               in.PublicDir,
				RequestPathPrefix: in.PublicRequestPathPrefix,
				CachePolicy:       strings.TrimSpace(in.CachePublicFiles),
			},
		},
		Hooks: webgen.Hooks{
			Middleware: []func(http.Handler) http.Handler{
				runtime.WithCanonicalNotesRedirects,
			},
			Mount: []func(*http.ServeMux) error{
				func(mux *http.ServeMux) error {
					return seo.Mount(mux, seo.FeedAndSitemapConfig{
						RootURL:    in.RootURL,
						I18nConfig: in.I18nConfig,
						Notes:      in.Notes,
					})
				},
				func(mux *http.ServeMux) error {
					return robots.Mount(mux, in.RootURL, cachePolicies.HTML)
				},
			},
		},
		Observability: webgen.Observability{
			CachePolicies:       cachePolicies,
			LogServerError:      in.LogServerError,
			LogResolverTiming:   in.LogResolverTiming,
			EnableResolverDebug: in.EnableResolverDebug,
		},
	}
}
