package i18n

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

type PrefixMode string

const (
	PrefixAlways   PrefixMode = "always"
	PrefixAsNeeded PrefixMode = "as-needed"
	PrefixNever    PrefixMode = "never"
)

var localeCodePattern = regexp.MustCompile(`^[a-z]{2}$`)

type Config struct {
	Locales       []string
	DefaultLocale string
	PrefixMode    PrefixMode
}

func NormalizeConfig(cfg Config) (Config, error) {
	prefixMode := PrefixMode(strings.TrimSpace(string(cfg.PrefixMode)))
	if prefixMode == "" {
		prefixMode = PrefixAsNeeded
	}
	switch prefixMode {
	case PrefixAlways, PrefixAsNeeded, PrefixNever:
	default:
		return Config{}, fmt.Errorf("invalid prefix mode %q", cfg.PrefixMode)
	}

	locales := make([]string, 0, len(cfg.Locales))
	seen := make(map[string]struct{}, len(cfg.Locales))
	for _, raw := range cfg.Locales {
		locale := normalizeLocale(raw)
		if locale == "" {
			continue
		}
		if !localeCodePattern.MatchString(locale) {
			return Config{}, fmt.Errorf("invalid locale %q", raw)
		}
		if _, ok := seen[locale]; ok {
			continue
		}
		seen[locale] = struct{}{}
		locales = append(locales, locale)
	}
	if len(locales) == 0 {
		return Config{}, fmt.Errorf("at least one locale is required")
	}

	defaultLocale := normalizeLocale(cfg.DefaultLocale)
	if defaultLocale == "" {
		defaultLocale = locales[0]
	}
	if !localeCodePattern.MatchString(defaultLocale) {
		return Config{}, fmt.Errorf("invalid default locale %q", cfg.DefaultLocale)
	}
	if !slices.Contains(locales, defaultLocale) {
		return Config{}, fmt.Errorf("default locale %q is not in locales", defaultLocale)
	}

	return Config{
		Locales:       locales,
		DefaultLocale: defaultLocale,
		PrefixMode:    prefixMode,
	}, nil
}

func normalizeLocale(locale string) string {
	return strings.ToLower(strings.TrimSpace(locale))
}
