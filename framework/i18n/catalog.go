package i18n

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"

	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

type Catalog struct {
	bundle        *goi18n.Bundle
	defaultLocale string
}

func LoadCatalog(fsys fs.FS, files []string, defaultLocale string) (*Catalog, error) {
	normalizedDefault := normalizeLocale(defaultLocale)
	if normalizedDefault == "" {
		return nil, fmt.Errorf("default locale is required")
	}

	bundle := goi18n.NewBundle(language.Make(normalizedDefault))
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	for _, file := range files {
		file = strings.TrimSpace(file)
		if file == "" {
			continue
		}
		content, err := fs.ReadFile(fsys, file)
		if err != nil {
			return nil, fmt.Errorf("read locale file %q: %w", file, err)
		}
		if _, err := bundle.ParseMessageFileBytes(content, file); err != nil {
			return nil, fmt.Errorf("parse locale file %q: %w", file, err)
		}
	}

	return &Catalog{
		bundle:        bundle,
		defaultLocale: normalizedDefault,
	}, nil
}

func (catalog *Catalog) Localize(
	locale string,
	messageID string,
	data map[string]any,
	fallback string,
) string {
	if catalog == nil || catalog.bundle == nil {
		if strings.TrimSpace(fallback) != "" {
			return fallback
		}
		return strings.TrimSpace(messageID)
	}

	messageID = strings.TrimSpace(messageID)
	fallback = strings.TrimSpace(fallback)
	if messageID == "" {
		return fallback
	}

	normalizedLocale := normalizeLocale(locale)
	if normalizedLocale == "" {
		normalizedLocale = catalog.defaultLocale
	}

	localizer := goi18n.NewLocalizer(catalog.bundle, normalizedLocale, catalog.defaultLocale)
	result, err := localizer.Localize(&goi18n.LocalizeConfig{
		MessageID:    messageID,
		TemplateData: data,
		DefaultMessage: &goi18n.Message{
			ID:    messageID,
			Other: fallback,
		},
	})
	if err != nil {
		if fallback != "" {
			return fallback
		}
		return messageID
	}

	if strings.TrimSpace(result) == "" {
		if fallback != "" {
			return fallback
		}
		return messageID
	}

	return result
}
