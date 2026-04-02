package i18n

import (
	"io/fs"
	"strings"

	i18nkeys "blog/web/generated/i18nkeys"
	frameworki18n "github.com/RevoTale/no-js/framework/i18n"
)

func ValidateMessageKeyParity(fsys fs.FS, files []string) error {
	return ValidateMessageCatalog(fsys, files)
}

func ValidateMessageCatalog(fsys fs.FS, files []string) error {
	return frameworki18n.ValidateMessageCatalog(
		fsys,
		files,
		"messages/active."+DefaultLocale+".json",
		expectedKeys(),
	)
}

func expectedKeys() []string {
	out := make([]string, 0, len(i18nkeys.AllKeys))
	for _, key := range i18nkeys.AllKeys {
		normalized := strings.TrimSpace(string(key))
		if normalized != "" {
			out = append(out, normalized)
		}
	}
	return out
}
