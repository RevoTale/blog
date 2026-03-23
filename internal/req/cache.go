package req

import (
	"net/http"
	"strings"
)

func SetCacheControl(w http.ResponseWriter, value string) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return
	}
	w.Header().Set("Cache-Control", trimmed)
}

func IsReadMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead
}
