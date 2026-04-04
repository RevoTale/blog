package i18n

import (
	"io/fs"
	"strings"

	generatedi18n "blog/web/generated/i18n"
	frameworki18n "github.com/RevoTale/no-js/framework/i18n"
)

func ValidateMessageKeyParity(fsys fs.FS, files []string) error {
	return ValidateMessageCatalog(fsys, files)
}

func ValidateMessageCatalog(fsys fs.FS, files []string) error {
	return frameworki18n.ValidateMessageCatalog(
		fsys,
		files,
		DefaultLocale,
		expectedKeys(),
	)
}

func expectedKeys() []string {
	out := make([]string, 0, len(generatedi18n.Keys))
	for _, key := range generatedi18n.Keys {
		normalized := strings.TrimSpace(string(key))
		if normalized != "" {
			out = append(out, normalized)
		}
	}
	return out
}
