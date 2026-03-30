package discovery

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"blog/internal/notes"
	frameworki18n "github.com/RevoTale/no-js/framework/i18n"
	"github.com/stretchr/testify/require"
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

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Header().Get("Content-Type"), "application/rss+xml")
	require.Equal(t, defaultRSSCachePolicy, rec.Header().Get("Cache-Control"))
	body := rec.Body.String()
	require.Contains(t, body, "<rss")
	require.Contains(t, body, "https://revotale.com/blog/notes/uk/note/hello-world")
	require.Contains(t, body, "https://revotale.com/blog/notes/feed.xml?locale=uk")

	recFallback := httptest.NewRecorder()
	handler.ServeHTTP(recFallback, httptest.NewRequest(http.MethodGet, "/feed.xml?locale=it", nil))
	require.Equal(t, http.StatusOK, recFallback.Code)
	fallbackBody := recFallback.Body.String()
	require.Contains(t, fallbackBody, "https://revotale.com/blog/notes/note/hello-world")
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

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "uk", gotLocale)
	require.Equal(t, 2, gotFilter.Page)
	require.Equal(t, "l-you", gotFilter.AuthorSlug)
	require.Equal(t, "go", gotFilter.TagName)
	require.Equal(t, notes.NoteTypeShort, gotFilter.Type)
	require.Equal(t, "build", gotFilter.Query)
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
	require.Equal(t, http.StatusOK, recRoot.Code)
	require.Contains(t, recRoot.Header().Get("Content-Type"), "application/xml")
	require.Equal(t, defaultSitemapCachePolicy, recRoot.Header().Get("Cache-Control"))
	rootBody := recRoot.Body.String()
	require.Contains(t, rootBody, "<loc>https://revotale.com/blog/notes</loc>")
	require.Contains(t, rootBody, `hreflang="uk" href="https://revotale.com/blog/notes/uk"`)

	recIndex := httptest.NewRecorder()
	handler.ServeHTTP(recIndex, httptest.NewRequest(http.MethodGet, "/sitemap-index", nil))
	require.Equal(t, http.StatusOK, recIndex.Code)
	require.Equal(t, defaultSitemapIndexCachePolicy, recIndex.Header().Get("Cache-Control"))
	indexBody := recIndex.Body.String()
	for _, mustContain := range []string{
		"https://revotale.com/blog/notes/sitemap.xml",
		"https://revotale.com/blog/notes/note/sitemap/0.xml",
		"https://revotale.com/blog/notes/note/sitemap/1.xml",
		"https://revotale.com/blog/notes/author/sitemap/0.xml",
		"https://revotale.com/blog/notes/notes/sitemap/0.xml",
	} {
		require.Contains(t, indexBody, mustContain)
	}

	recIndexXML := httptest.NewRecorder()
	handler.ServeHTTP(recIndexXML, httptest.NewRequest(http.MethodGet, "/sitemap-index.xml", nil))
	require.Equal(t, http.StatusOK, recIndexXML.Code)

	recNotesChunk := httptest.NewRecorder()
	handler.ServeHTTP(recNotesChunk, httptest.NewRequest(http.MethodGet, "/note/sitemap/0.xml", nil))
	require.Equal(t, http.StatusOK, recNotesChunk.Code)
	noteChunkBody := recNotesChunk.Body.String()
	require.Contains(t, noteChunkBody, "<loc>https://revotale.com/blog/notes/note/hello-world</loc>")
	require.Contains(t, noteChunkBody, "<image:loc>https://revotale.com/blog/notes/images/hello.png</image:loc>")

	recAuthorsChunk := httptest.NewRecorder()
	handler.ServeHTTP(recAuthorsChunk, httptest.NewRequest(http.MethodGet, "/author/sitemap/0.xml", nil))
	require.Equal(t, http.StatusOK, recAuthorsChunk.Code)
	require.Contains(t, recAuthorsChunk.Body.String(), "https://revotale.com/blog/notes/author/l-you")

	recTagsChunk := httptest.NewRecorder()
	handler.ServeHTTP(recTagsChunk, httptest.NewRequest(http.MethodGet, "/notes/sitemap/0.xml", nil))
	require.Equal(t, http.StatusOK, recTagsChunk.Code)
	require.Contains(t, recTagsChunk.Body.String(), "https://revotale.com/blog/notes/tag/go")
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
	require.Equal(t, http.StatusMethodNotAllowed, recMethod.Code)

	recInvalidChunk := httptest.NewRecorder()
	handler.ServeHTTP(recInvalidChunk, httptest.NewRequest(http.MethodGet, "/note/sitemap/invalid.xml", nil))
	require.Equal(t, http.StatusAccepted, recInvalidChunk.Code)

	recUnknown := httptest.NewRecorder()
	handler.ServeHTTP(recUnknown, httptest.NewRequest(http.MethodGet, "/unknown", nil))
	require.Equal(t, http.StatusAccepted, recUnknown.Code)
	require.GreaterOrEqual(t, nextCalls, 2)
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
