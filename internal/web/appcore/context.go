package appcore

import (
	"errors"
	"slices"
	"strings"

	"blog/internal/notes"
	webi18n "blog/internal/web/i18n"
	frameworki18n "github.com/RevoTale/no-js/framework/i18n"
)

var errNotesServiceUnavailable = errors.New("notes service unavailable")

type Context struct {
	service            *notes.Service
	rootURL            string
	lovelyEyeScriptURL string
	lovelyEyeSiteID    string
	i18nConfig         frameworki18n.Config
	i18nCatalog        *frameworki18n.Catalog
}

func NewContext(
	service *notes.Service,
	i18nConfig frameworki18n.Config,
	i18nCatalog *frameworki18n.Catalog,
	rootURL string,
	lovelyEyeScriptURL string,
	lovelyEyeSiteID string,
) *Context {
	return &Context{
		service:            service,
		rootURL:            strings.TrimSpace(rootURL),
		lovelyEyeScriptURL: strings.TrimSpace(lovelyEyeScriptURL),
		lovelyEyeSiteID:    strings.TrimSpace(lovelyEyeSiteID),
		i18nConfig:         i18nConfig,
		i18nCatalog:        i18nCatalog,
	}
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
	if ctx == nil {
		return ""
	}
	return strings.TrimSpace(ctx.rootURL)
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

func IsNotFoundError(err error) bool {
	return errors.Is(err, notes.ErrNotFound)
}
