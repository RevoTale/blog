package runtime

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"blog/internal/imageloader"
	"blog/internal/notes"
	i18nkeys "blog/web/generated/i18nkeys"
	webi18n "blog/web/i18n"
	"github.com/RevoTale/no-js/framework"
	frameworki18n "github.com/RevoTale/no-js/framework/i18n"
	frameworksite "github.com/RevoTale/no-js/framework/site"
)

var errNotesServiceUnavailable = errors.New("notes service unavailable")

type Context struct {
	service            *notes.Service
	siteResolver       frameworksite.Resolver
	lovelyEyeScriptURL string
	lovelyEyeSiteID    string
	i18nConfig         frameworki18n.Config
	i18nCatalog        *frameworki18n.Catalog
	i18nRuntime        *frameworki18n.Runtime[i18nkeys.Key]
}

type Config struct {
	Notes              *notes.Service
	SiteResolver       frameworksite.Resolver
	ImageLoader        imageloader.Loader
	LovelyEyeScriptURL string
	LovelyEyeSiteID    string
}

func NewContext(cfg Config) (*Context, error) {
	if cfg.Notes == nil {
		return nil, fmt.Errorf("notes service is required")
	}
	if cfg.SiteResolver == nil {
		return nil, fmt.Errorf("site resolver is required")
	}

	i18nConfig, err := frameworki18n.NormalizeConfig(webi18n.Config())
	if err != nil {
		return nil, fmt.Errorf("normalize i18n config: %w", err)
	}
	i18nCatalog, err := webi18n.LoadCatalog()
	if err != nil {
		return nil, fmt.Errorf("load i18n catalog: %w", err)
	}
	i18nRuntime, err := frameworki18n.NewRuntime(i18nConfig, i18nCatalog, i18nkeys.DefaultMessages)
	if err != nil {
		return nil, fmt.Errorf("build i18n runtime: %w", err)
	}

	Initialize(BootstrapConfig{
		ImageLoader:        cfg.ImageLoader,
		LovelyEyeScriptURL: cfg.LovelyEyeScriptURL,
		LovelyEyeSiteID:    cfg.LovelyEyeSiteID,
	})

	return &Context{
		service:            cfg.Notes,
		siteResolver:       cfg.SiteResolver,
		lovelyEyeScriptURL: strings.TrimSpace(cfg.LovelyEyeScriptURL),
		lovelyEyeSiteID:    strings.TrimSpace(cfg.LovelyEyeSiteID),
		i18nConfig:         i18nConfig,
		i18nCatalog:        i18nCatalog,
		i18nRuntime:        i18nRuntime,
	}, nil
}

func (ctx *Context) LocaleFromRequest(requestLocale string) string {
	normalized := strings.TrimSpace(strings.ToLower(requestLocale))
	if normalized == "" {
		normalized = ctx.i18nConfig.DefaultLocale
	}
	if !slices.Contains(ctx.i18nConfig.Locales, normalized) {
		return ctx.i18nConfig.DefaultLocale
	}
	return normalized
}

func (ctx *Context) LocalizedPath(locale string, strippedPath string) string {
	return frameworki18n.LocalizePath(ctx.i18nConfig, locale, strippedPath)
}

func (ctx *Context) T(locale string, key i18nkeys.Key, data map[string]any) string {
	if ctx == nil || ctx.i18nRuntime == nil {
		return strings.TrimSpace(i18nkeys.DefaultMessages[key])
	}
	return ctx.i18nRuntime.Localize(ctx.LocaleFromRequest(locale), key, data)
}

func (ctx *Context) ResolveRoot(r *http.Request) *url.URL {
	if ctx == nil {
		return nil
	}
	return frameworksite.ResolveRoot(ctx.siteResolver, r)
}

func (ctx *Context) ResolveRootURL(r *http.Request) string {
	root := ctx.ResolveRoot(r)
	if root == nil {
		return ""
	}
	return root.String()
}

func (ctx *Context) SiteResolver() frameworksite.Resolver {
	if ctx == nil {
		return nil
	}
	return ctx.siteResolver
}

func (ctx *Context) Notes() *notes.Service {
	if ctx == nil {
		return nil
	}
	return ctx.service
}

func (ctx *Context) LovelyEyeEnabled() bool {
	return strings.TrimSpace(ctx.LovelyEyeScriptURL()) != "" &&
		strings.TrimSpace(ctx.LovelyEyeSiteID()) != ""
}

func (ctx *Context) LovelyEyeScriptURL() string {
	if ctx == nil {
		return ""
	}

	return strings.TrimSpace(ctx.lovelyEyeScriptURL)
}

func (ctx *Context) LovelyEyeSiteID() string {
	if ctx == nil {
		return ""
	}

	return strings.TrimSpace(ctx.lovelyEyeSiteID)
}

func (ctx *Context) I18nConfig() frameworki18n.Config {
	if ctx == nil {
		return frameworki18n.Config{}
	}
	return ctx.i18nConfig
}

func (ctx *Context) I18n(r *http.Request) frameworki18n.Context[i18nkeys.Key] {
	if ctx == nil || ctx.i18nRuntime == nil {
		return nil
	}
	return ctx.i18nRuntime.Context(r, ctx.ResolveRoot(r))
}

func (ctx *Context) MetaContext(reqCtx context.Context, r *http.Request) framework.MetaContext[*Context] {
	if ctx == nil {
		return nil
	}
	return framework.NewMetaContext(reqCtx, ctx, r)
}
