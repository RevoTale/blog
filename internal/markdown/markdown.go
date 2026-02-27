package markdown

import (
	stdhtml "html"
	"html/template"
	"io"
	"net/url"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/alecthomas/chroma/v2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	md "github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	mdhtml "github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

const (
	externalLinkPrefix = "external_link://"
	internalLinkPrefix = "micro_post://"
)

type Options struct {
	TranslateLinks map[string]string
	RootURL        string
}

const lastGoodBreakRatio = 0.8

var (
	markdownCodeBlockPattern          = regexp.MustCompile("(?s)```.*?```")
	markdownTablePattern              = regexp.MustCompile(`(?m)^\|.*\|.*$`)
	markdownImagePattern              = regexp.MustCompile(`!\[.*?\]\(.*?\)`)
	markdownHorizontalRulePattern     = regexp.MustCompile(`(?m)^---+$`)
	markdownFootnoteDefinitionPattern = regexp.MustCompile(`(?m)^\[\^[^\]]+\]: .*$`)
	markdownFootnoteReferencePattern  = regexp.MustCompile(`\[\^[^\]]+\]`)
	markdownBoldItalicPattern         = regexp.MustCompile(`\*\*\*(.*?)\*\*\*`)
	markdownBoldPattern               = regexp.MustCompile(`\*\*(.*?)\*\*`)
	markdownItalicAsteriskPattern     = regexp.MustCompile(`\*(.*?)\*`)
	markdownItalicUnderscorePattern   = regexp.MustCompile(`_(.*?)_`)
	markdownHeadingPattern            = regexp.MustCompile(`(?m)^#{1,6}\s+(.*?)$`)
	markdownStrikethroughPattern      = regexp.MustCompile(`~~(.*?)~~`)
	markdownInlineCodePattern         = regexp.MustCompile("`(.*?)`")
	markdownLinkPattern               = regexp.MustCompile(`\[(.*?)\]\(.*?\)`)
	markdownBlockquotePattern         = regexp.MustCompile(`(?m)^\s*>\s*(.*?)$`)
	markdownTaskListPattern           = regexp.MustCompile(`(?m)^\s*-\s\[[ x]\]\s+`)
	markdownOrderedListPattern        = regexp.MustCompile(`(?m)^\s*\d+\.\s+`)
	htmlTagPattern                    = regexp.MustCompile(`<[^>]*>`)
)

func ToHTML(input string, opts Options) template.HTML {
	if strings.TrimSpace(input) == "" {
		return template.HTML("")
	}

	p := parser.NewWithExtensions(parser.CommonExtensions | parser.AutoHeadingIDs)
	doc := p.Parse([]byte(input))
	normalizeLinks(doc, opts)

	renderer := mdhtml.NewRenderer(mdhtml.RendererOptions{
		Flags:          mdhtml.CommonFlags | mdhtml.SkipHTML,
		RenderNodeHook: renderNodeHook,
	})

	return template.HTML(md.Render(doc, renderer))
}

func Excerpt(input string, maxChars int) string {
	if maxChars < 1 {
		return ""
	}

	clean := markdownToPlainText(input)
	if clean == "" {
		return ""
	}

	if utf8.RuneCountInString(clean) <= maxChars {
		return clean
	}

	return truncateRunes(clean, maxChars)
}

func markdownToPlainText(markdown string) string {
	text := markdown
	text = markdownCodeBlockPattern.ReplaceAllString(text, " ")
	text = markdownTablePattern.ReplaceAllString(text, " ")
	text = markdownImagePattern.ReplaceAllString(text, " ")
	text = markdownHorizontalRulePattern.ReplaceAllString(text, " ")
	text = markdownFootnoteDefinitionPattern.ReplaceAllString(text, " ")
	text = markdownFootnoteReferencePattern.ReplaceAllString(text, "")

	text = markdownBoldItalicPattern.ReplaceAllString(text, "$1")
	text = markdownBoldPattern.ReplaceAllString(text, "$1")
	text = markdownItalicAsteriskPattern.ReplaceAllString(text, "$1")
	text = markdownItalicUnderscorePattern.ReplaceAllString(text, "$1")
	text = markdownHeadingPattern.ReplaceAllString(text, "\n$1\n")
	text = markdownStrikethroughPattern.ReplaceAllString(text, "$1")
	text = markdownInlineCodePattern.ReplaceAllString(text, "$1")
	text = markdownLinkPattern.ReplaceAllString(text, "$1")
	text = markdownBlockquotePattern.ReplaceAllString(text, "$1")
	text = markdownTaskListPattern.ReplaceAllString(text, "- ")
	text = markdownOrderedListPattern.ReplaceAllString(text, "- ")
	text = htmlTagPattern.ReplaceAllString(text, "")

	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}

	return strings.Join(strings.Fields(text), " ")
}

