package metagen

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	frameworki18n "blog/framework/i18n"
)

func TestHeadRendersManagedSEOAndDeterministicOrder(t *testing.T) {
	t.Parallel()

	first := Metadata{
		Title:       "Example Title",
		Description: "Example Description",
		Alternates: Alternates{
			Canonical: "https://example.com/note/hello",
			Languages: map[string]string{
				"de": "https://example.com/de/note/hello",
				"en": "https://example.com/note/hello",
			},
			Types: map[string]string{
				"application/rss+xml":  "https://example.com/feed.xml",
				"application/atom+xml": "https://example.com/feed.atom",
			},
		},
		Robots: &Robots{
			Index:  Bool(false),
			Follow: Bool(true),
		},
		OpenGraph: &OpenGraph{
			Type:          "article",
			SiteName:      "Blog",
			Locale:        "en",
			PublishedTime: "2026-03-03T08:00:00Z",
			Authors: []string{
				"https://example.com/authors/b",
				"https://example.com/authors/a",
			},
			Tags: []string{"seo", "blog", "seo"},
			Images: []OpenGraphImage{
				{URL: "https://example.com/images/b.png"},
				{URL: "https://example.com/images/a.png", Alt: "alt-a"},
			},
		},
		Twitter: &Twitter{
			Card:    "summary_large_image",
			Creator: "@example",
			Images: []string{
				"https://example.com/images/z.png",
				"https://example.com/images/a.png",
			},
		},
		Authors: []Author{
			{Name: "Zed", URL: "https://example.com/authors/zed"},
			{Name: "Alice", URL: "https://example.com/authors/alice"},
		},
		Publisher: "RevoTale",
		Pinterest: &Pinterest{RichPin: Bool(true)},
	}

	second := Metadata{
		Title:       first.Title,
		Description: first.Description,
		Alternates: Alternates{
			Canonical: first.Alternates.Canonical,
			Languages: map[string]string{
				"en": "https://example.com/note/hello",
				"de": "https://example.com/de/note/hello",
			},
			Types: map[string]string{
				"application/atom+xml": "https://example.com/feed.atom",
				"application/rss+xml":  "https://example.com/feed.xml",
			},
		},
		Robots: first.Robots,
		OpenGraph: &OpenGraph{
			Type:          "article",
			SiteName:      "Blog",
			Locale:        "en",
			PublishedTime: "2026-03-03T08:00:00Z",
			Authors: []string{
				"https://example.com/authors/a",
				"https://example.com/authors/b",
			},
			Tags: []string{"seo", "blog"},
			Images: []OpenGraphImage{
				{URL: "https://example.com/images/a.png", Alt: "alt-a"},
				{URL: "https://example.com/images/b.png"},
			},
		},
		Twitter: &Twitter{
			Card:    "summary_large_image",
			Creator: "@example",
			Images: []string{
				"https://example.com/images/a.png",
				"https://example.com/images/z.png",
			},
		},
		Authors: []Author{
			{Name: "Alice", URL: "https://example.com/authors/alice"},
			{Name: "Zed", URL: "https://example.com/authors/zed"},
		},
		Publisher: "RevoTale",
		Pinterest: &Pinterest{RichPin: Bool(true)},
	}

	firstHead := renderHeadToString(t, first)
	secondHead := renderHeadToString(t, second)

	if firstHead != secondHead {
		t.Fatalf("expected deterministic rendered head\nfirst:\n%s\nsecond:\n%s", firstHead, secondHead)
	}

	required := []string{
		`<title data-metagen-managed="true">Example Title</title>`,
		`name="description" content="Example Description"`,
		`rel="canonical" href="https://example.com/note/hello"`,
		`hreflang="de" href="https://example.com/de/note/hello"`,
		`property="og:type" content="article"`,
		`property="article:published_time" content="2026-03-03T08:00:00Z"`,
		`property="article:author" content="https://example.com/authors/a"`,
		`property="article:tag" content="blog"`,
		`name="twitter:card" content="summary_large_image"`,
		`name="robots" content="noindex, follow"`,
		`name="author" content="Alice"`,
		`name="pinterest-rich-pin" content="true"`,
	}
	for _, token := range required {
		if !strings.Contains(firstHead, token) {
			t.Fatalf("expected rendered head to contain %q\n%s", token, firstHead)
		}
	}
}

func TestHeadRendersDangerRawHeadVerbatim(t *testing.T) {
	t.Parallel()

	head := renderHeadToString(t, Metadata{
		Title:         "Raw Head",
		DangerRawHead: []string{`<style id="test-style">.x{color:red}</style>`},
	})

	if !strings.Contains(head, `<style id="test-style">.x{color:red}</style>`) {
		t.Fatalf("expected DangerRawHead to be emitted verbatim, got %q", head)
	}
}

func TestMergeAllAppendsDangerRawHeadAndOverridesFields(t *testing.T) {
	t.Parallel()

	parent := Metadata{
		Title:         "Parent",
		Description:   "Parent Description",
		DangerRawHead: []string{"<style>.a{}</style>"},
	}
	child := Metadata{
		Title:         "Child",
		DangerRawHead: []string{"<script>window.x=1</script>"},
	}

	merged := MergeAll(parent, child)
	if merged.Title != "Child" {
		t.Fatalf("expected child title override, got %q", merged.Title)
	}
	if merged.Description != "Parent Description" {
		t.Fatalf("expected parent description inheritance, got %q", merged.Description)
	}
	if len(merged.DangerRawHead) != 2 {
		t.Fatalf("expected merged raw head length 2, got %d", len(merged.DangerRawHead))
	}
	if merged.DangerRawHead[0] != "<style>.a{}</style>" {
		t.Fatalf("expected parent raw head first, got %q", merged.DangerRawHead[0])
	}
	if merged.DangerRawHead[1] != "<script>window.x=1</script>" {
		t.Fatalf("expected child raw head second, got %q", merged.DangerRawHead[1])
	}
}

