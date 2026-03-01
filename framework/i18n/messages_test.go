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

func TestDiscoverMessageFilesSorted(t *testing.T) {
	t.Parallel()

	filesystem := fstest.MapFS{
		"messages/active.de.json": &fstest.MapFile{Data: []byte("[]")},
		"messages/active.en.json": &fstest.MapFile{Data: []byte("[]")},
	}

	files, err := DiscoverMessageFiles(filesystem)
	if err != nil {
		t.Fatalf("discover message files: %v", err)
	}

	if got, want := strings.Join(files, ","), "messages/active.de.json,messages/active.en.json"; got != want {
		t.Fatalf("files mismatch: got %q want %q", got, want)
	}
}

func TestDiscoverMessageFilesRejectsSubdirectories(t *testing.T) {
	t.Parallel()

	filesystem := fstest.MapFS{
		"messages/nested/active.en.json": &fstest.MapFile{Data: []byte("[]")},
	}

	_, err := DiscoverMessageFiles(filesystem)
	if err == nil {
		t.Fatal("expected error for nested messages directory")
	}
	if !strings.Contains(err.Error(), "must not contain subdirectories") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDiscoverMessageFilesRejectsNonJSON(t *testing.T) {
	t.Parallel()

	filesystem := fstest.MapFS{
		"messages/active.en.json": &fstest.MapFile{Data: []byte("[]")},
		"messages/README.md":      &fstest.MapFile{Data: []byte("docs")},
	}

	_, err := DiscoverMessageFiles(filesystem)
	if err == nil {
		t.Fatal("expected error for non-json file")
	}
	if !strings.Contains(err.Error(), "must contain only json files") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateMessageKeyParityPasses(t *testing.T) {
	t.Parallel()

	payload := buildLocalePayload(t, []string{"one", "two"})
	filesystem := fstest.MapFS{
		"messages/active.en.json": &fstest.MapFile{Data: payload},
		"messages/active.de.json": &fstest.MapFile{Data: payload},
	}

	err := ValidateMessageKeyParity(filesystem, []string{
		"messages/active.en.json",
		"messages/active.de.json",
	}, []string{"one", "two"})
	if err != nil {
		t.Fatalf("expected parity validation to pass, got %v", err)
	}
}

func TestValidateMessageKeyParityRejectsMissingKey(t *testing.T) {
	t.Parallel()

	payload := buildLocalePayload(t, []string{"one"})
	filesystem := fstest.MapFS{
		"messages/active.en.json": &fstest.MapFile{Data: payload},
	}

	err := ValidateMessageKeyParity(filesystem, []string{"messages/active.en.json"}, []string{"one", "two"})
	if err == nil {
		t.Fatal("expected missing key validation error")
	}
	if !strings.Contains(err.Error(), "missing=") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateMessageKeyParityRejectsExtraKey(t *testing.T) {
	t.Parallel()

	payload := buildLocalePayload(t, []string{"one", "two", "extra"})
	filesystem := fstest.MapFS{
		"messages/active.en.json": &fstest.MapFile{Data: payload},
	}

	err := ValidateMessageKeyParity(filesystem, []string{"messages/active.en.json"}, []string{"one", "two"})
	if err == nil {
		t.Fatal("expected extra key validation error")
	}
	if !strings.Contains(err.Error(), "extra=") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateMessageKeyParityRejectsDuplicateIDs(t *testing.T) {
	t.Parallel()

	entries := []localeEntry{
		{ID: "one", Translation: "first"},
		{ID: "one", Translation: "second"},
	}
	payload, err := json.Marshal(entries)
	if err != nil {
		t.Fatalf("marshal duplicate payload: %v", err)
	}
	filesystem := fstest.MapFS{
		"messages/active.en.json": &fstest.MapFile{Data: payload},
	}

	err = ValidateMessageKeyParity(filesystem, []string{"messages/active.en.json"}, []string{"one"})
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
