package i18n

import (
	"embed"
	"fmt"
	"strings"
	"sync"

	frameworki18n "github.com/RevoTale/no-js/framework/i18n"
)

//go:embed messages/*.json
var messageFS embed.FS

var runtimeCatalogOnce sync.Once
var runtimeCatalog *frameworki18n.Catalog
var runtimeCatalogErr error

func LoadCatalog() (*frameworki18n.Catalog, error) {
	messageFiles, err := frameworki18n.DiscoverMessageFiles(messageFS)
	if err != nil {
		return nil, fmt.Errorf("discover web i18n message files: %w", err)
	}

	if err := ValidateMessageKeyParity(messageFS, messageFiles); err != nil {
		return nil, err
	}

	catalog, err := frameworki18n.LoadCatalog(messageFS, messageFiles, DefaultLocale)
	if err != nil {
		return nil, fmt.Errorf("load web i18n catalog: %w", err)
	}
	return catalog, nil
}

func LocalizeMessage(locale string, key Key, data map[string]any) string {
	fallback := strings.TrimSpace(DefaultMessages[key])
	runtimeCatalogOnce.Do(func() {
		runtimeCatalog, runtimeCatalogErr = LoadCatalog()
	})
	if runtimeCatalogErr != nil || runtimeCatalog == nil {
		return fallback
	}
	return runtimeCatalog.Localize(locale, string(key), data, fallback)
}
