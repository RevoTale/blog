package gql

import "strings"

var localeInputByCode = map[string]LocaleInputType{
	"en": LocaleInputTypeEnUs,
	"de": LocaleInputTypeDeDe,
	"uk": LocaleInputTypeUkUa,
	"hi": LocaleInputTypeHiIn,
	"ru": LocaleInputTypeRuRu,
	"ja": LocaleInputTypeJaJp,
	"fr": LocaleInputTypeFrFr,
	"es": LocaleInputTypeEsEs,
}

var fallbackLocaleInputByCode = map[string]FallbackLocaleInputType{
	"en": FallbackLocaleInputTypeEnUs,
	"de": FallbackLocaleInputTypeDeDe,
	"uk": FallbackLocaleInputTypeUkUa,
	"hi": FallbackLocaleInputTypeHiIn,
	"ru": FallbackLocaleInputTypeRuRu,
	"ja": FallbackLocaleInputTypeJaJp,
	"fr": FallbackLocaleInputTypeFrFr,
	"es": FallbackLocaleInputTypeEsEs,
}

func LocaleInputFromCode(code string) *LocaleInputType {
	normalized := strings.ToLower(strings.TrimSpace(code))
	if value, ok := localeInputByCode[normalized]; ok {
		out := value
		return &out
	}

	out := LocaleInputTypeEnUs
	return &out
}

func FallbackLocaleInputFromCode(code string) *FallbackLocaleInputType {
	normalized := strings.ToLower(strings.TrimSpace(code))
	if value, ok := fallbackLocaleInputByCode[normalized]; ok {
		out := value
		return &out
	}

	out := FallbackLocaleInputTypeEnUs
	return &out
}
