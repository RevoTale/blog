package i18n

import (
	"path"
	"strings"
)

func NormalizePath(rawPath string) string {
	trimmed := strings.TrimSpace(rawPath)
	if trimmed == "" {
		return "/"
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}

	cleaned := path.Clean(trimmed)
	if cleaned == "." || cleaned == "" {
		return "/"
	}
	if !strings.HasPrefix(cleaned, "/") {
		return "/" + cleaned
	}

	return cleaned
}

func LocalizePath(cfg Config, locale string, strippedPath string) string {
	normalizedCfg, err := NormalizeConfig(cfg)
	if err != nil {
		return NormalizePath(strippedPath)
	}

	normalizedLocale := normalizeLocale(locale)
	if !containsLocale(normalizedCfg, normalizedLocale) {
		normalizedLocale = normalizedCfg.DefaultLocale
	}

	normalizedPath := NormalizePath(strippedPath)
	switch normalizedCfg.PrefixMode {
	case PrefixNever:
		return normalizedPath
	case PrefixAlways:
		return prefixedPath(normalizedLocale, normalizedPath)
	case PrefixAsNeeded:
		if normalizedLocale == normalizedCfg.DefaultLocale {
			return normalizedPath
		}
		return prefixedPath(normalizedLocale, normalizedPath)
	default:
		return normalizedPath
	}
}

func StripLocale(cfg Config, rawPath string) (locale string, strippedPath string, hadPrefix bool, ok bool) {
	normalizedCfg, err := NormalizeConfig(cfg)
	if err != nil {
		return "", NormalizePath(rawPath), false, false
	}

	normalizedPath := NormalizePath(rawPath)
	segments := pathSegments(normalizedPath)
	if len(segments) == 0 {
		return normalizedCfg.DefaultLocale, "/", false, true
	}

	first := normalizeLocale(segments[0])
	if containsLocale(normalizedCfg, first) {
		return first, joinPathSegments(segments[1:]), true, true
	}
	if isLocaleLike(first) {
		return "", normalizedPath, false, false
	}

	return normalizedCfg.DefaultLocale, normalizedPath, false, true
}

func prefixedPath(locale string, strippedPath string) string {
	strippedPath = NormalizePath(strippedPath)
	if strippedPath == "/" {
		return "/" + locale
	}
	return "/" + locale + strippedPath
}

func pathSegments(pathValue string) []string {
	pathValue = NormalizePath(pathValue)
	trimmed := strings.Trim(pathValue, "/")
	if trimmed == "" {
		return []string{}
	}
	return strings.Split(trimmed, "/")
}

func joinPathSegments(segments []string) string {
	if len(segments) == 0 {
		return "/"
	}
	return "/" + strings.Join(segments, "/")
}

func containsLocale(cfg Config, locale string) bool {
	for _, candidate := range cfg.Locales {
		if candidate == locale {
			return true
		}
	}
	return false
}

func isLocaleLike(value string) bool {
	return localeCodePattern.MatchString(value)
}
