package runtime

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"blog/internal/imageloader"
	"blog/internal/notes"
	i18n "blog/web/generated/i18n"
	messages "blog/web/generated/i18n/messages"
	frameworki18n "github.com/RevoTale/no-js/framework/i18n"
	frameworksite "github.com/RevoTale/no-js/framework/site"
)

var errNotesServiceUnavailable = errors.New("notes service unavailable")

type Context struct {
	service            *notes.Service
	siteResolver       frameworksite.Resolver
	lovelyEyeScriptURL string
	lovelyEyeSiteID    string
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
	}, nil
}

func (ctx *Context) LocaleFromRequest(requestLocale string) string {
	cfg := messages.Config()
	normalized := strings.TrimSpace(strings.ToLower(requestLocale))
	if normalized == "" {
		normalized = cfg.DefaultLocale
	}
	if !slices.Contains(cfg.Locales, normalized) {
		return cfg.DefaultLocale
	}
	return normalized
}

func (ctx *Context) ResolveRoot(r *http.Request) *url.URL {
	if ctx == nil {
		return nil
	}
	return frameworksite.ResolveRoot(ctx.siteResolver, r)
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

func (ctx *Context) I18n(r *http.Request) frameworki18n.Context[i18n.Key] {
	if ctx == nil {
		var zeroRuntime *frameworki18n.Runtime[i18n.Key]
		return zeroRuntime.Context(r, nil)
	}
	return messages.NewContext(r, ctx.ResolveRoot(r))
}
