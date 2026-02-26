package web

import "embed"

//go:embed all:app
var embeddedAppFS embed.FS
