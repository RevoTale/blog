package markdown

import (
	"strings"
	"testing"
)

func TestToHTML_TransformsExternalLinkTokens(t *testing.T) {
	html := string(ToHTML("[external](external_link://a1)", Options{
		TranslateLinks: map[string]string{"a1": "https://example.com/read"},
		RootURL:        "https://revotale.com",
	}))

	if !strings.Contains(html, `href="https://example.com/read"`) {
		t.Fatalf("expected translated external href, got %s", html)
	}
	if !strings.Contains(html, `target="_blank"`) {
		t.Fatalf("expected target blank, got %s", html)
	}
	if !strings.Contains(html, `rel="noopener noreferrer"`) {
		t.Fatalf("expected external rel attrs, got %s", html)
	}
}

func TestToHTML_TransformsInternalLinkTokens(t *testing.T) {
	html := string(ToHTML("[internal](micro_post://n1)", Options{
		TranslateLinks: map[string]string{"n1": "/note/hello-world"},
	}))

	if !strings.Contains(html, `href="/note/hello-world"`) {
		t.Fatalf("expected translated internal href, got %s", html)
	}
	if !strings.Contains(html, `target="_blank"`) {
		t.Fatalf("expected target blank, got %s", html)
	}
	if !strings.Contains(html, `rel="noopener noreferrer"`) {
		t.Fatalf("expected rel attrs for non-domain links, got %s", html)
	}
}

func TestToHTML_NormalizesSameDomainAbsoluteLinks(t *testing.T) {
	html := string(ToHTML("[same](https://revotale.com/note/a?x=1#k)", Options{
		RootURL: "https://revotale.com",
	}))

	if !strings.Contains(html, `href="/note/a?x=1#k"`) {
		t.Fatalf("expected normalized same-domain href, got %s", html)
	}
	if !strings.Contains(html, `target="_blank"`) {
		t.Fatalf("expected target blank, got %s", html)
	}
	if strings.Contains(html, `rel="noopener noreferrer"`) {
		t.Fatalf("did not expect rel attrs for same-domain absolute links, got %s", html)
	}
}

func TestToHTML_HighlightsCodeBlocks(t *testing.T) {
	source := "```go\nfmt.Println(\"hello\")\n```"
	html := string(ToHTML(source, Options{}))

	if !strings.Contains(html, `class="chroma"`) {
		t.Fatalf("expected chroma class for fenced code block, got %s", html)
	}
	if !strings.Contains(html, "Println") {
		t.Fatalf("expected code content in rendered block, got %s", html)
	}
}

func TestToHTML_RendersInlineCodeClass(t *testing.T) {
	html := string(ToHTML("Use `go test ./...` now.", Options{}))

	if !strings.Contains(html, `<code class="inline-code">go test ./...</code>`) {
		t.Fatalf("expected inline code class, got %s", html)
	}
}

func TestExcerpt_RemovesTokenizedMarkdownLinkTargets(t *testing.T) {
	input := "I'm tired of heavy NextJs runtime for a simple blog. " +
		"Rewriting the RevoTale blog to the custom Go + GoTempl framework: " +
		"[https://github.com/RevoTale/blog](external_link://dea8fb62-8df8-4301-b1b3-b30791abeaf8)"
	got := Excerpt(input, 300)

	if strings.Contains(got, "external_link://") {
		t.Fatalf("expected no external_link token in excerpt, got %s", got)
	}
	if !strings.Contains(got, "https://github.com/RevoTale/blog") {
		t.Fatalf("expected human-readable link text to stay in excerpt, got %s", got)
	}
}

func TestExcerpt_TruncatesOnWordBoundary(t *testing.T) {
	got := Excerpt("alpha beta gamma delta", 12)
	if got != "alpha beta..." {
		t.Fatalf("expected graceful word truncation, got %q", got)
	}
}
