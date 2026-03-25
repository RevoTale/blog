package req

import (
	"net/http"
)

func SetCacheControl(w http.ResponseWriter, value string) {
	w.Header().Set("Cache-Control", value)
}

func IsReadMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead
}