func truncateRunes(text string, maxChars int) string {
	runes := []rune(text)
	if len(runes) <= maxChars {
		return text
	}

	truncateAt := maxChars
	minBreak := int(float64(maxChars) * lastGoodBreakRatio)
	for idx := maxChars - 1; idx >= minBreak; idx-- {
		if unicode.IsSpace(runes[idx]) {
			truncateAt = idx
			break
		}
	}

	truncated := strings.TrimSpace(string(runes[:truncateAt]))
	if truncated == "" {
		truncated = strings.TrimSpace(string(runes[:maxChars]))
	}

	return truncated + "..."
}

func normalizeLinks(doc ast.Node, opts Options) {
	ast.WalkFunc(doc, func(node ast.Node, entering bool) ast.WalkStatus {
		if !entering {
			return ast.GoToNext
		}

		link, ok := node.(*ast.Link)
		if !ok {
			return ast.GoToNext
		}

		transformedHref := transformLink(string(link.Destination), opts.TranslateLinks)
		normalizedHref, isCurrentWebsite := normalizeCurrentWebsiteLink(transformedHref, opts.RootURL)
		link.Destination = []byte(normalizedHref)
		link.AdditionalAttributes = applyLinkAttributes(link.AdditionalAttributes, isCurrentWebsite)

		return ast.GoToNext
	})
}

func renderNodeHook(writer io.Writer, node ast.Node, entering bool) (ast.WalkStatus, bool) {
	if !entering {
		return ast.GoToNext, false
	}

	switch typedNode := node.(type) {
	case *ast.CodeBlock:
		renderCodeBlock(writer, typedNode)
		return ast.SkipChildren, true
	case *ast.Code:
		renderInlineCode(writer, typedNode)
		return ast.SkipChildren, true
	default:
		return ast.GoToNext, false
	}
}

func renderCodeBlock(writer io.Writer, block *ast.CodeBlock) {
	code := string(block.Literal)
	lexer := pickLexer(codeLanguage(block.Info), code)
	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		renderPlainCodeBlock(writer, code)
		return
	}

	formatter := chromahtml.New(chromahtml.WithClasses(true))
	if err := formatter.Format(writer, styles.Fallback, iterator); err != nil {
		renderPlainCodeBlock(writer, code)
	}
}

func renderInlineCode(writer io.Writer, code *ast.Code) {
	_, _ = io.WriteString(writer, `<code class="inline-code">`)
	_, _ = io.WriteString(writer, stdhtml.EscapeString(string(code.Literal)))
	_, _ = io.WriteString(writer, `</code>`)
}

func renderPlainCodeBlock(writer io.Writer, code string) {
	_, _ = io.WriteString(writer, `<pre class="chroma"><code>`)
	_, _ = io.WriteString(writer, stdhtml.EscapeString(code))
	_, _ = io.WriteString(writer, `</code></pre>`)
}

func pickLexer(language string, code string) chroma.Lexer {
	if language != "" {
		if lexer := lexers.Get(language); lexer != nil {
			return lexer
		}
	}

	if lexer := lexers.Analyse(code); lexer != nil {
		return lexer
	}

	return lexers.Fallback
}

func codeLanguage(info []byte) string {
	trimmed := strings.TrimSpace(string(info))
	if trimmed == "" {
		return ""
	}

	fields := strings.Fields(trimmed)
	if len(fields) == 0 {
		return ""
	}

	return strings.ToLower(fields[0])
}

func transformLink(href string, translateLinks map[string]string) string {
	if href == "" {
		return href
	}

	truncated := href
	if strings.HasPrefix(truncated, externalLinkPrefix) {
		truncated = strings.TrimPrefix(truncated, externalLinkPrefix)
	} else if strings.HasPrefix(truncated, internalLinkPrefix) {
		truncated = strings.TrimPrefix(truncated, internalLinkPrefix)
	}

	if target, ok := translateLinks[truncated]; ok && strings.TrimSpace(target) != "" {
		return target
	}

	return href
}

func normalizeCurrentWebsiteLink(href string, rootURL string) (string, bool) {
	if rootURL == "" || !strings.HasPrefix(href, rootURL) {
		return href, false
	}

	parsed, err := url.Parse(href)
	if err != nil {
		return href, true
	}

	normalized := parsed.Path
	if normalized == "" {
		normalized = "/"
	}
	if parsed.RawQuery != "" {
		normalized += "?" + parsed.RawQuery
	}
	if parsed.Fragment != "" {
		normalized += "#" + parsed.Fragment
	}

	return normalized, true
}

func applyLinkAttributes(existing []string, isCurrentWebsite bool) []string {
	attrs := make([]string, 0, len(existing)+2)
	for _, attr := range existing {
		normalized := strings.ToLower(strings.TrimSpace(attr))
		if strings.HasPrefix(normalized, "target=") || strings.HasPrefix(normalized, "rel=") {
			continue
		}
		attrs = append(attrs, attr)
	}

	attrs = append(attrs, `target="_blank"`)
	if !isCurrentWebsite {
		attrs = append(attrs, `rel="noopener noreferrer"`)
	}

	return attrs
}
