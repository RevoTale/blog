package runtime

import (
	"net/http"
	"strings"
	"sync"

	i18nkeys "blog/web/generated/i18nkeys"
	webi18n "blog/web/i18n"
	frameworki18n "github.com/RevoTale/no-js/framework/i18n"
)

var (
	fallbackI18nRuntimeOnce sync.Once
	fallbackI18nRuntime     *frameworki18n.Runtime[i18nkeys.Key]
	fallbackI18nRuntimeErr  error
)

func fallbackI18nContext(locale string) frameworki18n.Context[i18nkeys.Key] {
	fallbackI18nRuntimeOnce.Do(func() {
		fallbackI18nRuntime, fallbackI18nRuntimeErr = frameworki18n.NewRuntime(
			webi18n.Config(),
			nil,
			i18nkeys.DefaultMessages,
		)
	})
	if fallbackI18nRuntimeErr != nil || fallbackI18nRuntime == nil {
		return nil
	}

	request, _ := http.NewRequest(http.MethodGet, "http://runtime.invalid/", nil)
	request = request.WithContext(
		frameworki18n.WithRequestInfo(request.Context(), frameworki18n.RequestInfo{
			Locale:       normalizeLocaleForApp(locale),
			StrippedPath: "/",
		}),
	)
	return fallbackI18nRuntime.Context(request, nil)
}

func LocalizeAppPath(locale string, strippedPath string) string {
	i18n := fallbackI18nContext(locale)
	if i18n == nil {
		return frameworki18n.NormalizePath(strippedPath)
	}
	return i18n.Path(strippedPath)
}

func normalizeLocaleForApp(locale string) string {
	cfg, err := frameworki18n.NormalizeConfig(webi18n.Config())
	if err != nil {
		return strings.ToLower(strings.TrimSpace(locale))
	}

	normalized := strings.ToLower(strings.TrimSpace(locale))
	if normalized == "" {
		return cfg.DefaultLocale
	}
	for _, candidate := range cfg.Locales {
		if candidate == normalized {
			return normalized
		}
	}
	return cfg.DefaultLocale
}
