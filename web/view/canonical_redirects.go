package runtime

import (
	"net/http"
	"strings"

	webi18n "blog/web/i18n"
	frameworki18n "github.com/RevoTale/no-js/framework/i18n"
)

func WithCanonicalNotesRedirects(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if next == nil {
			return
		}
		if r == nil || r.URL == nil || !isReadMethod(r.Method) || shouldSkipCanonicalNotesRedirect(r) {
			next.ServeHTTP(w, r)
			return
		}

		cfg := canonicalNotesConfig()
		locale, strippedPath := canonicalNotesRequestDetails(r, cfg)
		target, ok := CanonicalNotesRedirectURL(cfg, locale, strippedPath, r.URL.Query())
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		if target == currentCanonicalRequestURL(r) {
			next.ServeHTTP(w, r)
			return
		}

		http.Redirect(w, r, target, http.StatusPermanentRedirect)
	})
}

func canonicalNotesConfig() frameworki18n.Config {
	cfg, err := frameworki18n.NormalizeConfig(webi18n.Config())
	if err == nil {
		return cfg
	}

	return frameworki18n.Config{
		Locales:       []string{"en"},
		DefaultLocale: "en",
		PrefixMode:    frameworki18n.PrefixAsNeeded,
	}
}

func canonicalNotesRequestDetails(r *http.Request, cfg frameworki18n.Config) (string, string) {
	if r == nil || r.URL == nil {
		return cfg.DefaultLocale, "/"
	}

	if info, ok := frameworki18n.RequestInfoFromContext(r.Context()); ok {
		return info.Locale, info.StrippedPath
	}

	locale, strippedPath, _, ok := frameworki18n.StripLocale(cfg, r.URL.Path)
	if ok {
		return locale, strippedPath
	}

	return cfg.DefaultLocale, frameworki18n.NormalizePath(r.URL.Path)
}

func shouldSkipCanonicalNotesRedirect(r *http.Request) bool {
	if r == nil || r.URL == nil {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(r.Header.Get("HX-Request")), "true") {
		return true
	}

	return strings.TrimSpace(r.URL.Query().Get(liveNavigationQueryKey)) != ""
}

func currentCanonicalRequestURL(r *http.Request) string {
	if r == nil || r.URL == nil {
		return "/"
	}

	currentPath := strings.TrimSpace(r.URL.Path)
	if info, ok := frameworki18n.RequestInfoFromContext(r.Context()); ok && strings.TrimSpace(info.OriginalPath) != "" {
		currentPath = strings.TrimSpace(info.OriginalPath)
	}
	if currentPath == "" {
		currentPath = "/"
	}

	queryValue := strings.TrimSpace(r.URL.RawQuery)
	if queryValue == "" {
		return currentPath
	}

	return currentPath + "?" + queryValue
}

func isReadMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead
}
