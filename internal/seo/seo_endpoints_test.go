package seo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"blog/internal/notes"
	frameworki18n "github.com/RevoTale/no-js/framework/i18n"
)

const seoTestRootURL = "https://revotale.com/blog/notes"

type stubNotesLister struct {
	listFn func(
		ctx context.Context,
		locale string,
		filter notes.ListFilter,
		options notes.ListOptions,
	) (notes.NotesListResult, error)
}

func (stub stubNotesLister) ListNotes(
	ctx context.Context,
	locale string,
	filter notes.ListFilter,
	options notes.ListOptions,
) (notes.NotesListResult, error) {
	if stub.listFn == nil {
		return notes.NotesListResult{}, nil
	}
	return stub.listFn(ctx, locale, filter, options)
}

func TestWithFeedAndSitemapEndpointsRSS(t *testing.T) {
	t.Parallel()

	handler := testSEOEndpointsHandler(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
		newStubSEOListService(),
	)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/feed.xml?locale=uk", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("rss status: expected %d, got %d", http.StatusOK, rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "application/rss+xml") {
		t.Fatalf("rss content-type: expected rss+xml, got %q", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != defaultRSSCachePolicy {
		t.Fatalf("rss cache-control: expected %q, got %q", defaultRSSCachePolicy, got)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<rss") {
		t.Fatalf("rss payload should include rss root element")
	}
	if !strings.Contains(body, "https://revotale.com/blog/notes/uk/note/hello-world") {
		t.Fatalf("rss payload should include localized note URL")
	}
	if !strings.Contains(body, "https://revotale.com/blog/notes/feed.xml?locale=uk") {
		t.Fatalf("rss payload should include self feed URL with locale query")
	}

	recFallback := httptest.NewRecorder()
	handler.ServeHTTP(recFallback, httptest.NewRequest(http.MethodGet, "/feed.xml?locale=it", nil))
	if recFallback.Code != http.StatusOK {
		t.Fatalf("rss fallback status: expected %d, got %d", http.StatusOK, recFallback.Code)
	}
	fallbackBody := recFallback.Body.String()
	if !strings.Contains(fallbackBody, "https://revotale.com/blog/notes/note/hello-world") {
		t.Fatalf("rss fallback should use default locale path for unsupported locale")
	}
}

func TestWithFeedAndSitemapEndpointsRSSUsesFilters(t *testing.T) {
	t.Parallel()

	var gotLocale string
	var gotFilter notes.ListFilter

	handler := testSEOEndpointsHandler(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
		stubNotesLister{
			listFn: func(
				_ context.Context,
				locale string,
				filter notes.ListFilter,
				_ notes.ListOptions,
			) (notes.NotesListResult, error) {
				gotLocale = locale
				gotFilter = filter
				return notes.NotesListResult{
					Notes: []notes.NoteSummary{
						{Slug: "hello-world", Title: "Hello"},
					},
				}, nil
			},
		},
	)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(
		rec,
		httptest.NewRequest(
			http.MethodGet,
			"/feed.xml?locale=uk&page=2&author=l-you&tag=go&type=short&q=build",
			nil,
		),
	)

	if rec.Code != http.StatusOK {
		t.Fatalf("rss status: expected %d, got %d", http.StatusOK, rec.Code)
	}
	if gotLocale != "uk" {
		t.Fatalf("rss locale: expected %q, got %q", "uk", gotLocale)
	}
	if gotFilter.Page != 2 {
		t.Fatalf("rss filter page: expected %d, got %d", 2, gotFilter.Page)
	}
	if gotFilter.AuthorSlug != "l-you" {
		t.Fatalf("rss filter author: expected %q, got %q", "l-you", gotFilter.AuthorSlug)
	}
	if gotFilter.TagName != "go" {
		t.Fatalf("rss filter tag: expected %q, got %q", "go", gotFilter.TagName)
	}
	if gotFilter.Type != notes.NoteTypeShort {
		t.Fatalf("rss filter type: expected %q, got %q", notes.NoteTypeShort, gotFilter.Type)
	}
	if gotFilter.Query != "build" {
		t.Fatalf("rss filter q: expected %q, got %q", "build", gotFilter.Query)
	}
}

func TestWithFeedAndSitemapEndpointsSitemaps(t *testing.T) {
	t.Parallel()

	handler := testSEOEndpointsHandler(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
		newStubSEOListService(),
	)

	recRoot := httptest.NewRecorder()
	handler.ServeHTTP(recRoot, httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil))
	if recRoot.Code != http.StatusOK {
		t.Fatalf("sitemap status: expected %d, got %d", http.StatusOK, recRoot.Code)
	}
	if got := recRoot.Header().Get("Content-Type"); !strings.Contains(got, "application/xml") {
		t.Fatalf("sitemap content-type: expected xml, got %q", got)
	}
	if got := recRoot.Header().Get("Cache-Control"); got != defaultSitemapCachePolicy {
		t.Fatalf("sitemap cache-control: expected %q, got %q", defaultSitemapCachePolicy, got)
	}
	rootBody := recRoot.Body.String()
	if !strings.Contains(rootBody, "<loc>https://revotale.com/blog/notes</loc>") {
		t.Fatalf("root sitemap should include canonical home URL")
	}
	if !strings.Contains(rootBody, `hreflang="uk" href="https://revotale.com/blog/notes/uk"`) {
		t.Fatalf("root sitemap should include hreflang alternates")
	}

	recIndex := httptest.NewRecorder()
	handler.ServeHTTP(recIndex, httptest.NewRequest(http.MethodGet, "/sitemap-index", nil))
	if recIndex.Code != http.StatusOK {
		t.Fatalf("sitemap index status: expected %d, got %d", http.StatusOK, recIndex.Code)
	}
	if got := recIndex.Header().Get("Cache-Control"); got != defaultSitemapIndexCachePolicy {
		t.Fatalf("sitemap index cache-control: expected %q, got %q", defaultSitemapIndexCachePolicy, got)
	}
	indexBody := recIndex.Body.String()
	for _, mustContain := range []string{
		"https://revotale.com/blog/notes/sitemap.xml",
		"https://revotale.com/blog/notes/note/sitemap/0.xml",
		"https://revotale.com/blog/notes/note/sitemap/1.xml",
		"https://revotale.com/blog/notes/author/sitemap/0.xml",
		"https://revotale.com/blog/notes/notes/sitemap/0.xml",
	} {
		if !strings.Contains(indexBody, mustContain) {
			t.Fatalf("sitemap index missing %q", mustContain)
		}
	}

	recIndexXML := httptest.NewRecorder()
	handler.ServeHTTP(recIndexXML, httptest.NewRequest(http.MethodGet, "/sitemap-index.xml", nil))
	if recIndexXML.Code != http.StatusOK {
		t.Fatalf("sitemap index xml alias status: expected %d, got %d", http.StatusOK, recIndexXML.Code)
	}

	recNotesChunk := httptest.NewRecorder()
	handler.ServeHTTP(recNotesChunk, httptest.NewRequest(http.MethodGet, "/note/sitemap/0.xml", nil))
	if recNotesChunk.Code != http.StatusOK {
		t.Fatalf("note chunk sitemap status: expected %d, got %d", http.StatusOK, recNotesChunk.Code)
	}
	noteChunkBody := recNotesChunk.Body.String()
	if !strings.Contains(noteChunkBody, "<loc>https://revotale.com/blog/notes/note/hello-world</loc>") {
		t.Fatalf("note chunk sitemap should include first note URL")
	}
	if !strings.Contains(noteChunkBody, "<image:loc>https://revotale.com/blog/notes/images/hello.png</image:loc>") {
		t.Fatalf("note chunk sitemap should include note image URL")
	}

	recAuthorsChunk := httptest.NewRecorder()
	handler.ServeHTTP(recAuthorsChunk, httptest.NewRequest(http.MethodGet, "/author/sitemap/0.xml", nil))
	if recAuthorsChunk.Code != http.StatusOK {
		t.Fatalf("author chunk sitemap status: expected %d, got %d", http.StatusOK, recAuthorsChunk.Code)
	}
	if body := recAuthorsChunk.Body.String(); !strings.Contains(body, "https://revotale.com/blog/notes/author/l-you") {
		t.Fatalf("author chunk sitemap should include author URL")
	}

	recTagsChunk := httptest.NewRecorder()
	handler.ServeHTTP(recTagsChunk, httptest.NewRequest(http.MethodGet, "/notes/sitemap/0.xml", nil))
	if recTagsChunk.Code != http.StatusOK {
		t.Fatalf("tags chunk sitemap status: expected %d, got %d", http.StatusOK, recTagsChunk.Code)
	}
	if body := recTagsChunk.Body.String(); !strings.Contains(body, "https://revotale.com/blog/notes/tag/go") {
		t.Fatalf("tags chunk sitemap should include tag URL")
	}
}

