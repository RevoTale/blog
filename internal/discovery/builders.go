package discovery

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"blog/internal/notes"
	frameworkdiscovery "github.com/RevoTale/no-js/framework/discovery"
	frameworki18n "github.com/RevoTale/no-js/framework/i18n"
)

func BuildRobots(rootURL string) frameworkdiscovery.Robots {
	document := frameworkdiscovery.Robots{
		Rules: []frameworkdiscovery.RobotsRule{
			{
				UserAgent: "*",
				Allow:     []string{"/"},
			},
		},
	}

	trimmedRoot := strings.TrimSpace(rootURL)
	if trimmedRoot != "" {
		document.Sitemaps = []string{joinRootAndPath(trimmedRoot, sitemapIndexPath)}
	}

	return document
}

func BuildFeedDocument(
	rootURL string,
	i18nConfig frameworki18n.Config,
	locale string,
	noteItems []notes.NoteSummary,
) frameworkdiscovery.FeedDocument {
	homeURL := joinRootAndPath(rootURL, frameworki18n.LocalizePath(i18nConfig, locale, routePathRoot))
	feedURL := joinRootAndPath(rootURL, rssEndpointPath) + "?" + queryParamLocale + "=" + url.QueryEscape(locale)

	items := make([]frameworkdiscovery.FeedItem, 0, len(noteItems))
	for _, note := range noteItems {
		slug := strings.TrimSpace(note.Slug)
		if slug == "" {
			continue
		}

		link := joinRootAndPath(
			rootURL,
			frameworki18n.LocalizePath(i18nConfig, locale, routePathNote+url.PathEscape(slug)),
		)
		title := firstNonEmpty(note.Title, note.MetaTitle, "Untitled Note")
		description := firstNonEmpty(note.Description, note.Excerpt)
		author := "RevoTale"
		if len(note.Authors) > 0 {
			names := make([]string, 0, len(note.Authors))
			for _, candidate := range note.Authors {
				name := strings.TrimSpace(candidate.Name)
				if name == "" {
					continue
				}
				names = append(names, name)
			}
			if len(names) > 0 {
				author = strings.Join(names, ", ")
			}
		}
		categories := make([]string, 0, len(note.Tags))
		for _, tag := range note.Tags {
			name := firstNonEmpty(tag.Name, tag.Title)
			if name == "" {
				continue
			}
			categories = append(categories, name)
		}

		items = append(items, frameworkdiscovery.FeedItem{
			Title:       title,
			Link:        link,
			GUID:        link,
			Description: description,
			Author:      author,
			PublishedAt: parseRFC3339Pointer(note.PublishedAtISO),
			Categories:  categories,
		})
	}

	lastBuildDate := time.Now().UTC()
	if len(noteItems) > 0 {
		if parsed := parseRFC3339Pointer(noteItems[0].PublishedAtISO); parsed != nil {
			lastBuildDate = parsed.UTC()
		}
	}

	return frameworkdiscovery.FeedDocument{
		Title:         "RevoTale Notes",
		Link:          homeURL,
		Description:   "Latest notes and micro posts from RevoTale",
		Language:      locale,
		LastBuildDate: &lastBuildDate,
		Generator:     "RevoTale RSS Generator",
		Copyright:     fmt.Sprintf("© %d RevoTale", time.Now().UTC().Year()),
		SelfURL:       feedURL,
		Items:         items,
	}
}

func FeedListFilterFromQuery(query url.Values) notes.ListFilter {
	return rssListFilterFromQuery(query)
}

func BuildRootSitemapEntries(
	rootURL string,
	i18nConfig frameworki18n.Config,
) ([]frameworkdiscovery.SitemapEntry, error) {
	now := time.Now().UTC()
	paths := []string{routePathRoot, routePathChannels, routePathTales, routePathMicroTales}
	entries := make([]frameworkdiscovery.SitemapEntry, 0, len(paths))
	for _, pathValue := range paths {
		entry, err := sitemapEntryForPath(rootURL, i18nConfig, pathValue)
		if err != nil {
			continue
		}
		out := toDiscoverySitemapEntry(entry)
		out.LastModified = &now
		out.ChangeFrequency = "weekly"
		entries = append(entries, out)
	}
	return entries, nil
}

