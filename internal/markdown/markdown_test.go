package markdown

import (
	"testing"

	"blog/internal/imageloader"
	"github.com/stretchr/testify/require"
)

func TestToHTML_TransformsExternalLinkTokens(t *testing.T) {
	html := string(ToHTML("[external](external_link://a1)", Options{
		TranslateLinks: map[string]string{"a1": "https://example.com/read"},
		RootURL:        "https://revotale.com",
	}))

	require.Contains(t, html, `href="https://example.com/read"`)
	require.Contains(t, html, `target="_blank"`)
	require.Contains(t, html, `rel="noopener noreferrer"`)
}

func TestToHTML_TransformsInternalLinkTokens(t *testing.T) {
	html := string(ToHTML("[internal](micro_post://n1)", Options{
		TranslateLinks: map[string]string{"n1": "/note/hello-world"},
	}))

	require.Contains(t, html, `href="/note/hello-world"`)
	require.Contains(t, html, `target="_blank"`)
	require.Contains(t, html, `rel="noopener noreferrer"`)
}

func TestToHTML_NormalizesSameDomainAbsoluteLinks(t *testing.T) {
	html := string(ToHTML("[same](https://revotale.com/note/a?x=1#k)", Options{
		RootURL: "https://revotale.com",
	}))

	require.Contains(t, html, `href="/note/a?x=1#k"`)
	require.Contains(t, html, `target="_blank"`)
	require.NotContains(t, html, `rel="noopener noreferrer"`)
}

func TestToHTML_NormalizesSameDomainAbsoluteLinksAcrossConfiguredRoots(t *testing.T) {
	html := string(ToHTML("[same](https://revotale.com/note/a?x=1#k)", Options{
		RootURLs: []string{"https://mirror.example", "https://revotale.com"},
	}))

	require.Contains(t, html, `href="/note/a?x=1#k"`)
	require.Contains(t, html, `target="_blank"`)
	require.NotContains(t, html, `rel="noopener noreferrer"`)
}

func TestToHTML_HighlightsCodeBlocks(t *testing.T) {
	source := "```go\nfmt.Println(\"hello\")\n```"
	html := string(ToHTML(source, Options{}))

	require.Contains(t, html, `class="chroma"`)
	require.Contains(t, html, `class="code-copy-button"`)
	require.Contains(t, html, `class="code-block-language">go</p>`)
	require.Contains(t, html, `class="code-copy-source"`)
	require.Contains(t, html, "Println")
}

func TestToHTML_UsesPlainTextLabelWhenCodeLanguageIsMissing(t *testing.T) {
	source := "```\nfmt.Println(\"hello\")\n```"
	html := string(ToHTML(source, Options{}))

	require.Contains(t, html, `class="code-block-language">plain text</p>`)
}

func TestToHTML_RendersInlineCodeClass(t *testing.T) {
	html := string(ToHTML("Use `go test ./...` now.", Options{}))

	require.Contains(t, html, `<code class="inline-code">go test ./...</code>`)
}

func TestExcerpt_RemovesTokenizedMarkdownLinkTargets(t *testing.T) {
	input := "I'm tired of heavy NextJs runtime for a simple blog. " +
		"Rewriting the RevoTale blog to the custom Go + GoTempl framework: " +
		"[https://github.com/RevoTale/blog](external_link://dea8fb62-8df8-4301-b1b3-b30791abeaf8)"
	got := Excerpt(input, 300)

	require.NotContains(t, got, "external_link://")
	require.Contains(t, got, "https://github.com/RevoTale/blog")
}

func TestExcerpt_TruncatesOnWordBoundary(t *testing.T) {
	got := Excerpt("alpha beta gamma delta", 12)
	require.Equal(t, "alpha beta...", got)
}

func TestExcerpt_ReplacesSpecialMarkdownBlocksWithLabels(t *testing.T) {
	input := "" +
		"before\n" +
		"```go\nfmt.Println(\"x\")\n```\n" +
		"![img](https://example.com/p.png)\n" +
		"| a | b |\n" +
		"| - | - |\n" +
		"after"

	got := Excerpt(input, 500)

	require.Contains(t, got, "[code block]")
	require.Contains(t, got, "[image]")
	require.Contains(t, got, "[table]")
	require.NotContains(t, got, "PHCODEBLOCK")
}

func TestExcerpt_DoesNotCutPlaceholderToken(t *testing.T) {
	got := Excerpt("alpha ![img](https://example.com/p.png) omega", 10)
	require.Equal(t, "alpha...", got)
}

func TestToHTML_TransformsImageSourcesWithLoader(t *testing.T) {
	t.Parallel()

	html := string(ToHTML(
		"![hero image](/images/hero.webp)",
		Options{
			ImageLoader: imageloader.New(true),
		},
	))

	require.Contains(t, html, `src="/cdn/image/blog/1080/images/hero.webp"`)
	require.Contains(t, html, `srcset="/cdn/image/blog/32/images/hero.webp 32w`)
	require.Contains(t, html, `/cdn/image/blog/1080/images/hero.webp 1080w`)
	require.Contains(t, html, `sizes="(max-width: 660px) 100vw, 672px"`)
}

func TestToHTML_DemotesHeadingsToAvoidH1(t *testing.T) {
	t.Parallel()

	html := string(ToHTML("# Main title\n\n## Section title\n\n###### Small title", Options{}))

	require.NotContains(t, html, "<h1")
	require.Contains(t, html, `<h2 id="main-title">Main title</h2>`)
	require.Contains(t, html, `<h3 id="section-title">Section title</h3>`)
	require.Contains(t, html, `<h6 id="small-title">Small title</h6>`)
}
