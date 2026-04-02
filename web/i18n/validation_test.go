package i18n

import (
	"encoding/json"
	"testing"
	"testing/fstest"

	i18nkeys "blog/web/generated/i18nkeys"
	"github.com/stretchr/testify/require"
)

type localeEntry struct {
	ID          string `json:"id"`
	Translation string `json:"translation"`
}

func TestValidateMessageKeyParityPassesForMatchingKeySets(t *testing.T) {
	t.Parallel()

	payload := buildLocalePayload(t, keysToStrings(i18nkeys.AllKeys))
	filesystem := fstest.MapFS{
		"messages/active.en.json": &fstest.MapFile{Data: payload},
		"messages/active.de.json": &fstest.MapFile{Data: payload},
	}

	err := ValidateMessageKeyParity(filesystem, []string{
		"messages/active.en.json",
		"messages/active.de.json",
	})
	require.NoError(t, err)
}

func TestValidateMessageKeyParityRejectsMissingKey(t *testing.T) {
	t.Parallel()

	keys := keysToStrings(i18nkeys.AllKeys)
	payload := buildLocalePayload(t, keys[1:])
	filesystem := fstest.MapFS{
		"messages/active.en.json": &fstest.MapFile{Data: payload},
	}

	err := ValidateMessageKeyParity(filesystem, []string{"messages/active.en.json"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing=")
}

func TestValidateMessageKeyParityRejectsExtraKey(t *testing.T) {
	t.Parallel()

	keys := keysToStrings(i18nkeys.AllKeys)
	keys = append(keys, "extra.invalidKey")
	payload := buildLocalePayload(t, keys)
	filesystem := fstest.MapFS{
		"messages/active.en.json": &fstest.MapFile{Data: payload},
	}

	err := ValidateMessageKeyParity(filesystem, []string{"messages/active.en.json"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "extra=")
}

func TestValidateMessageKeyParityRejectsDuplicateIDs(t *testing.T) {
	t.Parallel()

	keys := keysToStrings(i18nkeys.AllKeys)
	entries := []localeEntry{
		{ID: keys[0], Translation: "first"},
		{ID: keys[0], Translation: "second"},
	}
	payload, err := json.Marshal(entries)
	require.NoError(t, err)
	filesystem := fstest.MapFS{
		"messages/active.en.json": &fstest.MapFile{Data: payload},
	}

	err = ValidateMessageKeyParity(filesystem, []string{"messages/active.en.json"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "duplicate message id")
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
	require.NoError(t, err)
	return payload
}

func keysToStrings(keys []i18nkeys.Key) []string {
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, string(key))
	}
	return out
}
