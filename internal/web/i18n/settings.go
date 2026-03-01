package i18n

import frameworki18n "blog/framework/i18n"

const DefaultLocale = "en"

var Locales = []string{"en", "de", "uk", "hi", "ru", "ja", "fr", "es"}

func Config() frameworki18n.Config {
	return frameworki18n.Config{
		Locales:       append([]string(nil), Locales...),
		DefaultLocale: DefaultLocale,
		PrefixMode:    frameworki18n.PrefixAsNeeded,
	}
}
