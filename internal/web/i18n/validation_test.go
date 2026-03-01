package i18n

import (
	"encoding/json"
	"strings"
	"testing"
	"testing/fstest"
)

type localeEntry struct {
	ID          string `json:"id"`
	Translation string `json:"translation"`
}

func TestValidateMessageKeyParityPassesForMatchingKeySets(t *testing.T) {
	t.Parallel()

	payload := buildLocalePayload(t, keysToStrings(AllKeys))
	filesystem := fstest.MapFS{
		"messages/active.en.json": &fstest.MapFile{Data: payload},
		"messages/active.de.json": &fstest.MapFile{Data: payload},
	}

	err := ValidateMessageKeyParity(filesystem, []string{
		"messages/active.en.json",
		"messages/active.de.json",
	})
	if err != nil {
		t.Fatalf("expected parity validation to pass, got %v", err)
	}
}

func TestValidateMessageKeyParityRejectsMissingKey(t *testing.T) {
	t.Parallel()

	keys := keysToStrings(AllKeys)
	payload := buildLocalePayload(t, keys[1:])
	filesystem := fstest.MapFS{
		"messages/active.en.json": &fstest.MapFile{Data: payload},
	}

	err := ValidateMessageKeyParity(filesystem, []string{"messages/active.en.json"})
	if err == nil {
		t.Fatal("expected missing key validation error")
	}
	if !strings.Contains(err.Error(), "missing=") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateMessageKeyParityRejectsExtraKey(t *testing.T) {
	t.Parallel()

	keys := keysToStrings(AllKeys)
	keys = append(keys, "extra.invalidKey")
	payload := buildLocalePayload(t, keys)
	filesystem := fstest.MapFS{
		"messages/active.en.json": &fstest.MapFile{Data: payload},
	}

	err := ValidateMessageKeyParity(filesystem, []string{"messages/active.en.json"})
	if err == nil {
		t.Fatal("expected extra key validation error")
	}
	if !strings.Contains(err.Error(), "extra=") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateMessageKeyParityRejectsDuplicateIDs(t *testing.T) {
	t.Parallel()

	keys := keysToStrings(AllKeys)
	entries := []localeEntry{
		{ID: keys[0], Translation: "first"},
		{ID: keys[0], Translation: "second"},
	}
	payload, err := json.Marshal(entries)
	if err != nil {
		t.Fatalf("marshal duplicate payload: %v", err)
	}
	filesystem := fstest.MapFS{
		"messages/active.en.json": &fstest.MapFile{Data: payload},
	}

	err = ValidateMessageKeyParity(filesystem, []string{"messages/active.en.json"})
	if err == nil {
		t.Fatal("expected duplicate key validation error")
	}
	if !strings.Contains(err.Error(), "duplicate message id") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func buildLocalePayload(t *testing.T, keys []string) []byte {
	t.Helper()

	entries := make([]localeEntry, 0, len(keys))
	for _, key := range keys {
		entries = append(entries, localeEntry{
			ID:          key,
			Translation: key,
		})
	}

	payload, err := json.Marshal(entries)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return payload
}

func keysToStrings(keys []Key) []string {
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, string(key))
	}
	return out
}
