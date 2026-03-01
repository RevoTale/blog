package i18n

import (
	"testing"
	"testing/fstest"
)

func TestCatalogLocalize(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"messages/active.en.json": {
			Data: []byte(`[
				{"id":"hello","translation":"Hello"},
				{"id":"greet","translation":"Hello {{.Name}}"}
			]`),
		},
		"messages/active.uk.json": {
			Data: []byte(`[
				{"id":"hello","translation":"Привіт"}
			]`),
		},
	}

	catalog, err := LoadCatalog(fsys, []string{
		"messages/active.en.json",
		"messages/active.uk.json",
	}, "en")
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}

	if got := catalog.Localize("uk", "hello", nil, "Hello"); got != "Привіт" {
		t.Fatalf("localized text: expected %q, got %q", "Привіт", got)
	}
	if got := catalog.Localize("uk", "missing", nil, "Fallback"); got != "Fallback" {
		t.Fatalf("fallback text: expected %q, got %q", "Fallback", got)
	}
	if got := catalog.Localize("en", "greet", map[string]any{"Name": "Bob"}, "Hello Bob"); got != "Hello Bob" {
		t.Fatalf("templated text: expected %q, got %q", "Hello Bob", got)
	}
}
