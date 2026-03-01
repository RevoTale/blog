package appcore

import (
	"strings"
	"sync/atomic"
)

const defaultStaticAssetBasePath = "/.revotale/"

var staticAssetBasePath atomic.Value

func init() {
	staticAssetBasePath.Store(defaultStaticAssetBasePath)
}

func SetStaticAssetBasePath(prefix string) {
	staticAssetBasePath.Store(normalizeStaticAssetBasePath(prefix))
}

func StaticAssetURL(path string) string {
	basePath, _ := staticAssetBasePath.Load().(string)
	if strings.TrimSpace(basePath) == "" {
		basePath = defaultStaticAssetBasePath
	}

	trimmed := strings.TrimSpace(path)
	trimmed = strings.ReplaceAll(trimmed, `\`, "/")
	trimmed = strings.TrimPrefix(trimmed, "/")
	return basePath + trimmed
}

func normalizeStaticAssetBasePath(prefix string) string {
	trimmed := strings.TrimSpace(prefix)
	if trimmed == "" {
		trimmed = defaultStaticAssetBasePath
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	if !strings.HasSuffix(trimmed, "/") {
		trimmed += "/"
	}

	return trimmed
}
