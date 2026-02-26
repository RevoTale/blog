package markdown

import (
	"bytes"
	"html/template"
	"strings"
	"sync"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/styles"
)

const (
	chromaLightStyle = "github"
	chromaDarkStyle  = "monokai"
)

var (
	chromaCSSOnce sync.Once
	chromaCSS     template.CSS
)

func ChromaCSS() template.CSS {
	chromaCSSOnce.Do(func() {
		chromaCSS = template.CSS(buildChromaCSS())
	})

	return chromaCSS
}

func buildChromaCSS() string {
	lightCSS := buildSingleStyleCSS(chromaLightStyle)
	darkCSS := buildSingleStyleCSS(chromaDarkStyle)

	var out strings.Builder
	if lightCSS != "" {
		out.WriteString("@media (prefers-color-scheme: light) {\n")
		out.WriteString(lightCSS)
		out.WriteString("}\n")
	}
	if darkCSS != "" {
		out.WriteString("@media (prefers-color-scheme: dark) {\n")
		out.WriteString(darkCSS)
		out.WriteString("}\n")
	}

	return out.String()
}

func buildSingleStyleCSS(styleName string) string {
	style := styles.Get(styleName)
	if style == nil {
		style = styles.Fallback
	}

	formatter := chromahtml.New(chromahtml.WithClasses(true))
	var buffer bytes.Buffer
	if err := formatter.WriteCSS(&buffer, style); err != nil {
		return ""
	}

	return buffer.String()
}