func TestBuildAlternatesPrefixAsNeeded(t *testing.T) {
	t.Parallel()

	alternates, err := BuildAlternates(
		"https://example.com/blog",
		frameworki18n.Config{
			Locales:       []string{"en", "de"},
			DefaultLocale: "en",
			PrefixMode:    frameworki18n.PrefixAsNeeded,
		},
		"de",
		"/note/hello?tag=go&__live=navigation",
		map[string]string{
			"application/rss+xml":  "/feed.xml?__live=navigation",
			"application/atom+xml": "https://cdn.example.com/feed.atom?__live=navigation",
		},
	)
	if err != nil {
		t.Fatalf("build alternates: %v", err)
	}

	if alternates.Canonical != "https://example.com/blog/de/note/hello?tag=go" {
		t.Fatalf("canonical: expected %q, got %q", "https://example.com/blog/de/note/hello?tag=go", alternates.Canonical)
	}
	if got := alternates.Languages["en"]; got != "https://example.com/blog/note/hello?tag=go" {
		t.Fatalf("en alternate: expected %q, got %q", "https://example.com/blog/note/hello?tag=go", got)
	}
	if got := alternates.Languages["de"]; got != "https://example.com/blog/de/note/hello?tag=go" {
		t.Fatalf("de alternate: expected %q, got %q", "https://example.com/blog/de/note/hello?tag=go", got)
	}
	if got := alternates.Types["application/rss+xml"]; got != "https://example.com/blog/feed.xml" {
		t.Fatalf("rss alternate: expected %q, got %q", "https://example.com/blog/feed.xml", got)
	}
	if got := alternates.Types["application/atom+xml"]; got != "https://cdn.example.com/feed.atom" {
		t.Fatalf("atom alternate: expected %q, got %q", "https://cdn.example.com/feed.atom", got)
	}
}

func TestBuildHTMXPatchAndWriteHeaders(t *testing.T) {
	t.Parallel()

	patch, err := BuildHTMXPatch(Metadata{
		Title:       "Notes",
		Description: "A notes feed",
	})
	if err != nil {
		t.Fatalf("build htmx patch: %v", err)
	}
	if patch.Title != "Notes" {
		t.Fatalf("patch title: expected %q, got %q", "Notes", patch.Title)
	}
	if strings.Contains(patch.Head, "<title") {
		t.Fatalf("htmx patch should not include <title>, got %q", patch.Head)
	}
	if !strings.Contains(patch.Head, `name="description"`) {
		t.Fatalf("htmx patch should include managed head metadata, got %q", patch.Head)
	}

	recorder := httptest.NewRecorder()
	if err := WriteHTMXHeaders(recorder, patch); err != nil {
		t.Fatalf("write htmx headers: %v", err)
	}

	rawHeader := recorder.Header().Get("HX-Trigger-After-Settle")
	if strings.TrimSpace(rawHeader) == "" {
		t.Fatal("expected HX-Trigger-After-Settle header")
	}

	payload := make(map[string]Patch)
	if err := json.Unmarshal([]byte(rawHeader), &payload); err != nil {
		t.Fatalf("parse htmx header payload: %v", err)
	}
	eventPayload, ok := payload[HTMXPatchEvent]
	if !ok {
		t.Fatalf("expected event %q in header payload", HTMXPatchEvent)
	}
	if eventPayload.Title != "Notes" {
		t.Fatalf("payload title: expected %q, got %q", "Notes", eventPayload.Title)
	}
}

func TestWriteHTMXHeadersMergesJSONPayload(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	recorder.Header().Set("HX-Trigger-After-Settle", `{"existing":{"ok":true}}`)

	err := WriteHTMXHeaders(recorder, Patch{Title: "Merged"})
	if err != nil {
		t.Fatalf("write merged headers: %v", err)
	}

	out := make(map[string]json.RawMessage)
	if err := json.Unmarshal([]byte(recorder.Header().Get("HX-Trigger-After-Settle")), &out); err != nil {
		t.Fatalf("parse merged header: %v", err)
	}
	if _, ok := out["existing"]; !ok {
		t.Fatalf("expected existing event to remain in merged header")
	}
	if _, ok := out[HTMXPatchEvent]; !ok {
		t.Fatalf("expected %q event in merged header", HTMXPatchEvent)
	}
}

func renderHeadToString(t *testing.T, meta Metadata) string {
	t.Helper()

	component := Head(meta)
	var buffer bytes.Buffer
	if err := component.Render(context.Background(), &buffer); err != nil {
		t.Fatalf("render head: %v", err)
	}
	return strings.TrimSpace(buffer.String())
}

func TestWriteHTMXHeadersNilResponseWriter(t *testing.T) {
	t.Parallel()

	var writer http.ResponseWriter
	if err := WriteHTMXHeaders(writer, Patch{Title: "noop"}); err != nil {
		t.Fatalf("expected nil writer to be ignored, got %v", err)
	}
}
