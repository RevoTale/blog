package keygen

import (
	"strings"
	"testing"
)

func TestBuildKeyDefsDeterministicNames(t *testing.T) {
	t.Parallel()

	messages := []Message{
		{ID: "layout.channelsButton", Translation: "Channels"},
		{ID: "note.publishedPrefix", Translation: "published"},
		{ID: "composer.readOnly", Translation: "read only"},
		{ID: "markdown.code.copy", Translation: "copy"},
	}

	defs, err := BuildKeyDefs(messages)
	if err != nil {
		t.Fatalf("build key defs: %v", err)
	}
	if len(defs) != len(messages) {
		t.Fatalf("defs length: expected %d, got %d", len(messages), len(defs))
	}

	expectedByID := map[string]string{
		"composer.readOnly":     "KeyComposerReadOnly",
		"layout.channelsButton": "KeyLayoutChannelsButton",
		"markdown.code.copy":    "KeyMarkdownCodeCopy",
		"note.publishedPrefix":  "KeyNotePublishedPrefix",
	}
	for _, def := range defs {
		expectedName, ok := expectedByID[def.ID]
		if !ok {
			t.Fatalf("unexpected id in defs: %q", def.ID)
		}
		if def.Name != expectedName {
			t.Fatalf("const name for %q: expected %q, got %q", def.ID, expectedName, def.Name)
		}
	}
}

func TestParseCanonicalRejectsDuplicateIDs(t *testing.T) {
	t.Parallel()

	_, err := ParseCanonical([]byte(`[
		{"id":"a.b","translation":"x"},
		{"id":"a.b","translation":"y"}
	]`))
	if err == nil {
		t.Fatal("expected duplicate id error")
	}
	if !strings.Contains(err.Error(), "duplicate message id") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildKeyDefsDetectsConstNameCollision(t *testing.T) {
	t.Parallel()

	_, err := BuildKeyDefs([]Message{
		{ID: "a.bC", Translation: "first"},
		{ID: "aB.c", Translation: "second"},
	})
	if err == nil {
		t.Fatal("expected key constant collision error")
	}
	if !strings.Contains(err.Error(), "collide") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGenerateFromJSONStableOutput(t *testing.T) {
	t.Parallel()

	source := []byte(`[
		{"id":"z.b","translation":"Z"},
		{"id":"a.b","translation":"A"}
	]`)

	first, err := GenerateFromJSON("i18n", source)
	if err != nil {
		t.Fatalf("first generation failed: %v", err)
	}
	second, err := GenerateFromJSON("i18n", source)
	if err != nil {
		t.Fatalf("second generation failed: %v", err)
	}

	if string(first) != string(second) {
		t.Fatalf("generated output should be deterministic")
	}
	if !strings.Contains(string(first), "type Key string") {
		t.Fatalf("generated output missing Key type")
	}
	if !strings.Contains(string(first), "var DefaultMessages = map[Key]string") {
		t.Fatalf("generated output missing DefaultMessages map")
	}
}
