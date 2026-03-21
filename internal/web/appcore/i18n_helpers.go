package appcore

import (
	"slices"
	"strings"
	"sync/atomic"

	webi18n "blog/internal/web/i18n"
	frameworki18n "github.com/RevoTale/no-js/framework/i18n"
)

var routingConfigValue atomic.Value

func init() {
	defaultConfig, _ := frameworki18n.NormalizeConfig(frameworki18n.Config{
		Locales:       []string{"en"},
		DefaultLocale: "en",
		PrefixMode:    frameworki18n.PrefixAsNeeded,
	})
	routingConfigValue.Store(defaultConfig)
}

func SetLocalizationConfig(cfg frameworki18n.Config) {
	normalized, err := frameworki18n.NormalizeConfig(cfg)
	if err != nil {
		return
	}
	routingConfigValue.Store(normalized)
}

func Message(messages map[webi18n.Key]string, key webi18n.Key) string {
	if messages != nil {
		if value := strings.TrimSpace(messages[key]); value != "" {
			return value
		}
	}
	return strings.TrimSpace(webi18n.DefaultMessages[key])
}

func LocalizeAppPath(locale string, strippedPath string) string {
	cfg, _ := routingConfigValue.Load().(frameworki18n.Config)
	return frameworki18n.LocalizePath(cfg, locale, strippedPath)
}

func normalizeLocaleForApp(locale string) string {
	cfg, _ := routingConfigValue.Load().(frameworki18n.Config)
	normalized := strings.ToLower(strings.TrimSpace(locale))
	if normalized == "" {
		return cfg.DefaultLocale
	}
	if slices.Contains(cfg.Locales, normalized) {
		return normalized
	}
	return cfg.DefaultLocale
}

func localizedMessages(appCtx *Context, locale string) map[webi18n.Key]string {
	out := make(map[webi18n.Key]string, len(webi18n.AllKeys))
	for _, key := range webi18n.AllKeys {
		if appCtx == nil {
			out[key] = webi18n.LocalizeMessage(locale, key, nil)
			continue
		}
		out[key] = appCtx.T(locale, key, nil)
	}
	return out
}
