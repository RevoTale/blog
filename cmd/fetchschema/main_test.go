package main

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveProjectRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module blog\n"), 0o644))

	nested := filepath.Join(root, "internal", "cmd", "fetchschema")
	require.NoError(t, os.MkdirAll(nested, 0o755))

	got, err := resolveProjectRoot(nested)
	require.NoError(t, err)
	assert.Equal(t, root, got)
}

func TestResolveProjectRootFailsWithoutGoMod(t *testing.T) {
	t.Parallel()

	_, err := resolveProjectRoot(t.TempDir())
	assert.Error(t, err)
}

func TestBuildHeaders(t *testing.T) {
	t.Parallel()

	extra := headerFlags{
		"X-Test": {"one"},
	}

	headers := buildHeaders("secret-token", extra)
	assert.Equal(t, "Bearer secret-token", headers.Get("Authorization"))
	assert.Equal(t, "one", headers.Get("X-Test"))
}

func TestBuildHeadersKeepsExplicitAuthorization(t *testing.T) {
	t.Parallel()

	extra := headerFlags{
		"Authorization": {"Basic abc"},
	}

	headers := buildHeaders("secret-token", extra)
	assert.Equal(t, "Basic abc", headers.Get("Authorization"))
}

func TestWriteFileAtomicReplacesExistingFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "schema.graphql")
	require.NoError(t, os.WriteFile(path, []byte("old"), 0o644))

	require.NoError(t, writeFileAtomic(path, []byte("new schema")))

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "new schema", string(got))
}

func TestHeaderFlagsSet(t *testing.T) {
	t.Parallel()

	headers := make(headerFlags)
	require.NoError(t, headers.Set("X-Test=value"))

	expected := http.Header{"X-Test": {"value"}}
	assert.Equal(t, expected.Get("X-Test"), http.Header(headers).Get("X-Test"))
}