func BuildSitemapIDs(
	ctx context.Context,
	rootURL string,
	i18nConfig frameworki18n.Config,
	service notesLister,
	authorsPageSize int,
	tagsPageSize int,
) ([]frameworkdiscovery.SitemapID, error) {
	if service == nil {
		return nil, nil
	}

	if authorsPageSize < 1 {
		authorsPageSize = defaultSitemapAuthorsPageSize
	}
	if tagsPageSize < 1 {
		tagsPageSize = defaultSitemapTagsPageSize
	}

	baseResult, err := service.ListNotes(
		ctx,
		i18nConfig.DefaultLocale,
		notes.ListFilter{Page: 1},
		notes.ListOptions{},
	)
	if err != nil {
		return nil, err
	}

	ids := []frameworkdiscovery.SitemapID{
		{
			ID:       "root",
			Path:     sitemapPath,
			Location: joinRootAndPath(rootURL, sitemapPath),
		},
	}
	for i := 0; i < max(baseResult.TotalPages, 0); i++ {
		appendSitemapID(&ids, rootURL, "note", fmt.Sprintf("%s%d%s", noteSitemapPrefix, i, xmlExtension), i)
	}
	for i := 0; i < pageCount(len(baseResult.Authors), authorsPageSize); i++ {
		appendSitemapID(&ids, rootURL, "author", fmt.Sprintf("%s%d%s", authorSitemapPrefix, i, xmlExtension), i)
	}
	for i := 0; i < pageCount(len(baseResult.Tags), tagsPageSize); i++ {
		appendSitemapID(&ids, rootURL, "tag", fmt.Sprintf("%s%d%s", tagSitemapPrefix, i, xmlExtension), i)
	}

	return ids, nil
}

func BuildSitemapEntriesByID(
	ctx context.Context,
	rootURL string,
	i18nConfig frameworki18n.Config,
	service notesLister,
	id string,
	authorsPageSize int,
	tagsPageSize int,
) ([]frameworkdiscovery.SitemapEntry, error) {
	kind, chunkID, ok := parseGeneratedSitemapID(id)
	if !ok {
		return nil, fmt.Errorf("invalid sitemap id %q", id)
	}

	switch kind {
	case "root":
		return BuildRootSitemapEntries(rootURL, i18nConfig)
	case "note":
		return buildNoteSitemapEntries(ctx, rootURL, i18nConfig, service, chunkID)
	case "author":
		return buildAuthorSitemapEntries(ctx, rootURL, i18nConfig, service, chunkID, authorsPageSize)
	case "tag":
		return buildTagSitemapEntries(ctx, rootURL, i18nConfig, service, chunkID, tagsPageSize)
	default:
		return nil, fmt.Errorf("unsupported sitemap id %q", id)
	}
}

func buildNoteSitemapEntries(
	ctx context.Context,
	rootURL string,
	i18nConfig frameworki18n.Config,
	service notesLister,
	chunkID int,
) ([]frameworkdiscovery.SitemapEntry, error) {
	if service == nil {
		return nil, nil
	}

	listResult, err := service.ListNotes(
		ctx,
		i18nConfig.DefaultLocale,
		notes.ListFilter{Page: chunkID + 1},
		notes.ListOptions{},
	)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	entries := make([]frameworkdiscovery.SitemapEntry, 0, len(listResult.Notes))
	for _, item := range listResult.Notes {
		noteSlug := strings.TrimSpace(item.Slug)
		if noteSlug == "" {
			continue
		}

		pathValue := routePathNote + url.PathEscape(noteSlug)
		entry, buildErr := sitemapEntryForPath(rootURL, i18nConfig, pathValue)
		if buildErr != nil {
			continue
		}
		out := toDiscoverySitemapEntry(entry)
		out.ChangeFrequency = "weekly"
		out.LastModified = parseRFC3339Pointer(firstNonEmpty(strings.TrimSpace(item.PublishedAtISO), now))
		out.Images = make([]frameworkdiscovery.SitemapImage, 0, 2)
		for _, image := range noteSitemapImages(rootURL, item.MetaImage, item.Attachment) {
			out.Images = append(out.Images, frameworkdiscovery.SitemapImage{URL: image})
		}
		entries = append(entries, out)
	}

	return entries, nil
}

