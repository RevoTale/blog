package site

import (
	"testing"

	"blog/internal/config"
	"github.com/stretchr/testify/require"
)

func TestNewResolverCanonicalURLNormalizesParsedRoot(t *testing.T) {
	t.Parallel()

	resolver, err := NewResolver(config.Config{
		RootURL: " https://example.com/blog?utm_source=test#fragment ",
	})
	require.NoError(t, err)
	require.Equal(t, "https://example.com/blog", resolver.CanonicalURL())
}

func TestNewResolverRejectsRelativeRootURL(t *testing.T) {
	t.Parallel()

	resolver, err := NewResolver(config.Config{RootURL: "/blog"})
	require.Nil(t, resolver)
	require.Error(t, err)
	require.Contains(t, err.Error(), "must be absolute")
}
