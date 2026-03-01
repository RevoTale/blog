package appcore

import (
	"errors"
	"slices"
	"strings"

	frameworki18n "blog/framework/i18n"
	"blog/internal/notes"
	webi18n "blog/internal/web/i18n"
)

var errNotesServiceUnavailable = errors.New("notes service unavailable")

type Context struct {
	service     *notes.Service
	i18nConfig  frameworki18n.Config
	i18nCatalog *frameworki18n.Catalog
}

func NewContext(
	service *notes.Service,
	i18nConfig frameworki18n.Config,
	i18nCatalog *frameworki18n.Catalog,
) *Context {
	return &Context{
		service:     service,
		i18nConfig:  i18nConfig,
		i18nCatalog: i18nCatalog,
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

func IsNotFoundError(err error) bool {
	return errors.Is(err, notes.ErrNotFound)
}
