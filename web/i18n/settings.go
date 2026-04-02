package i18n

import frameworki18n "github.com/RevoTale/no-js/framework/i18n"

const DefaultLocale = "en"

var Locales = []string{"en", "de", "uk", "hi", "ru", "ja", "fr", "es"}

func Config() frameworki18n.Config {
	return frameworki18n.Config{
		Locales:       append([]string(nil), Locales...),
		DefaultLocale: DefaultLocale,
		PrefixMode:    frameworki18n.PrefixAsNeeded,
		DisplayLabels: map[string]string{
			"en": "English",
			"de": "Deutsch",
			"uk": "Українська",
			"hi": "हिंदी",
			"ru": "Русский",
			"ja": "日本語",
			"fr": "Français",
			"es": "Español",
		},
		DisplayOrder: []string{"en", "de", "es", "hi", "uk", "ru", "ja", "fr"},
	}
}
