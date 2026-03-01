package i18n

import (
	"io/fs"
	"strings"

	frameworki18n "blog/framework/i18n"
)

func ValidateMessageKeyParity(fsys fs.FS, files []string) error {
	return frameworki18n.ValidateMessageKeyParity(fsys, files, expectedKeys())
}

func expectedKeys() []string {
	out := make([]string, 0, len(AllKeys))
	for _, key := range AllKeys {
		normalized := strings.TrimSpace(string(key))
		if normalized != "" {
			out = append(out, normalized)
		}
	}
	return out
}
