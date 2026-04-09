package discovery

import (
	"context"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"

	"blog/internal/notes"
	frameworki18n "github.com/RevoTale/no-js/framework/i18n"
	"github.com/RevoTale/no-js/framework/metagen"
)

const rssEndpointPath = "/feed.xml"
const sitemapIndexPath = "/sitemap-index.xml"

const routePathRoot = "/"
const routePathChannels = "/channels"
const routePathTales = "/tales"
const routePathMicroTales = "/micro-tales"

const routePathNote = "/note/"
const routePathAuthor = "/author/"
const routePathTag = "/tag/"

const defaultSitemapAuthorsPageSize = 1000
const defaultSitemapTagsPageSize = 50

const queryParamLocale = "locale"
const queryParamPage = "page"
const queryParamAuthor = "author"
const queryParamTag = "tag"
const queryParamType = "type"
const queryParamSearch = "q"

type notesLister interface {
	ListNotes(
		ctx context.Context,
		locale string,
		filter notes.ListFilter,
		options notes.ListOptions,
	) (notes.NotesListResult, error)
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

type sitemapURLEntry struct {
	Loc        string
	Alternates map[string]string
	Images     []string
	LastMod    string
	ChangeFreq string
	Priority   string
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
