package imageloader

import "testing"

func TestLoaderURL_DisabledReturnsOriginal(t *testing.T) {
	t.Parallel()

	loader := New(false)
	const src = "/images/hello world.webp"
	if got := loader.URL(src, 640); got != src {
		t.Fatalf("URL disabled: expected %q, got %q", src, got)
	}
}

func TestLoaderURL_RewritesCDNS3Width(t *testing.T) {
	t.Parallel()

	loader := New(true)
	got := loader.URL("/cdn/image/s3/828/files/pic.webp", 1080)
	want := "/cdn/image/s3/1080/files/pic.webp"
	if got != want {
		t.Fatalf("URL s3 rewrite: expected %q, got %q", want, got)
	}
}

func TestLoaderURL_BuildsRelativeEndpointAndEncodesSpaces(t *testing.T) {
	t.Parallel()

	loader := New(true)
	got := loader.URL("/images/hello world.webp", 640)
	want := "/cdn/image/relative/640/images/hello%20world.webp"
	if got != want {
		t.Fatalf("URL relative: expected %q, got %q", want, got)
	}
}

func TestLoaderFixedSrcSet_EmitsOneXTwoX(t *testing.T) {
	t.Parallel()

	loader := New(true)
	got := loader.FixedSrcSet("/cdn/image/s3/828/files/pic.webp", 40)
	want := "/cdn/image/s3/40/files/pic.webp 1x, /cdn/image/s3/80/files/pic.webp 2x"
	if got != want {
		t.Fatalf("fixed srcset: expected %q, got %q", want, got)
	}
}

func TestLoaderResponsiveSrcSet_UsesCMSDeviceWidths(t *testing.T) {
	t.Parallel()

	loader := New(true)
	got := loader.ResponsiveSrcSet("/images/pic.webp", 1080)
	want := "/cdn/image/relative/384/images/pic.webp 384w, " +
		"/cdn/image/relative/450/images/pic.webp 450w, " +
		"/cdn/image/relative/530/images/pic.webp 530w, " +
		"/cdn/image/relative/640/images/pic.webp 640w, " +
		"/cdn/image/relative/828/images/pic.webp 828w, " +
		"/cdn/image/relative/1080/images/pic.webp 1080w"
	if got != want {
		t.Fatalf("responsive srcset: expected %q, got %q", want, got)
	}
}

func TestLoaderThumb_Uses1080AndScalesHeight(t *testing.T) {
	t.Parallel()

	loader := New(true)
	url, width, height := loader.Thumb("/images/hello.webp", 1200, 630)

	if want := "/cdn/image/relative/1080/images/hello.webp"; url != want {
		t.Fatalf("thumb url: expected %q, got %q", want, url)
	}
	if width != 1080 {
		t.Fatalf("thumb width: expected %d, got %d", 1080, width)
	}
	if height != 567 {
		t.Fatalf("thumb height: expected %d, got %d", 567, height)
	}
}
