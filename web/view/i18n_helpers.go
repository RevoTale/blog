package runtime

import (
	"net/url"
	"strings"

	i18n "blog/web/generated/i18n"
	frameworki18n "github.com/RevoTale/no-js/framework/i18n"
)

func localeCode(i18n frameworki18n.Context[i18n.Key], fallback string) string {
	if i18n != nil {
		if normalized := normalizeLocaleCode(i18n.Locale()); normalized != "" {
			return normalized
		}
	}
	return normalizeLocaleCode(fallback)
}

func normalizeLocaleCode(locale string) string {
	return strings.ToLower(strings.TrimSpace(locale))
}

func localizePath(i18n frameworki18n.Context[i18n.Key], strippedPath string) string {
	if i18n == nil {
		return frameworki18n.NormalizePath(strippedPath)
	}
	return i18n.Path(strippedPath)
}

func localizePathForConfig(cfg frameworki18n.Config, locale string, strippedPath string) string {
	return frameworki18n.LocalizePath(cfg, normalizeLocaleCode(locale), strippedPath)
}

func buildLocalizedPathWithQuery(
	i18n frameworki18n.Context[i18n.Key],
	strippedPath string,
	query url.Values,
) string {
	localizedPath := localizePath(i18n, strippedPath)
	encoded := query.Encode()
	if strings.TrimSpace(encoded) == "" {
		return localizedPath
	}
	return localizedPath + "?" + encoded
}

func buildLocalizedPathWithConfigAndQuery(
	cfg frameworki18n.Config,
	locale string,
	strippedPath string,
	query url.Values,
) string {
	localizedPath := localizePathForConfig(cfg, locale, strippedPath)
	encoded := query.Encode()
	if strings.TrimSpace(encoded) == "" {
		return localizedPath
	}
	return localizedPath + "?" + encoded
}
