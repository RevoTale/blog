package i18n

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadCatalog(t *testing.T) {
	t.Parallel()

	catalog, err := LoadCatalog()
	require.NoError(t, err)
	require.NotNil(t, catalog)
}
