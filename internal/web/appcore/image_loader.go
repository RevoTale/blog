package appcore

import (
	"fmt"
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

func ImageResponsiveSrcSet(src string, maxWidth int) string {
	srcset, err := currentImageLoader().ResponsiveSrcSet(strings.TrimSpace(src), maxWidth)

	if nil != err {
		return fmt.Sprintf("server_error:%s",err.Error())
	}

	return srcset
}

func ImageThumb(src string, originalWidth int, originalHeight int) (string, int, int) {
	return currentImageLoader().Thumb(strings.TrimSpace(src), originalWidth, originalHeight)
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