func buildAuthorSitemapEntries(
	ctx context.Context,
	rootURL string,
	i18nConfig frameworki18n.Config,
	service notesLister,
	chunkID int,
	pageSize int,
) ([]frameworkdiscovery.SitemapEntry, error) {
	if service == nil {
		return nil, nil
	}
	if pageSize < 1 {
		pageSize = defaultSitemapAuthorsPageSize
	}

	baseResult, err := service.ListNotes(
		ctx,
		i18nConfig.DefaultLocale,
		notes.ListFilter{Page: 1},
		notes.ListOptions{},
	)
	if err != nil {
		return nil, err
	}

	authors := sliceAuthorsPage(baseResult.Authors, chunkID, pageSize)
	now := time.Now().UTC()
	priority := 1.0
	entries := make([]frameworkdiscovery.SitemapEntry, 0, len(authors))
	for _, author := range authors {
		authorSlug := strings.TrimSpace(author.Slug)
		if authorSlug == "" {
			continue
		}

		pathValue := routePathAuthor + url.PathEscape(authorSlug)
		entry, buildErr := sitemapEntryForPath(rootURL, i18nConfig, pathValue)
		if buildErr != nil {
			continue
		}
		out := toDiscoverySitemapEntry(entry)
		out.ChangeFrequency = "weekly"
		out.Priority = &priority
		out.LastModified = &now
		entries = append(entries, out)
	}

	return entries, nil
}

func buildTagSitemapEntries(
	ctx context.Context,
	rootURL string,
	i18nConfig frameworki18n.Config,
	service notesLister,
	chunkID int,
	pageSize int,
) ([]frameworkdiscovery.SitemapEntry, error) {
	if service == nil {
		return nil, nil
	}
	if pageSize < 1 {
		pageSize = defaultSitemapTagsPageSize
	}

	baseResult, err := service.ListNotes(
		ctx,
		i18nConfig.DefaultLocale,
		notes.ListFilter{Page: 1},
		notes.ListOptions{},
	)
	if err != nil {
		return nil, err
	}

	tags := sliceTagsPage(baseResult.Tags, chunkID, pageSize)
	now := time.Now().UTC()
	entries := make([]frameworkdiscovery.SitemapEntry, 0, len(tags))
	for _, tag := range tags {
		tagName := strings.TrimSpace(tag.Name)
		if tagName == "" {
			continue
		}

		pathValue := routePathTag + url.PathEscape(tagName)
		entry, buildErr := sitemapEntryForPath(rootURL, i18nConfig, pathValue)
		if buildErr != nil {
			continue
		}
		out := toDiscoverySitemapEntry(entry)
		out.ChangeFrequency = "weekly"
		out.LastModified = &now
		entries = append(entries, out)
	}

	return entries, nil
}

func appendSitemapID(ids *[]frameworkdiscovery.SitemapID, rootURL string, kind string, pathValue string, chunkID int) {
	*ids = append(*ids, frameworkdiscovery.SitemapID{
		ID:       fmt.Sprintf("%s:%d", kind, chunkID),
		Path:     pathValue,
		Location: joinRootAndPath(rootURL, pathValue),
	})
}

func parseGeneratedSitemapID(raw string) (string, int, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "root" {
		return "root", 0, true
	}

	kind, chunk, ok := strings.Cut(trimmed, ":")
	if !ok {
		return "", 0, false
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(chunk))
	if err != nil || parsed < 0 {
		return "", 0, false
	}
	return strings.TrimSpace(kind), parsed, true
}

func toDiscoverySitemapEntry(entry sitemapURLEntry) frameworkdiscovery.SitemapEntry {
	out := frameworkdiscovery.SitemapEntry{
		URL:             strings.TrimSpace(entry.Loc),
		Alternates:      entry.Alternates,
		ChangeFrequency: strings.TrimSpace(entry.ChangeFreq),
	}
	out.LastModified = parseRFC3339Pointer(entry.LastMod)
	if priority := strings.TrimSpace(entry.Priority); priority != "" {
		if parsed, err := strconv.ParseFloat(priority, 64); err == nil {
			out.Priority = &parsed
		}
	}
	for _, image := range entry.Images {
		location := strings.TrimSpace(image)
		if location == "" {
			continue
		}
		out.Images = append(out.Images, frameworkdiscovery.SitemapImage{URL: location})
	}
	return out
}

func parseRFC3339Pointer(raw string) *time.Time {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}

	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339Nano, trimmed)
		if err != nil {
			return nil
		}
	}
	return &parsed
}
