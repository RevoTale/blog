package seo

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"
	texttemplate "text/template"
	"time"

	"blog/internal/notes"
	"blog/internal/req"

	frameworki18n "github.com/RevoTale/no-js/framework/i18n"
	"github.com/RevoTale/no-js/framework/metagen"
)

const rssEndpointPath = "/feed.xml"
const sitemapPath = "/sitemap.xml"
const sitemapIndexPath = "/sitemap-index"
const sitemapIndexXMLPath = "/sitemap-index.xml"

const routePathRoot = "/"
const routePathChannels = "/channels"
const routePathTales = "/tales"
const routePathMicroTales = "/micro-tales"

const noteSitemapPrefix = "/note/sitemap/"
const authorSitemapPrefix = "/author/sitemap/"
const tagSitemapPrefix = "/notes/sitemap/"

const routePathNote = "/note/"
const routePathAuthor = "/author/"
const routePathTag = "/tag/"

const defaultRSSCachePolicy = "public, max-age=3600, s-maxage=3600"
const defaultSitemapCachePolicy = "public, max-age=3600, s-maxage=3600"
const defaultSitemapIndexCachePolicy = "public, max-age=3600, s-maxage=3600, " +
	"stale-while-revalidate=9000, stale-if-error=86400"

const defaultSitemapAuthorsPageSize = 1000
const defaultSitemapTagsPageSize = 50

const queryParamLocale = "locale"
const queryParamPage = "page"
const queryParamAuthor = "author"
const queryParamTag = "tag"
const queryParamType = "type"
const queryParamSearch = "q"
const xmlExtension = ".xml"

const contentTypeRSSXML = "application/rss+xml; charset=utf-8"
const contentTypeApplicationXML = "application/xml; charset=utf-8"

type notesLister interface {
	ListNotes(
		ctx context.Context,
		locale string,
		filter notes.ListFilter,
		options notes.ListOptions,
	) (notes.NotesListResult, error)
}

type FeedAndSitemapConfig struct {
	RootURL string

	I18nConfig frameworki18n.Config
	Notes      notesLister

	RSSCachePolicy      string
	SitemapCachePolicy  string
	SitemapIndexPolicy  string
	AuthorsSitemapLimit int
	TagsSitemapLimit    int
}

