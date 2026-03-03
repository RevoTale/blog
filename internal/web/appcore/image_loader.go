package appcore

import (
	"strconv"
	"strings"
	"sync/atomic"

	"blog/internal/imageloader"
)

var imageLoaderValue atomic.Value

func init() {
	imageLoaderValue.Store(imageloader.New(false))
}

func SetImageLoader(loader imageloader.Loader) {
	imageLoaderValue.Store(loader)
}

func ImageLoaderEnabled() bool {
	return currentImageLoader().Enabled()
}

func ImageURL(src string, width int) string {
	return currentImageLoader().URL(strings.TrimSpace(src), width)
}

func ImageFixedSrcSet(src string, displayWidth int) string {
	return currentImageLoader().FixedSrcSet(strings.TrimSpace(src), displayWidth)
}

func ImageResponsiveSrcSet(src string, maxWidth int) string {
	return currentImageLoader().ResponsiveSrcSet(strings.TrimSpace(src), maxWidth)
}

func ImageThumb(src string, originalWidth int, originalHeight int) (string, int, int) {
	return currentImageLoader().Thumb(strings.TrimSpace(src), originalWidth, originalHeight)
}

func ImageDisplaySize(width int) string {
	if width < 1 {
		return "100vw"
	}
	return strconv.Itoa(width) + "px"
}

func ImageResponsiveSizes() string {
	return "100vw"
}

func MarkdownImageSizes() string {
	return imageloader.MarkdownSizes()
}

func currentImageLoader() imageloader.Loader {
	loader, ok := imageLoaderValue.Load().(imageloader.Loader)
	if !ok {
		return imageloader.New(false)
	}
	return loader
}
