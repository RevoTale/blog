package runtime

import (
	"strings"

	"blog/internal/imageloader"
	frameworki18n "github.com/RevoTale/no-js/framework/i18n"
)

type BootstrapConfig struct {
	LocalizationConfig  frameworki18n.Config
	StaticAssetBasePath string
	ImageLoader         imageloader.Loader
	LovelyEyeScriptURL  string
	LovelyEyeSiteID     string
}

func Initialize(cfg BootstrapConfig) {
	SetLocalizationConfig(cfg.LocalizationConfig)
	SetStaticAssetBasePath(cfg.StaticAssetBasePath)
	SetImageLoader(cfg.ImageLoader)

	SetLovelyEye(
		strings.TrimSpace(cfg.LovelyEyeScriptURL),
		strings.TrimSpace(cfg.LovelyEyeSiteID),
	)
}
