package runtime

import (
	"strings"
	"sync/atomic"
)

var lovelyEyeScriptURLValue atomic.Value
var lovelyEyeSiteIDValue atomic.Value

func SetLovelyEye(scriptURL string, siteID string) {
	scriptURL = strings.TrimSpace(scriptURL)
	siteID = strings.TrimSpace(siteID)

	if scriptURL == "" || siteID == "" {
		lovelyEyeScriptURLValue.Store("")
		lovelyEyeSiteIDValue.Store("")
		return
	}

	lovelyEyeScriptURLValue.Store(scriptURL)
	lovelyEyeSiteIDValue.Store(siteID)
}

func LovelyEyeEnabled() bool {
	return LovelyEyeScriptURL() != "" && LovelyEyeSiteID() != ""
}

func LovelyEyeScriptURL() string {
	value, _ := lovelyEyeScriptURLValue.Load().(string)
	return strings.TrimSpace(value)
}

func LovelyEyeSiteID() string {
	value, _ := lovelyEyeSiteIDValue.Load().(string)
	return strings.TrimSpace(value)
}
