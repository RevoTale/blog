package runtime

import (
	"strings"

	"blog/internal/imageloader"
)

type BootstrapConfig struct {
	StaticAssetBasePath string
	ImageLoader         imageloader.Loader
	LovelyEyeScriptURL  string
	LovelyEyeSiteID     string
}

func Initialize(cfg BootstrapConfig) {
	SetStaticAssetBasePath(cfg.StaticAssetBasePath)
	SetImageLoader(cfg.ImageLoader)

	SetLovelyEye(
		strings.TrimSpace(cfg.LovelyEyeScriptURL),
		strings.TrimSpace(cfg.LovelyEyeSiteID),
	)
}
