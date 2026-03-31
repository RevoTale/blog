package runtime

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"blog/internal/imageloader"
	"blog/internal/notes"
	"blog/internal/site"
	webi18n "blog/web/i18n"
	frameworki18n "github.com/RevoTale/no-js/framework/i18n"
)

var errNotesServiceUnavailable = errors.New("notes service unavailable")

type Context struct {
	service            *notes.Service
	siteResolver       site.Resolver
	lovelyEyeScriptURL string
	lovelyEyeSiteID    string
	i18nConfig         frameworki18n.Config
	i18nCatalog        *frameworki18n.Catalog
}

type Config struct {
	Notes              *notes.Service
	SiteResolver       site.Resolver
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

	Initialize(BootstrapConfig{
		LocalizationConfig: i18nConfig,
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

func (ctx *Context) T(locale string, key webi18n.Key, data map[string]any) string {
	fallback := strings.TrimSpace(webi18n.DefaultMessages[key])
	if ctx == nil || ctx.i18nCatalog == nil {
		return fallback
	}
	return ctx.i18nCatalog.Localize(locale, string(key), data, fallback)
}

func (ctx *Context) RootURL() string {
	if ctx == nil || ctx.siteResolver == nil {
		return ""
	}
	return strings.TrimSpace(ctx.siteResolver.CanonicalURL())
}

func (ctx *Context) SiteResolver() site.Resolver {
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
