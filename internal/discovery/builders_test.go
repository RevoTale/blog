package discovery

import (
	"context"
	"net/url"
	"testing"
	"time"

	"blog/internal/notes"
	frameworki18n "github.com/RevoTale/no-js/framework/i18n"
	"github.com/stretchr/testify/require"
)

func TestBuildRobotsIncludesSitemap(t *testing.T) {
	t.Parallel()

	document := BuildRobots("https://revotale.com/blog/notes")
	require.Len(t, document.Rules, 1)
	require.Equal(t, "*", document.Rules[0].UserAgent)
	require.Equal(t, []string{"/"}, document.Rules[0].Allow)
	require.Equal(t, []string{"https://revotale.com/blog/notes/sitemap-index"}, document.Sitemaps)
}

func TestFeedListFilterFromQuery(t *testing.T) {
	t.Parallel()

	filter := FeedListFilterFromQuery(url.Values{
		"page":   []string{"2"},
		"author": []string{"l-you"},
		"tag":    []string{"go"},
		"type":   []string{"short"},
		"q":      []string{"build"},
	})

	require.Equal(t, 2, filter.Page)
	require.Equal(t, "l-you", filter.AuthorSlug)
	require.Equal(t, "go", filter.TagName)
	require.Equal(t, notes.NoteTypeShort, filter.Type)
	require.Equal(t, "build", filter.Query)
}

func TestBuildFeedDocumentUsesLocalizedPaths(t *testing.T) {
	t.Parallel()

	document := BuildFeedDocument(
		"https://revotale.com/blog/notes",
		frameworki18n.Config{
			Locales:       []string{"en", "uk"},
			DefaultLocale: "en",
			PrefixMode:    frameworki18n.PrefixAsNeeded,
		},
		"uk",
		[]notes.NoteSummary{
			{
				Slug:           "hello-world",
				Title:          "Hello World",
				Description:    "Hello note",
				PublishedAtISO: "2024-01-02T00:00:00Z",
				Authors:        []notes.Author{{Name: "L You", Slug: "l-you"}},
				Tags:           []notes.Tag{{Name: "go", Title: "Go"}},
			},
		},
	)

	require.Equal(t, "https://revotale.com/blog/notes/uk", document.Link)
	require.Equal(t, "https://revotale.com/blog/notes/feed.xml?locale=uk", document.SelfURL)
	require.Len(t, document.Items, 1)
	require.Equal(t, "https://revotale.com/blog/notes/uk/note/hello-world", document.Items[0].Link)
	require.Equal(t, "L You", document.Items[0].Author)
}

func TestBuildSitemapIDsAndEntriesByID(t *testing.T) {
	t.Parallel()

	service := stubNotesLister{
		listFn: func(
			_ context.Context,
			_ string,
			filter notes.ListFilter,
			_ notes.ListOptions,
		) (notes.NotesListResult, error) {
			if filter.Page == 2 {
				return notes.NotesListResult{
					Notes: []notes.NoteSummary{
						{
							Slug:           "second-note",
							Title:          "Second Note",
							PublishedAtISO: "2024-02-03T00:00:00Z",
						},
					},
				}, nil
			}

			return notes.NotesListResult{
				Notes: []notes.NoteSummary{
					{
						Slug:           "hello-world",
						Title:          "Hello World",
						PublishedAtISO: "2024-01-02T00:00:00Z",
						Attachment:     &notes.Attachment{URL: "/images/hello.png"},
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
				TotalPages: 2,
			}, nil
		},
	}

	i18nConfig := frameworki18n.Config{
		Locales:       []string{"en", "uk"},
		DefaultLocale: "en",
		PrefixMode:    frameworki18n.PrefixAsNeeded,
	}

	ids, err := BuildSitemapIDs(context.Background(), "https://revotale.com/blog/notes", i18nConfig, service, 1, 1)
	require.NoError(t, err)
	require.Len(t, ids, 1+2+2+2)
	require.Equal(t, "root", ids[0].ID)
	require.Equal(t, "https://revotale.com/blog/notes/sitemap.xml", ids[0].Location)
	require.Equal(t, "note:0", ids[1].ID)
	require.Equal(t, "/note/sitemap/0.xml", ids[1].Path)

	noteEntries, err := BuildSitemapEntriesByID(
		context.Background(),
		"https://revotale.com/blog/notes",
		i18nConfig,
		service,
		"note:0",
		1,
		1,
	)
	require.NoError(t, err)
	require.Len(t, noteEntries, 1)
	require.Equal(t, "https://revotale.com/blog/notes/note/hello-world", noteEntries[0].URL)
	require.Equal(t, "weekly", noteEntries[0].ChangeFrequency)
	require.Len(t, noteEntries[0].Images, 1)
	require.Equal(t, "https://revotale.com/blog/notes/images/hello.png", noteEntries[0].Images[0].URL)

	authorEntries, err := BuildSitemapEntriesByID(
		context.Background(),
		"https://revotale.com/blog/notes",
		i18nConfig,
		service,
		"author:0",
		1,
		1,
	)
	require.NoError(t, err)
	require.Len(t, authorEntries, 1)
	require.Equal(t, "https://revotale.com/blog/notes/author/l-you", authorEntries[0].URL)
	require.NotNil(t, authorEntries[0].Priority)
	require.Equal(t, 1.0, *authorEntries[0].Priority)

	rootEntries, err := BuildSitemapEntriesByID(
		context.Background(),
		"https://revotale.com/blog/notes",
		i18nConfig,
		service,
		"root",
		1,
		1,
	)
	require.NoError(t, err)
	require.Len(t, rootEntries, 4)
	for _, entry := range rootEntries {
		require.NotNil(t, entry.LastModified)
		require.WithinDuration(t, time.Now().UTC(), entry.LastModified.UTC(), time.Minute)
	}
}

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