func TestWithFeedAndSitemapEndpointsMethodAndDelegation(t *testing.T) {
	t.Parallel()

	nextCalls := 0
	handler := testSEOEndpointsHandler(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			nextCalls++
			w.WriteHeader(http.StatusAccepted)
		}),
		newStubSEOListService(),
	)

	recMethod := httptest.NewRecorder()
	handler.ServeHTTP(recMethod, httptest.NewRequest(http.MethodPost, "/feed.xml", nil))
	if recMethod.Code != http.StatusMethodNotAllowed {
		t.Fatalf("feed method status: expected %d, got %d", http.StatusMethodNotAllowed, recMethod.Code)
	}

	recInvalidChunk := httptest.NewRecorder()
	handler.ServeHTTP(recInvalidChunk, httptest.NewRequest(http.MethodGet, "/note/sitemap/invalid.xml", nil))
	if recInvalidChunk.Code != http.StatusAccepted {
		t.Fatalf("invalid chunk path should delegate: expected %d, got %d", http.StatusAccepted, recInvalidChunk.Code)
	}

	recUnknown := httptest.NewRecorder()
	handler.ServeHTTP(recUnknown, httptest.NewRequest(http.MethodGet, "/unknown", nil))
	if recUnknown.Code != http.StatusAccepted {
		t.Fatalf("unknown path should delegate: expected %d, got %d", http.StatusAccepted, recUnknown.Code)
	}
	if nextCalls < 2 {
		t.Fatalf("expected delegated requests to call next handler, got %d calls", nextCalls)
	}
}

