package i18n

import (
	"embed"
	"fmt"

	frameworki18n "github.com/RevoTale/no-js/framework/i18n"
)

//go:embed messages/*.json
var messageFS embed.FS

func LoadCatalog() (*frameworki18n.Catalog, error) {
	messageFiles, err := frameworki18n.DiscoverMessageFiles(messageFS)
	if err != nil {
		return nil, fmt.Errorf("discover web i18n message files: %w", err)
	}

	if err := ValidateMessageCatalog(messageFS, messageFiles); err != nil {
		return nil, err
	}

	catalog, err := frameworki18n.LoadCatalog(messageFS, messageFiles, DefaultLocale)
	if err != nil {
		return nil, fmt.Errorf("load web i18n catalog: %w", err)
	}
	return catalog, nil
}