func WithFeedAndSitemapEndpoints(next http.Handler, cfg FeedAndSitemapConfig) http.Handler {
	normalizedI18n, err := frameworki18n.NormalizeConfig(cfg.I18nConfig)
	if err != nil {
		normalizedI18n = frameworki18n.Config{
			Locales:       []string{"en"},
			DefaultLocale: "en",
			PrefixMode:    frameworki18n.PrefixAsNeeded,
		}
	}

	authorsPageSize := cfg.AuthorsSitemapLimit
	if authorsPageSize < 1 {
		authorsPageSize = defaultSitemapAuthorsPageSize
	}

	tagsPageSize := cfg.TagsSitemapLimit
	if tagsPageSize < 1 {
		tagsPageSize = defaultSitemapTagsPageSize
	}

	rssCachePolicy := strings.TrimSpace(cfg.RSSCachePolicy)
	if rssCachePolicy == "" {
		rssCachePolicy = defaultRSSCachePolicy
	}

	sitemapCachePolicy := strings.TrimSpace(cfg.SitemapCachePolicy)
	if sitemapCachePolicy == "" {
		sitemapCachePolicy = defaultSitemapCachePolicy
	}

	sitemapIndexPolicy := strings.TrimSpace(cfg.SitemapIndexPolicy)
	if sitemapIndexPolicy == "" {
		sitemapIndexPolicy = defaultSitemapIndexCachePolicy
	}

	rootURL := strings.TrimSpace(cfg.RootURL)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if next == nil {
			return
		}
		if r == nil || r.URL == nil {
			next.ServeHTTP(w, r)
			return
		}

		path := strings.TrimSpace(r.URL.Path)
		switch path {
		case rssEndpointPath:
			serveRSSFeedEndpoint(w, r, feedEndpointConfig{
				RootURL:     rootURL,
				I18nConfig:  normalizedI18n,
				Notes:       cfg.Notes,
				CachePolicy: rssCachePolicy,
			})
			return
		case sitemapPath:
			serveRootSitemapEndpoint(w, r, sitemapEndpointConfig{
				RootURL:     rootURL,
				I18nConfig:  normalizedI18n,
				CachePolicy: sitemapCachePolicy,
			})
			return
		case sitemapIndexPath, sitemapIndexXMLPath:
			serveSitemapIndexEndpoint(w, r, sitemapIndexEndpointConfig{
				RootURL:         rootURL,
				I18nConfig:      normalizedI18n,
				Notes:           cfg.Notes,
				CachePolicy:     sitemapIndexPolicy,
				AuthorsPageSize: authorsPageSize,
				TagsPageSize:    tagsPageSize,
			})
			return
		}

		if chunkID, ok := parseSitemapChunkID(path, noteSitemapPrefix); ok {
			serveNoteSitemapEndpoint(w, r, chunkedSitemapEndpointConfig{
				RootURL:     rootURL,
				I18nConfig:  normalizedI18n,
				Notes:       cfg.Notes,
				ChunkID:     chunkID,
				CachePolicy: sitemapCachePolicy,
			})
			return
		}

		if chunkID, ok := parseSitemapChunkID(path, authorSitemapPrefix); ok {
			serveAuthorSitemapEndpoint(w, r, chunkedSitemapEndpointConfig{
				RootURL:         rootURL,
				I18nConfig:      normalizedI18n,
				Notes:           cfg.Notes,
				ChunkID:         chunkID,
				CachePolicy:     sitemapCachePolicy,
				AuthorsPageSize: authorsPageSize,
			})
			return
		}

		if chunkID, ok := parseSitemapChunkID(path, tagSitemapPrefix); ok {
			serveTagSitemapEndpoint(w, r, chunkedSitemapEndpointConfig{
				RootURL:      rootURL,
				I18nConfig:   normalizedI18n,
				Notes:        cfg.Notes,
				ChunkID:      chunkID,
				CachePolicy:  sitemapCachePolicy,
				TagsPageSize: tagsPageSize,
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func Mount(mux *http.ServeMux, cfg FeedAndSitemapConfig) error {
	if mux == nil {
		return fmt.Errorf("mux is required")
	}

	normalizedI18n, err := frameworki18n.NormalizeConfig(cfg.I18nConfig)
	if err != nil {
		normalizedI18n = frameworki18n.Config{
			Locales:       []string{"en"},
			DefaultLocale: "en",
			PrefixMode:    frameworki18n.PrefixAsNeeded,
		}
	}

	authorsPageSize := cfg.AuthorsSitemapLimit
	if authorsPageSize < 1 {
		authorsPageSize = defaultSitemapAuthorsPageSize
	}

	tagsPageSize := cfg.TagsSitemapLimit
	if tagsPageSize < 1 {
		tagsPageSize = defaultSitemapTagsPageSize
	}

	rssCachePolicy := strings.TrimSpace(cfg.RSSCachePolicy)
	if rssCachePolicy == "" {
		rssCachePolicy = defaultRSSCachePolicy
	}

	sitemapCachePolicy := strings.TrimSpace(cfg.SitemapCachePolicy)
	if sitemapCachePolicy == "" {
		sitemapCachePolicy = defaultSitemapCachePolicy
	}

	sitemapIndexPolicy := strings.TrimSpace(cfg.SitemapIndexPolicy)
	if sitemapIndexPolicy == "" {
		sitemapIndexPolicy = defaultSitemapIndexCachePolicy
	}

	rootURL := strings.TrimSpace(cfg.RootURL)

	mux.HandleFunc(rssEndpointPath, func(w http.ResponseWriter, r *http.Request) {
		serveRSSFeedEndpoint(w, r, feedEndpointConfig{
			RootURL:     rootURL,
			I18nConfig:  normalizedI18n,
			Notes:       cfg.Notes,
			CachePolicy: rssCachePolicy,
		})
	})
	mux.HandleFunc(sitemapPath, func(w http.ResponseWriter, r *http.Request) {
		serveRootSitemapEndpoint(w, r, sitemapEndpointConfig{
			RootURL:     rootURL,
			I18nConfig:  normalizedI18n,
			CachePolicy: sitemapCachePolicy,
		})
	})
	indexHandler := func(w http.ResponseWriter, r *http.Request) {
		serveSitemapIndexEndpoint(w, r, sitemapIndexEndpointConfig{
			RootURL:         rootURL,
			I18nConfig:      normalizedI18n,
			Notes:           cfg.Notes,
			CachePolicy:     sitemapIndexPolicy,
			AuthorsPageSize: authorsPageSize,
			TagsPageSize:    tagsPageSize,
		})
	}
	mux.HandleFunc(sitemapIndexPath, indexHandler)
	mux.HandleFunc(sitemapIndexXMLPath, indexHandler)
	mux.HandleFunc(noteSitemapPrefix, func(w http.ResponseWriter, r *http.Request) {
		chunkID, ok := parseSitemapChunkID(r.URL.Path, noteSitemapPrefix)
		if !ok {
			http.NotFound(w, r)
			return
		}
		serveNoteSitemapEndpoint(w, r, chunkedSitemapEndpointConfig{
			RootURL:     rootURL,
			I18nConfig:  normalizedI18n,
			Notes:       cfg.Notes,
			ChunkID:     chunkID,
			CachePolicy: sitemapCachePolicy,
		})
	})
	mux.HandleFunc(authorSitemapPrefix, func(w http.ResponseWriter, r *http.Request) {
		chunkID, ok := parseSitemapChunkID(r.URL.Path, authorSitemapPrefix)
		if !ok {
			http.NotFound(w, r)
			return
		}
		serveAuthorSitemapEndpoint(w, r, chunkedSitemapEndpointConfig{
			RootURL:         rootURL,
			I18nConfig:      normalizedI18n,
			Notes:           cfg.Notes,
			ChunkID:         chunkID,
			CachePolicy:     sitemapCachePolicy,
			AuthorsPageSize: authorsPageSize,
		})
	})
	mux.HandleFunc(tagSitemapPrefix, func(w http.ResponseWriter, r *http.Request) {
		chunkID, ok := parseSitemapChunkID(r.URL.Path, tagSitemapPrefix)
		if !ok {
			http.NotFound(w, r)
			return
		}
		serveTagSitemapEndpoint(w, r, chunkedSitemapEndpointConfig{
			RootURL:      rootURL,
			I18nConfig:   normalizedI18n,
			Notes:        cfg.Notes,
			ChunkID:      chunkID,
			CachePolicy:  sitemapCachePolicy,
			TagsPageSize: tagsPageSize,
		})
	})

	return nil
}

type feedEndpointConfig struct {
	RootURL     string
	I18nConfig  frameworki18n.Config
	Notes       notesLister
	CachePolicy string
}

func serveRSSFeedEndpoint(w http.ResponseWriter, r *http.Request, cfg feedEndpointConfig) {
	if !ensureReadMethod(w, r.Method) {
		return
	}
	if !ensureNotesLister(w, cfg.Notes) {
		return
	}

	locale := resolveLocale(r.URL.Query().Get(queryParamLocale), cfg.I18nConfig)
	filter := rssListFilterFromQuery(r.URL.Query())
	listResult, err := cfg.Notes.ListNotes(
		r.Context(),
		locale,
		filter,
		notes.ListOptions{},
	)
	if err != nil {
		writeEndpointInternalServerError(w)
		return
	}

	payload, err := buildRSSFeed(cfg.RootURL, cfg.I18nConfig, locale, listResult.Notes)
	if err != nil {
		writeEndpointInternalServerError(w)
		return
	}

	writeXMLResponse(w, payload, cfg.CachePolicy, contentTypeRSSXML)
}

type sitemapEndpointConfig struct {
	RootURL     string
	I18nConfig  frameworki18n.Config
	CachePolicy string
}

func serveRootSitemapEndpoint(w http.ResponseWriter, r *http.Request, cfg sitemapEndpointConfig) {
	if !ensureReadMethod(w, r.Method) {
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	paths := []string{routePathRoot, routePathChannels, routePathTales, routePathMicroTales}
	entries := make([]sitemapURLEntry, 0, len(paths))
	for _, pathValue := range paths {
		entry, err := sitemapEntryForPath(cfg.RootURL, cfg.I18nConfig, pathValue)
		if err != nil {
			continue
		}
		entry.LastMod = now
		entry.ChangeFreq = "weekly"
		entries = append(entries, entry)
	}

	payload, err := renderSitemapXML(entries)
	if err != nil {
		writeEndpointInternalServerError(w)
		return
	}

	writeXMLResponse(w, payload, cfg.CachePolicy, contentTypeApplicationXML)
}

type sitemapIndexEndpointConfig struct {
	RootURL         string
	I18nConfig      frameworki18n.Config
	Notes           notesLister
	CachePolicy     string
	AuthorsPageSize int
	TagsPageSize    int
}

func serveSitemapIndexEndpoint(w http.ResponseWriter, r *http.Request, cfg sitemapIndexEndpointConfig) {
	if !ensureReadMethod(w, r.Method) {
		return
	}
	if !ensureNotesLister(w, cfg.Notes) {
		return
	}

	baseResult, err := cfg.Notes.ListNotes(
		r.Context(),
		cfg.I18nConfig.DefaultLocale,
		notes.ListFilter{Page: 1},
		notes.ListOptions{},
	)
	if err != nil {
		writeEndpointInternalServerError(w)
		return
	}

	locations := make([]string, 0, 1+baseResult.TotalPages)
	locations = append(locations, joinRootAndPath(cfg.RootURL, sitemapPath))

	for i := 0; i < max(baseResult.TotalPages, 0); i++ {
		locations = append(
			locations,
			joinRootAndPath(cfg.RootURL, fmt.Sprintf("%s%d%s", noteSitemapPrefix, i, xmlExtension)),
		)
	}

	for i := 0; i < pageCount(len(baseResult.Authors), cfg.AuthorsPageSize); i++ {
		locations = append(
			locations,
			joinRootAndPath(cfg.RootURL, fmt.Sprintf("%s%d%s", authorSitemapPrefix, i, xmlExtension)),
		)
	}

	for i := 0; i < pageCount(len(baseResult.Tags), cfg.TagsPageSize); i++ {
		locations = append(
			locations,
			joinRootAndPath(cfg.RootURL, fmt.Sprintf("%s%d%s", tagSitemapPrefix, i, xmlExtension)),
		)
	}

	payload, renderErr := renderSitemapIndexXML(locations)
	if renderErr != nil {
		writeEndpointInternalServerError(w)
		return
	}

	writeXMLResponse(w, payload, cfg.CachePolicy, contentTypeApplicationXML)
}

type chunkedSitemapEndpointConfig struct {
	RootURL string

	I18nConfig frameworki18n.Config
	Notes      notesLister

	ChunkID         int
	CachePolicy     string
	AuthorsPageSize int
	TagsPageSize    int
}

func serveNoteSitemapEndpoint(w http.ResponseWriter, r *http.Request, cfg chunkedSitemapEndpointConfig) {
	if !ensureReadMethod(w, r.Method) {
		return
	}
	if !ensureNotesLister(w, cfg.Notes) {
		return
	}

	listResult, err := cfg.Notes.ListNotes(
		r.Context(),
		cfg.I18nConfig.DefaultLocale,
		notes.ListFilter{Page: cfg.ChunkID + 1},
		notes.ListOptions{},
	)
	if err != nil {
		writeEndpointInternalServerError(w)
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	entries := make([]sitemapURLEntry, 0, len(listResult.Notes))
	for _, item := range listResult.Notes {
		noteSlug := strings.TrimSpace(item.Slug)
		if noteSlug == "" {
			continue
		}

		pathValue := routePathNote + url.PathEscape(noteSlug)
		entry, buildErr := sitemapEntryForPath(cfg.RootURL, cfg.I18nConfig, pathValue)
		if buildErr != nil {
			continue
		}
		entry.ChangeFreq = "weekly"
		entry.LastMod = firstNonEmpty(strings.TrimSpace(item.PublishedAtISO), now)
		entry.Images = noteSitemapImages(cfg.RootURL, item.MetaImage, item.Attachment)
		entries = append(entries, entry)
	}

	payload, renderErr := renderSitemapXML(entries)
	if renderErr != nil {
		writeEndpointInternalServerError(w)
		return
	}

	writeXMLResponse(w, payload, cfg.CachePolicy, contentTypeApplicationXML)
}

func serveAuthorSitemapEndpoint(w http.ResponseWriter, r *http.Request, cfg chunkedSitemapEndpointConfig) {
	if !ensureReadMethod(w, r.Method) {
		return
	}
	if !ensureNotesLister(w, cfg.Notes) {
		return
	}

	pageSize := cfg.AuthorsPageSize
	if pageSize < 1 {
		pageSize = defaultSitemapAuthorsPageSize
	}

	baseResult, err := cfg.Notes.ListNotes(
		r.Context(),
		cfg.I18nConfig.DefaultLocale,
		notes.ListFilter{Page: 1},
		notes.ListOptions{},
	)
	if err != nil {
		writeEndpointInternalServerError(w)
		return
	}

	authors := sliceAuthorsPage(baseResult.Authors, cfg.ChunkID, pageSize)
	now := time.Now().UTC().Format(time.RFC3339)
	entries := make([]sitemapURLEntry, 0, len(authors))
	for _, author := range authors {
		authorSlug := strings.TrimSpace(author.Slug)
		if authorSlug == "" {
			continue
		}

		pathValue := routePathAuthor + url.PathEscape(authorSlug)
		entry, buildErr := sitemapEntryForPath(cfg.RootURL, cfg.I18nConfig, pathValue)
		if buildErr != nil {
			continue
		}
		entry.ChangeFreq = "weekly"
		entry.Priority = "1.0"
		entry.LastMod = now
		entries = append(entries, entry)
	}

	payload, renderErr := renderSitemapXML(entries)
	if renderErr != nil {
		writeEndpointInternalServerError(w)
		return
	}

	writeXMLResponse(w, payload, cfg.CachePolicy, contentTypeApplicationXML)
}

func serveTagSitemapEndpoint(w http.ResponseWriter, r *http.Request, cfg chunkedSitemapEndpointConfig) {
	if !ensureReadMethod(w, r.Method) {
		return
	}
	if !ensureNotesLister(w, cfg.Notes) {
		return
	}

	pageSize := cfg.TagsPageSize
	if pageSize < 1 {
		pageSize = defaultSitemapTagsPageSize
	}

	baseResult, err := cfg.Notes.ListNotes(
		r.Context(),
		cfg.I18nConfig.DefaultLocale,
		notes.ListFilter{Page: 1},
		notes.ListOptions{},
	)
	if err != nil {
		writeEndpointInternalServerError(w)
		return
	}

	tags := sliceTagsPage(baseResult.Tags, cfg.ChunkID, pageSize)
	now := time.Now().UTC().Format(time.RFC3339)
	entries := make([]sitemapURLEntry, 0, len(tags))
	for _, tag := range tags {
		tagName := strings.TrimSpace(tag.Name)
		if tagName == "" {
			continue
		}

		pathValue := routePathTag + url.PathEscape(tagName)
		entry, buildErr := sitemapEntryForPath(cfg.RootURL, cfg.I18nConfig, pathValue)
		if buildErr != nil {
			continue
		}
		entry.ChangeFreq = "weekly"
		entry.LastMod = now
		entries = append(entries, entry)
	}

	payload, renderErr := renderSitemapXML(entries)
	if renderErr != nil {
		writeEndpointInternalServerError(w)
		return
	}

	writeXMLResponse(w, payload, cfg.CachePolicy, contentTypeApplicationXML)
}

func resolveLocale(raw string, cfg frameworki18n.Config) string {
	locale := strings.ToLower(strings.TrimSpace(raw))
	if locale == "" {
		return cfg.DefaultLocale
	}
	for _, supported := range cfg.Locales {
		if locale == supported {
			return locale
		}
	}
	return cfg.DefaultLocale
}

func parseSitemapChunkID(pathValue string, prefix string) (int, bool) {
	if !strings.HasPrefix(pathValue, prefix) {
		return 0, false
	}

	chunk := strings.TrimPrefix(pathValue, prefix)
	if !strings.HasSuffix(chunk, xmlExtension) {
		return 0, false
	}
	chunk = strings.TrimSuffix(chunk, xmlExtension)
	if chunk == "" || strings.Contains(chunk, "/") {
		return 0, false
	}

	parsed, err := strconv.Atoi(chunk)
	if err != nil || parsed < 0 {
		return 0, false
	}
	return parsed, true
}

func rssListFilterFromQuery(query url.Values) notes.ListFilter {
	return notes.ListFilter{
		Page:       parsePositiveInt(query.Get(queryParamPage), 1),
		AuthorSlug: strings.TrimSpace(query.Get(queryParamAuthor)),
		TagName:    strings.TrimSpace(query.Get(queryParamTag)),
		Type:       notes.ParseNoteType(query.Get(queryParamType)),
		Query:      strings.TrimSpace(query.Get(queryParamSearch)),
	}
}

func parsePositiveInt(raw string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || parsed < 1 {
		return fallback
	}
	return parsed
}

func sitemapEntryForPath(
	rootURL string,
	i18nConfig frameworki18n.Config,
	strippedPath string,
) (sitemapURLEntry, error) {
	alternates, err := metagen.BuildAlternates(rootURL, i18nConfig, i18nConfig.DefaultLocale, strippedPath, nil)
	if err != nil {
		return sitemapURLEntry{}, err
	}

	return sitemapURLEntry{
		Loc:        strings.TrimSpace(alternates.Canonical),
		Alternates: alternates.Languages,
	}, nil
}

func noteSitemapImages(rootURL string, metaImage *notes.Attachment, attachment *notes.Attachment) []string {
	unique := map[string]struct{}{}
	for _, candidate := range []string{
		absoluteMediaURL(rootURL, attachmentURL(metaImage)),
		absoluteMediaURL(rootURL, attachmentURL(attachment)),
	} {
		if strings.TrimSpace(candidate) == "" {
			continue
		}
		unique[candidate] = struct{}{}
	}

	out := make([]string, 0, len(unique))
	for item := range unique {
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

func attachmentURL(value *notes.Attachment) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(value.URL)
}

func absoluteMediaURL(rootURL string, rawURL string) string {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return ""
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return ""
	}
	if parsed.IsAbs() && strings.TrimSpace(parsed.Host) != "" {
		return parsed.String()
	}
	return joinRootAndPath(rootURL, parsed.Path)
}

func pageCount(total int, pageSize int) int {
	if total < 1 || pageSize < 1 {
		return 0
	}
	count := total / pageSize
	if total%pageSize != 0 {
		count++
	}
	return count
}

func sliceAuthorsPage(authors []notes.Author, pageID int, pageSize int) []notes.Author {
	start := pageID * pageSize
	if start < 0 || start >= len(authors) {
		return []notes.Author{}
	}
	end := min(start+pageSize, len(authors))
	return authors[start:end]
}

func sliceTagsPage(tags []notes.Tag, pageID int, pageSize int) []notes.Tag {
	start := pageID * pageSize
	if start < 0 || start >= len(tags) {
		return []notes.Tag{}
	}
	end := min(start+pageSize, len(tags))
	return tags[start:end]
}

func firstNonEmpty(candidates ...string) string {
	for _, candidate := range candidates {
		trimmed := strings.TrimSpace(candidate)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

type rssItem struct {
	Title       string
	Link        string
	GUID        string
	Description string
	Author      string
	PubDate     string
	Categories  []string
}

func buildRSSFeed(
	rootURL string,
	i18nConfig frameworki18n.Config,
	locale string,
	noteItems []notes.NoteSummary,
) (string, error) {
	homeURL := joinRootAndPath(rootURL, frameworki18n.LocalizePath(i18nConfig, locale, routePathRoot))
	feedURL := joinRootAndPath(rootURL, rssEndpointPath) + "?" + queryParamLocale + "=" + url.QueryEscape(locale)

	items := make([]rssItem, 0, len(noteItems))
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

		items = append(
			items,
			rssItem{
				Title:       title,
				Link:        link,
				GUID:        link,
				Description: description,
				Author:      author,
				PubDate:     toRFC1123Z(note.PublishedAtISO),
				Categories:  categories,
			},
		)
	}

	lastBuildDate := time.Now().UTC().Format(time.RFC1123Z)
	if len(noteItems) > 0 {
		if parsed := toRFC1123Z(noteItems[0].PublishedAtISO); parsed != "" {
			lastBuildDate = parsed
		}
	}

	return executeXMLTemplate(
		rssXMLTemplate,
		rssXMLTemplateData{
			XMLHeader:     xml.Header,
			Title:         "RevoTale Notes",
			Link:          homeURL,
			Description:   "Latest notes and micro posts from RevoTale",
			Language:      locale,
			LastBuildDate: lastBuildDate,
			Generator:     "RevoTale RSS Generator",
			Copyright:     fmt.Sprintf("© %d RevoTale", time.Now().UTC().Year()),
			FeedURL:       feedURL,
			Items:         items,
		},
	)
}

func toRFC1123Z(isoValue string) string {
	trimmed := strings.TrimSpace(isoValue)
	if trimmed == "" {
		return ""
	}

	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339Nano, trimmed)
		if err != nil {
			return ""
		}
	}
	return parsed.UTC().Format(time.RFC1123Z)
}

type sitemapURLEntry struct {
	Loc        string
	Alternates map[string]string
	Images     []string
	LastMod    string
	ChangeFreq string
	Priority   string
}

func renderSitemapXML(entries []sitemapURLEntry) (string, error) {
	return executeXMLTemplate(sitemapXMLTemplate, newSitemapXMLTemplateData(entries))
}

func renderSitemapIndexXML(locations []string) (string, error) {
	return executeXMLTemplate(
		sitemapIndexXMLTemplate,
		sitemapIndexXMLTemplateData{
			XMLHeader: xml.Header,
			Locations: compactNonEmptyStrings(locations),
		},
	)
}

type rssXMLTemplateData struct {
	XMLHeader     string
	Title         string
	Link          string
	Description   string
	Language      string
	LastBuildDate string
	Generator     string
	Copyright     string
	FeedURL       string
	Items         []rssItem
}

type sitemapAlternate struct {
	Locale string
	Href   string
}

type sitemapXMLEntry struct {
	Loc        string
	Alternates []sitemapAlternate
	Images     []string
	LastMod    string
	ChangeFreq string
	Priority   string
}

type sitemapXMLTemplateData struct {
	XMLHeader      string
	WithAlternates bool
	WithImages     bool
	Entries        []sitemapXMLEntry
}

type sitemapIndexXMLTemplateData struct {
	XMLHeader string
	Locations []string
}

func newSitemapXMLTemplateData(entries []sitemapURLEntry) sitemapXMLTemplateData {
	view := sitemapXMLTemplateData{
		XMLHeader: xml.Header,
		Entries:   make([]sitemapXMLEntry, 0, len(entries)),
	}

	for _, entry := range entries {
		loc := strings.TrimSpace(entry.Loc)
		if loc == "" {
			continue
		}

		outEntry := sitemapXMLEntry{
			Loc:        loc,
			Alternates: make([]sitemapAlternate, 0, len(entry.Alternates)),
			Images:     compactNonEmptyStrings(entry.Images),
			LastMod:    strings.TrimSpace(entry.LastMod),
			ChangeFreq: strings.TrimSpace(entry.ChangeFreq),
			Priority:   strings.TrimSpace(entry.Priority),
		}

		keys := make([]string, 0, len(entry.Alternates))
		for key := range entry.Alternates {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			href := strings.TrimSpace(entry.Alternates[key])
			if href == "" {
				continue
			}
			outEntry.Alternates = append(
				outEntry.Alternates,
				sitemapAlternate{
					Locale: strings.TrimSpace(key),
					Href:   href,
				},
			)
		}

		if len(outEntry.Alternates) > 0 {
			view.WithAlternates = true
		}
		if len(outEntry.Images) > 0 {
			view.WithImages = true
		}

		view.Entries = append(view.Entries, outEntry)
	}

	return view
}

func compactNonEmptyStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

func ensureReadMethod(w http.ResponseWriter, method string) bool {
	if req.IsReadMethod(method) {
		return true
	}
	w.WriteHeader(http.StatusMethodNotAllowed)
	return false
}

func ensureNotesLister(w http.ResponseWriter, service notesLister) bool {
	if service != nil {
		return true
	}
	writeEndpointInternalServerError(w)
	return false
}

func writeEndpointInternalServerError(w http.ResponseWriter) {
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func writeXMLResponse(
	w http.ResponseWriter,
	payload string,
	cachePolicy string,
	contentType string,
) {
	req.SetCacheControl(w, cachePolicy)
	w.Header().Set("Content-Type", strings.TrimSpace(contentType))
	_, _ = io.WriteString(w, payload)
}

func executeXMLTemplate(template *texttemplate.Template, data any) (string, error) {
	var buffer bytes.Buffer
	if err := template.Execute(&buffer, data); err != nil {
		return "", err
	}
	return buffer.String(), nil
}

var xmlTemplateFunctions = texttemplate.FuncMap{
	"xml": xmlEscape,
}

var rssXMLTemplate = texttemplate.Must(
	texttemplate.New("rss.xml").Funcs(xmlTemplateFunctions).Parse(
		`{{.XMLHeader}}<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">
  <channel>
    <title>{{xml .Title}}</title>
    <link>{{xml .Link}}</link>
    <description>{{xml .Description}}</description>
    <language>{{xml .Language}}</language>
    <lastBuildDate>{{xml .LastBuildDate}}</lastBuildDate>
    <generator>{{xml .Generator}}</generator>
    <copyright>{{xml .Copyright}}</copyright>
    <atom:link href="{{xml .FeedURL}}" rel="self" type="application/rss+xml"/>
{{- range .Items}}
    <item>
      <title>{{xml .Title}}</title>
      <link>{{xml .Link}}</link>
      <guid>{{xml .GUID}}</guid>
{{- if .Description}}
      <description>{{xml .Description}}</description>
{{- end}}
{{- if .Author}}
      <author>{{xml .Author}}</author>
{{- end}}
{{- if .PubDate}}
      <pubDate>{{xml .PubDate}}</pubDate>
{{- end}}
{{- range .Categories}}
      <category>{{xml .}}</category>
{{- end}}
    </item>
{{- end}}
  </channel>
</rss>
`,
	),
)

var sitemapXMLTemplate = texttemplate.Must(
	texttemplate.New("sitemap.xml").Funcs(xmlTemplateFunctions).Parse(
		`{{.XMLHeader}}<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"{{if .WithAlternates}}
 xmlns:xhtml="http://www.w3.org/1999/xhtml"{{end}}{{if .WithImages}}
 xmlns:image="http://www.google.com/schemas/sitemap-image/1.1"{{end}}>
{{- range .Entries}}
  <url>
    <loc>{{xml .Loc}}</loc>
{{- range .Alternates}}
    <xhtml:link rel="alternate" hreflang="{{xml .Locale}}" href="{{xml .Href}}"/>
{{- end}}
{{- if .LastMod}}
    <lastmod>{{xml .LastMod}}</lastmod>
{{- end}}
{{- if .ChangeFreq}}
    <changefreq>{{xml .ChangeFreq}}</changefreq>
{{- end}}
{{- if .Priority}}
    <priority>{{xml .Priority}}</priority>
{{- end}}
{{- range .Images}}
    <image:image><image:loc>{{xml .}}</image:loc></image:image>
{{- end}}
  </url>
{{- end}}
</urlset>
`,
	),
)

var sitemapIndexXMLTemplate = texttemplate.Must(
	texttemplate.New("sitemap-index.xml").Funcs(xmlTemplateFunctions).Parse(
		`{{.XMLHeader}}<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
{{- range .Locations}}
  <sitemap><loc>{{xml .}}</loc></sitemap>
{{- end}}
</sitemapindex>
`,
	),
)

func xmlEscape(value string) string {
	var builder strings.Builder
	if err := xml.EscapeText(&builder, []byte(value)); err != nil {
		return value
	}
	return builder.String()
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func joinRootAndPath(rootURL string, routePath string) string {
	trimmedPath := strings.TrimSpace(routePath)
	if trimmedPath == "" {
		trimmedPath = "/"
	}
	if !strings.HasPrefix(trimmedPath, "/") {
		trimmedPath = "/" + trimmedPath
	}

	parsedRoot, err := url.Parse(strings.TrimSpace(rootURL))
	if err != nil || !parsedRoot.IsAbs() || strings.TrimSpace(parsedRoot.Host) == "" {
		return trimmedPath
	}

	base := strings.TrimSuffix(strings.TrimSpace(parsedRoot.Path), "/")
	if trimmedPath == "/" {
		if base == "" {
			parsedRoot.Path = "/"
		} else {
			parsedRoot.Path = base
		}
		parsedRoot.RawQuery = ""
		parsedRoot.Fragment = ""
		return parsedRoot.String()
	}

	joined := path.Join(base, strings.TrimPrefix(trimmedPath, "/"))
	if !strings.HasPrefix(joined, "/") {
		joined = "/" + joined
	}
	parsedRoot.Path = joined
	parsedRoot.RawQuery = ""
	parsedRoot.Fragment = ""
	return parsedRoot.String()
}