func testSEOEndpointsHandler(next http.Handler, lister notesLister) http.Handler {
	return WithFeedAndSitemapEndpoints(next, FeedAndSitemapConfig{
		RootURL: seoTestRootURL,
		I18nConfig: frameworki18n.Config{
			Locales:       []string{"en", "uk"},
			DefaultLocale: "en",
			PrefixMode:    frameworki18n.PrefixAsNeeded,
		},
		Notes: lister,
	})
}

func newStubSEOListService() notesLister {
	pageOne := notes.NotesListResult{
		Notes: []notes.NoteSummary{
			{
				Slug:           "hello-world",
				Title:          "Hello World",
				Description:    "Hello note",
				PublishedAtISO: "2024-01-02T00:00:00Z",
				Attachment: &notes.Attachment{
					URL: "/images/hello.png",
				},
				Authors: []notes.Author{
					{Name: "L You", Slug: "l-you"},
				},
				Tags: []notes.Tag{
					{Name: "go", Title: "Go"},
				},
			},
		},
		Authors: []notes.Author{
			{Name: "L You", Slug: "l-you"},
			{Name: "Zed", Slug: "zed"},
		},
		Tags: []notes.Tag{
			{Name: "go", Title: "Go"},
			{Name: "rust", Title: "Rust"},
		},
		Page:       1,
		TotalPages: 2,
	}

	pageTwo := notes.NotesListResult{
		Notes: []notes.NoteSummary{
			{
				Slug:           "second-note",
				Title:          "Second Note",
				Description:    "Second note",
				PublishedAtISO: "2024-02-03T00:00:00Z",
				Authors: []notes.Author{
					{Name: "Zed", Slug: "zed"},
				},
				Tags: []notes.Tag{
					{Name: "rust", Title: "Rust"},
				},
			},
		},
		Authors:    pageOne.Authors,
		Tags:       pageOne.Tags,
		Page:       2,
		TotalPages: 2,
	}

	return stubNotesLister{
		listFn: func(
			_ context.Context,
			_ string,
			filter notes.ListFilter,
			_ notes.ListOptions,
		) (notes.NotesListResult, error) {
			switch filter.Page {
			case 2:
				return pageTwo, nil
			default:
				return pageOne, nil
			}
		},
	}
}
