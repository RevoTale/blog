package imageloader

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func blogImageURL(width int, path string) string {
	return fmt.Sprintf("%s/%d/%s", blogPathPrefix, width, path)
}

func blogSrcSet(path string, widths ...int) string {
	parts := make([]string, 0, len(widths))
	for _, width := range widths {
		parts = append(parts, fmt.Sprintf("%s %dw", blogImageURL(width, path), width))
	}
	return strings.Join(parts, ", ")
}

func TestLoaderURL_DisabledReturnsOriginal(t *testing.T) {
	t.Parallel()

	loader := New(false)
	const src = "/images/hello world.webp"
	assert.Equal(t, src, loader.URL(src, 640))
}

func TestLoaderURL_RewritesCDNS3Width(t *testing.T) {
	t.Parallel()

	loader := New(true)
	got := loader.URL("/cdn/image/s3/828/files/pic.webp", 1080)
	want := "/cdn/image/s3/1080/files/pic.webp"
	assert.Equal(t, want, got)
}

func TestLoaderURL_BuildsRelativeEndpointAndEncodesSpaces(t *testing.T) {
	t.Parallel()

	loader := New(true)
	got := loader.URL("/images/hello world.webp", 640)
	want := blogImageURL(640, "images/hello%20world.webp")
	assert.Equal(t, want, got)
}

func TestLoaderURL_NormalizesToNearestAllowedWidth(t *testing.T) {
	t.Parallel()

	loader := New(true)
	assert.Equal(t, blogImageURL(48, "images/pic.webp"), loader.URL("/images/pic.webp", 40))
	assert.Equal(t, blogImageURL(828, "images/pic.webp"), loader.URL("/images/pic.webp", 768))
}

func TestLoaderResponsiveSrcSet_UsesCMSDeviceWidths(t *testing.T) {
	t.Parallel()

	loader := New(true)
	got, err := loader.ResponsiveSrcSet("/images/pic.webp", 1080)
	require.NoError(t, err)
	want := blogSrcSet("images/pic.webp", 16, 32, 48, 64, 96, 128, 256, 384, 450, 530, 640, 750, 828, 1080)
	assert.Equal(t, want, got)
}

func TestLoaderResponsiveSrcSet_SmallWidthUsesAllowedSizes(t *testing.T) {
	t.Parallel()

	loader := New(true)
	got, err := loader.ResponsiveSrcSet("/images/pic.webp", 40)
	require.NoError(t, err)
	want := blogSrcSet("images/pic.webp", 16, 32, 48)
	assert.Equal(t, want, got)
}

func TestLoaderResponsiveSrcSet_RoundsUpToAllowedTargetWidth(t *testing.T) {
	t.Parallel()

	loader := New(true)
	got, err := loader.ResponsiveSrcSet("/images/pic.webp", 768)
	require.NoError(t, err)
	want := blogSrcSet("images/pic.webp", 16, 32, 48, 64, 96, 128, 256, 384, 450, 530, 640, 750, 828)
	assert.Equal(t, want, got)
}

func TestLoaderThumb_Uses1080AndScalesHeight(t *testing.T) {
	t.Parallel()

	loader := New(true)
	url, width, height := loader.Thumb("/images/hello.webp", 1200, 630)

	assert.Equal(t, blogImageURL(1080, "images/hello.webp"), url)
	assert.Equal(t, 1080, width)
	assert.Equal(t, 567, height)
}
