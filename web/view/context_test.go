package runtime

import (
	"net/http"
	"testing"

	"blog/internal/config"
	"blog/internal/imageloader"
	"blog/internal/notes"
	"blog/internal/site"
	"github.com/stretchr/testify/require"
)

type stubSiteResolver struct{}

func (stubSiteResolver) CanonicalURL() string {
	return "https://example.com"
}

func (stubSiteResolver) Resolve(*http.Request) string {
	return "https://example.com"
}

func TestNewContextRejectsMissingNotesService(t *testing.T) {
	t.Parallel()

	ctx, err := NewContext(Config{
		SiteResolver: stubSiteResolver{},
		ImageLoader:  imageloader.New(false),
	})

	require.Nil(t, ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "notes service is required")
}

func TestNewContextRejectsMissingSiteResolver(t *testing.T) {
	t.Parallel()

	ctx, err := NewContext(Config{
		Notes:       notes.NewService(nil, 12, imageloader.New(false)),
		ImageLoader: imageloader.New(false),
	})

	require.Nil(t, ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "site resolver is required")
}

func TestNewContextAcceptsRequiredDependencies(t *testing.T) {
	t.Parallel()

	resolver, err := site.NewResolver(config.Config{RootURL: "https://example.com"})
	require.NoError(t, err)

	ctx, err := NewContext(Config{
		Notes:        notes.NewService(nil, 12, imageloader.New(false)),
		SiteResolver: resolver,
		ImageLoader:  imageloader.New(false),
	})

	require.NoError(t, err)
	require.NotNil(t, ctx)
}
