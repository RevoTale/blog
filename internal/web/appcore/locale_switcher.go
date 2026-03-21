package appcore

import (
	"sort"
	"strings"

	"github.com/RevoTale/no-js/framework/metagen"
)

type LocaleLinkItem struct {
	Code   string
	Label  string
	Href   string
	Active bool
}

var localeEndonymByCode = map[string]string{
	"en": "English",
	"de": "Deutsch",
	"uk": "Українська",
	"hi": "हिंदी",
	"ru": "Русский",
	"ja": "日本語",
	"fr": "Français",
	"es": "Español",
}

var preferredLocaleOrder = []string{"en", "de", "es", "hi", "uk", "ru", "ja", "fr"}

func FooterLocaleLinks(meta metagen.Metadata, currentLocale string) []LocaleLinkItem {
	available := meta.Alternates.Languages
	if len(available) == 0 {
		return []LocaleLinkItem{}
	}

	cleaned := make(map[string]string, len(available))
	for code, href := range available {
		normalizedCode := normalizeLocaleForApp(code)
		trimmedHref := strings.TrimSpace(href)
		if normalizedCode == "" || trimmedHref == "" {
			continue
		}
		cleaned[normalizedCode] = trimmedHref
	}
	if len(cleaned) == 0 {
		return []LocaleLinkItem{}
	}

	activeLocale := normalizeLocaleForApp(currentLocale)
	ordered := orderedLocaleCodes(cleaned)
	items := make([]LocaleLinkItem, 0, len(ordered))
	for _, code := range ordered {
		items = append(items, LocaleLinkItem{
			Code:   code,
			Label:  localeDisplayName(code),
			Href:   cleaned[code],
			Active: code == activeLocale,
		})
	}
	return items
}

func orderedLocaleCodes(values map[string]string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))

	for _, code := range preferredLocaleOrder {
		if _, ok := values[code]; !ok {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		out = append(out, code)
	}

	rest := make([]string, 0, len(values))
	for code := range values {
		if _, ok := seen[code]; ok {
			continue
		}
		rest = append(rest, code)
	}
	sort.Strings(rest)
	out = append(out, rest...)

	return out
}

func localeDisplayName(code string) string {
	normalized := normalizeLocaleForApp(code)
	if label, ok := localeEndonymByCode[normalized]; ok {
		return label
	}

	if normalized == "" {
		return ""
	}
	return strings.ToUpper(normalized)
}
