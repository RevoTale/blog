package runtime

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"

	"blog/internal/imageloader"
)

var imageLoaderValue atomic.Value
var maxWidthMediaPattern = regexp.MustCompile(`(?i)max-width\s*:\s*([0-9]+)px`)

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

	if err != nil {
		return fmt.Sprintf("server_error:%s", err.Error())
	}

	return srcset
}

func ImageThumb(src string, originalWidth int, originalHeight int) (string, int, int) {
	return currentImageLoader().Thumb(strings.TrimSpace(src), originalWidth, originalHeight)
}

func ImageResponsiveSizes(rawSizes string, fallbackWidth int) string {
	sizes := strings.TrimSpace(rawSizes)
	if sizes != "" && imageSizesMaxWidth(sizes) > 0 {
		return sizes
	}
	if fallbackWidth > 0 {
		return strconv.Itoa(fallbackWidth) + "px"
	}
	return "100vw"
}

func ImageResponsiveTargetWidth(intrinsicWidth int, sizes string) int {
	slotMax := imageSizesMaxWidth(strings.TrimSpace(sizes))
	if slotMax <= 0 {
		return intrinsicWidth
	}
	if intrinsicWidth > 0 && intrinsicWidth < slotMax {
		return intrinsicWidth
	}
	return slotMax
}

func currentImageLoader() imageloader.Loader {
	loader, ok := imageLoaderValue.Load().(imageloader.Loader)
	if !ok {
		return imageloader.New(false)
	}
	return loader
}

func imageSizesMaxWidth(sizes string) int {
	segments := strings.Split(strings.TrimSpace(sizes), ",")
	maxWidth := 0
	for _, segment := range segments {
		value := strings.TrimSpace(segment)
		if value == "" {
			continue
		}
		mediaMax := mediaMaxWidth(value)
		slot := slotSizeToken(value)
		slotWidth := parseSlotWidth(slot, mediaMax)
		if slotWidth > maxWidth {
			maxWidth = slotWidth
		}
	}
	return maxWidth
}

func mediaMaxWidth(value string) int {
	match := maxWidthMediaPattern.FindStringSubmatch(value)
	if len(match) != 2 {
		return 0
	}
	width, err := strconv.Atoi(strings.TrimSpace(match[1]))
	if err != nil || width <= 0 {
		return 0
	}
	return width
}

func slotSizeToken(value string) string {
	slot := strings.TrimSpace(value)
	if closeParen := strings.LastIndex(slot, ")"); closeParen >= 0 {
		slot = strings.TrimSpace(slot[closeParen+1:])
	}
	fields := strings.Fields(slot)
	if len(fields) == 0 {
		return ""
	}
	return strings.TrimSpace(fields[0])
}

func parseSlotWidth(slot string, mediaMax int) int {
	const maxViewportWidth = 1920
	trimmed := strings.TrimSpace(strings.ToLower(slot))
	switch {
	case strings.HasSuffix(trimmed, "px"):
		value := strings.TrimSuffix(trimmed, "px")
		width, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err != nil || width <= 0 {
			return 0
		}
		return int(width + 0.5)
	case strings.HasSuffix(trimmed, "vw"):
		value := strings.TrimSuffix(trimmed, "vw")
		ratio, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err != nil || ratio <= 0 {
			return 0
		}
		viewport := maxViewportWidth
		if mediaMax > 0 && mediaMax < viewport {
			viewport = mediaMax
		}
		return int((ratio*float64(viewport))/100.0 + 0.5)
	default:
		return 0
	}
}
