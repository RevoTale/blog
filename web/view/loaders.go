package runtime

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"blog/internal/notes"
	i18nkeys "blog/web/generated/i18nkeys"
	"github.com/RevoTale/no-js/framework"
	frameworki18n "github.com/RevoTale/no-js/framework/i18n"
	"github.com/RevoTale/no-js/framework/metagen"
)

const liveNavigationQueryKey = "__live"
const liveNavigationQueryValue = "navigation"
const rssEndpointPath = "/feed.xml"

func LoadNotesPage(
	ctx context.Context,
	appCtx *Context,
	r *http.Request,
	_ framework.EmptyParams,
) (NotesPageView, error) {
	locale := localeFromRequest(appCtx, r)
	filter := listFilterFromQuery(r, notes.ListFilter{})
	cacheKey := loaderCacheKey("LoadNotesPage", locale, r)
	return framework.CachedCall(ctx, cacheKey, func(runCtx context.Context) (NotesPageView, error) {
		view, err := loadNotesListPage(
			runCtx,
			appCtx,
			r,
			locale,
			filter,
			notes.ListOptions{},
			sidebarModeForFilter(filter),
		)
		if err != nil {
			return NotesPageView{}, err
		}
		applyStructuredDataContextForNotesView(&view, appCtx, r, locale)
		view.EmptyStateMessage = i18nkeys.TEmptyRoot(view.I18n())
		return view, nil
	})
}

func LoadAuthorPage(
	ctx context.Context,
	appCtx *Context,
	r *http.Request,
	params framework.SlugParams,
) (AuthorPageView, error) {
	locale := localeFromRequest(appCtx, r)
	defaults := notes.ListFilter{AuthorSlug: params.Slug}
	filter := listFilterFromQuery(r, defaults)
	filter.AuthorSlug = strings.TrimSpace(params.Slug)
	cacheKey := loaderCacheKey("LoadAuthorPage", locale, r, filter.AuthorSlug)
	return framework.CachedCall(ctx, cacheKey, func(runCtx context.Context) (AuthorPageView, error) {
		view, err := loadNotesListPage(
			runCtx,
			appCtx,
			r,
			locale,
			filter,
			notes.ListOptions{RequireAuthor: true},
			SidebarModeFiltered,
		)
		if err != nil {
			return AuthorPageView{}, err
		}
		applyStructuredDataContextForNotesView(&view, appCtx, r, locale)
		view.EmptyStateMessage = i18nkeys.TEmptyAuthor(view.I18n())
		return AuthorPageView(view), nil
	})
}

func LoadTagPage(
	ctx context.Context,
	appCtx *Context,
	r *http.Request,
	params framework.SlugParams,
) (NotesPageView, error) {
	locale := localeFromRequest(appCtx, r)
	defaults := notes.ListFilter{TagName: params.Slug}
	filter := listFilterFromQuery(r, defaults)
	filter.TagName = strings.TrimSpace(params.Slug)
	cacheKey := loaderCacheKey("LoadTagPage", locale, r, filter.TagName)
	return framework.CachedCall(ctx, cacheKey, func(runCtx context.Context) (NotesPageView, error) {
		view, err := loadNotesListPage(
			runCtx,
			appCtx,
			r,
			locale,
			filter,
			notes.ListOptions{RequireTag: true},
			SidebarModeFiltered,
		)
		if err != nil {
			return NotesPageView{}, err
		}
		applyStructuredDataContextForNotesView(&view, appCtx, r, locale)
		view.EmptyStateMessage = i18nkeys.TEmptyTag(view.I18n())
		return view, nil
	})
}

func LoadNotesTalesPage(
	ctx context.Context,
	appCtx *Context,
	r *http.Request,
	_ framework.EmptyParams,
) (NotesPageView, error) {
	locale := localeFromRequest(appCtx, r)
	defaults := notes.ListFilter{Type: notes.NoteTypeLong}
	filter := listFilterFromQuery(r, defaults)
	filter.Type = notes.NoteTypeLong
	cacheKey := loaderCacheKey("LoadNotesTalesPage", locale, r)
	return framework.CachedCall(ctx, cacheKey, func(runCtx context.Context) (NotesPageView, error) {
		view, err := loadNotesListPage(runCtx, appCtx, r, locale, filter, notes.ListOptions{}, SidebarModeFiltered)
		if err != nil {
			return NotesPageView{}, err
		}
		applyStructuredDataContextForNotesView(&view, appCtx, r, locale)
		view.EmptyStateMessage = i18nkeys.TEmptyTales(view.I18n())
		return view, nil
	})
}

func LoadNotesMicroTalesPage(
	ctx context.Context,
	appCtx *Context,
	r *http.Request,
	_ framework.EmptyParams,
) (NotesPageView, error) {
	locale := localeFromRequest(appCtx, r)
	defaults := notes.ListFilter{Type: notes.NoteTypeShort}
	filter := listFilterFromQuery(r, defaults)
	filter.Type = notes.NoteTypeShort
	cacheKey := loaderCacheKey("LoadNotesMicroTalesPage", locale, r)
	return framework.CachedCall(ctx, cacheKey, func(runCtx context.Context) (NotesPageView, error) {
		view, err := loadNotesListPage(runCtx, appCtx, r, locale, filter, notes.ListOptions{}, SidebarModeFiltered)
		if err != nil {
			return NotesPageView{}, err
		}
		applyStructuredDataContextForNotesView(&view, appCtx, r, locale)
		view.EmptyStateMessage = i18nkeys.TEmptyMicro(view.I18n())
		return view, nil
	})
}

func LoadChannelsPage(
	ctx context.Context,
	appCtx *Context,
	r *http.Request,
	_ framework.EmptyParams,
) (NotesPageView, error) {
	locale := localeFromRequest(appCtx, r)
	filter := listFilterFromQuery(r, notes.ListFilter{})
	cacheKey := loaderCacheKey("LoadChannelsPage", locale, r)
	return framework.CachedCall(ctx, cacheKey, func(runCtx context.Context) (NotesPageView, error) {
		view, err := loadNotesListPage(runCtx, appCtx, r, locale, filter, notes.ListOptions{}, sidebarModeForFilter(filter))
		if err != nil {
			return NotesPageView{}, err
		}
		applyStructuredDataContextForNotesView(&view, appCtx, r, locale)

		view.PageTitle = i18nkeys.TChannelsPageTitle(view.I18n())
		return view, nil
	})
}

func loadNotesListPage(
	ctx context.Context,
	appCtx *Context,
	r *http.Request,
	locale string,
	filter notes.ListFilter,
	options notes.ListOptions,
	mode SidebarMode,
) (NotesPageView, error) {
	service, err := notesService(appCtx)
	if err != nil {
		return NotesPageView{}, err
	}

	result, err := service.ListNotes(ctx, locale, filter, options)
	if err != nil {
		return NotesPageView{}, err
	}

	return newNotesPageView(locale, appCtx.I18n(r), result, mode), nil
}

func LoadNotePage(
	ctx context.Context,
	appCtx *Context,
	r *http.Request,
	params framework.SlugParams,
) (NotePageView, error) {
	locale := localeFromRequest(appCtx, r)
	slug := strings.TrimSpace(params.Slug)
	cacheKey := loaderCacheKey("LoadNotePage", locale, r, slug)
	return framework.CachedCall(ctx, cacheKey, func(runCtx context.Context) (NotePageView, error) {
		service, err := notesService(appCtx)
		if err != nil {
			return NotePageView{}, err
		}

		rootURL := resolvedRootURL(appCtx, r)
		note, err := service.GetNoteBySlug(runCtx, locale, slug, noteSiteRootURLs(appCtx, rootURL))
		if err != nil {
			return NotePageView{}, err
		}
		i18n := appCtx.I18n(r)
		pageTitle := strings.TrimSpace(note.Title)

		return NotePageView{
			Locale:                locale,
			RootURL:               rootURL,
			CanonicalURL:          canonicalURLFromRequest(appCtx, r, locale),
			IncludeStructuredData: shouldIncludeStructuredData(r),
			I18nCtx:               i18n,
			PageTitle:             pageTitle,
			Note:                  *note,
			SidebarAuthorItems:    uniqueSortedAuthors(note.Authors),
			SidebarTagItems:       uniqueSortedTags(note.Tags),
			AnalyticsEnabled:      appCtx != nil && appCtx.LovelyEyeEnabled(),
		}, nil
	})
}

func listFilterFromQuery(r *http.Request, defaults notes.ListFilter) notes.ListFilter {
	if defaults.Page < 1 {
		defaults.Page = 1
	}

	query := url.Values{}
	if r != nil && r.URL != nil {
		query = r.URL.Query()
	}

	return listFilterFromValues(query, defaults)
}

func listFilterFromValues(query url.Values, defaults notes.ListFilter) notes.ListFilter {
	filter := notes.ListFilter{
		Page:       parsePage(query.Get("page")),
		AuthorSlug: strings.TrimSpace(query.Get("author")),
		TagName:    strings.TrimSpace(query.Get("tag")),
		Type:       notes.ParseNoteType(query.Get("type")),
		Query:      strings.TrimSpace(query.Get("q")),
	}

	if filter.Page < 1 {
		filter.Page = defaults.Page
	}
	if filter.AuthorSlug == "" {
		filter.AuthorSlug = strings.TrimSpace(defaults.AuthorSlug)
	}
	if filter.TagName == "" {
		filter.TagName = strings.TrimSpace(defaults.TagName)
	}
	if filter.Type == notes.NoteTypeAll {
		filter.Type = notes.ParseNoteType(string(defaults.Type))
	}
	if filter.Query == "" {
		filter.Query = strings.TrimSpace(defaults.Query)
	}

	return filter
}

func BuildRSSFeedURL(
	locale string,
	page int,
	authorSlug string,
	tagName string,
	noteType notes.NoteType,
	searchQuery string,
) string {
	if page < 1 {
		page = 1
	}

	noteType = notes.ParseNoteType(string(noteType))
	authorSlug = strings.TrimSpace(authorSlug)
	tagName = strings.TrimSpace(tagName)
	searchQuery = strings.TrimSpace(searchQuery)
	locale = normalizeLocaleForApp(locale)

	q := make(url.Values)
	q.Set("locale", locale)
	if page > 1 {
		q.Set("page", strconv.Itoa(page))
	}
	if authorSlug != "" {
		q.Set("author", authorSlug)
	}
	if tagName != "" {
		q.Set("tag", tagName)
	}
	if noteType == notes.NoteTypeLong || noteType == notes.NoteTypeShort {
		q.Set("type", noteType.QueryValue())
	}
	if searchQuery != "" {
		q.Set("q", searchQuery)
	}

	return rssEndpointPath + "?" + q.Encode()
}

func BuildNotesFilterURL(
	locale string,
	page int,
	authorSlug string,
	tagName string,
	noteType notes.NoteType,
	searchQuery string,
) string {
	if page < 1 {
		page = 1
	}

	noteType = notes.ParseNoteType(string(noteType))
	authorSlug = strings.TrimSpace(authorSlug)
	tagName = strings.TrimSpace(tagName)
	searchQuery = strings.TrimSpace(searchQuery)

	canonicalPath := canonicalNotesListingPath(authorSlug, tagName, noteType)
	if canonicalPath != "" && searchQuery == "" {
		if page == 1 {
			return LocalizeAppPath(locale, canonicalPath)
		}

		q := make(url.Values)
		q.Set("page", strconv.Itoa(page))
		return buildLocalizedPathWithQuery(locale, canonicalPath, q)
	}

	q := make(url.Values)
	if page > 1 {
		q.Set("page", strconv.Itoa(page))
	}
	if authorSlug != "" {
		q.Set("author", authorSlug)
	}
	if tagName != "" {
		q.Set("tag", tagName)
	}
	if noteType == notes.NoteTypeLong || noteType == notes.NoteTypeShort {
		q.Set("type", noteType.QueryValue())
	}
	if searchQuery != "" {
		q.Set("q", searchQuery)
	}

	encoded := q.Encode()
	if encoded == "" {
		return LocalizeAppPath(locale, "/")
	}

	return buildLocalizedPathWithQuery(locale, "/", q)
}

func CanonicalNotesRedirectURL(locale string, strippedPath string, query url.Values) (string, bool) {
	if !queryContainsOnlyCanonicalNotesParams(query) {
		return "", false
	}

	defaults, ok := canonicalNotesDefaultsForPath(strippedPath)
	if !ok {
		return "", false
	}

	filter := listFilterFromValues(query, defaults)
	enforceCanonicalNotesRouteFilters(strippedPath, &filter)
	if strings.TrimSpace(filter.Query) != "" || activeNotesListingFilterCount(filter) > 1 {
		return "", false
	}

	return BuildNotesFilterURL(locale, filter.Page, filter.AuthorSlug, filter.TagName, filter.Type, ""), true
}

func BuildChannelsURL(
	locale string,
	authorSlug string,
	tagName string,
	noteType notes.NoteType,
	searchQuery string,
) string {
	noteType = notes.ParseNoteType(string(noteType))
	authorSlug = strings.TrimSpace(authorSlug)
	tagName = strings.TrimSpace(tagName)
	searchQuery = strings.TrimSpace(searchQuery)

	q := make(url.Values)
	if authorSlug != "" {
		q.Set("author", authorSlug)
	}
	if tagName != "" {
		q.Set("tag", tagName)
	}
	if noteType == notes.NoteTypeLong || noteType == notes.NoteTypeShort {
		q.Set("type", noteType.QueryValue())
	}
	if searchQuery != "" {
		q.Set("q", searchQuery)
	}

	encoded := q.Encode()
	if encoded == "" {
		return LocalizeAppPath(locale, "/channels")
	}

	return buildLocalizedPathWithQuery(locale, "/channels", q)
}

func BuildAuthorURL(locale string, slug string, page int) string {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return LocalizeAppPath(locale, "/")
	}

	if page < 1 {
		page = 1
	}

	if page == 1 {
		return LocalizeAppPath(locale, "/author/"+slug)
	}

	q := make(url.Values)
	q.Set("page", strconv.Itoa(page))
	return buildLocalizedPathWithQuery(locale, "/author/"+slug, q)
}

func BuildHTMXNavigationURL(pageURL string) string {
	canonicalPath, query := normalizePageURL(pageURL)
	query.Set(liveNavigationQueryKey, liveNavigationQueryValue)

	encoded := query.Encode()
	if encoded == "" {
		return canonicalPath
	}

	return canonicalPath + "?" + encoded
}

func BuildTagURL(locale string, tagSlug string) string {
	tagSlug = strings.TrimSpace(tagSlug)
	if tagSlug == "" {
		return LocalizeAppPath(locale, "/")
	}

	return LocalizeAppPath(locale, "/tag/"+tagSlug)
}

func BuildTalesURL(locale string, page int, authorSlug string, tagName string) string {
	if page < 1 {
		page = 1
	}

	q := make(url.Values)
	if page > 1 {
		q.Set("page", strconv.Itoa(page))
	}
	if strings.TrimSpace(authorSlug) != "" {
		q.Set("author", strings.TrimSpace(authorSlug))
	}
	if strings.TrimSpace(tagName) != "" {
		q.Set("tag", strings.TrimSpace(tagName))
	}

	encoded := q.Encode()
	if encoded == "" {
		return LocalizeAppPath(locale, "/tales")
	}

	return buildLocalizedPathWithQuery(locale, "/tales", q)
}

func BuildMicroTalesURL(locale string, page int, authorSlug string, tagName string) string {
	if page < 1 {
		page = 1
	}

	q := make(url.Values)
	if page > 1 {
		q.Set("page", strconv.Itoa(page))
	}
	if strings.TrimSpace(authorSlug) != "" {
		q.Set("author", strings.TrimSpace(authorSlug))
	}
	if strings.TrimSpace(tagName) != "" {
		q.Set("tag", strings.TrimSpace(tagName))
	}

	encoded := q.Encode()
	if encoded == "" {
		return LocalizeAppPath(locale, "/micro-tales")
	}

	return buildLocalizedPathWithQuery(locale, "/micro-tales", q)
}

func canonicalNotesListingPath(authorSlug string, tagName string, noteType notes.NoteType) string {
	authorSlug = strings.TrimSpace(authorSlug)
	tagName = strings.TrimSpace(tagName)
	noteType = notes.ParseNoteType(string(noteType))

	if authorSlug != "" && tagName == "" && noteType == notes.NoteTypeAll {
		return "/author/" + authorSlug
	}
	if tagName != "" && authorSlug == "" && noteType == notes.NoteTypeAll {
		return "/tag/" + tagName
	}
	if noteType == notes.NoteTypeLong && authorSlug == "" && tagName == "" {
		return "/tales"
	}
	if noteType == notes.NoteTypeShort && authorSlug == "" && tagName == "" {
		return "/micro-tales"
	}

	return ""
}

func activeNotesListingFilterCount(filter notes.ListFilter) int {
	count := 0
	if strings.TrimSpace(filter.AuthorSlug) != "" {
		count++
	}
	if strings.TrimSpace(filter.TagName) != "" {
		count++
	}
	if notes.ParseNoteType(string(filter.Type)) != notes.NoteTypeAll {
		count++
	}

	return count
}

func queryContainsOnlyCanonicalNotesParams(query url.Values) bool {
	for key := range query {
		switch strings.TrimSpace(key) {
		case "page", "author", "tag", "type", "q":
			continue
		default:
			return false
		}
	}

	return true
}

func canonicalNotesDefaultsForPath(pathValue string) (notes.ListFilter, bool) {
	normalizedPath := frameworki18n.NormalizePath(pathValue)

	switch normalizedPath {
	case "/":
		return notes.ListFilter{}, true
	case "/tales":
		return notes.ListFilter{Type: notes.NoteTypeLong}, true
	case "/micro-tales":
		return notes.ListFilter{Type: notes.NoteTypeShort}, true
	}

	if slug, ok := canonicalNotesSlugForPath(normalizedPath, "/author/"); ok {
		return notes.ListFilter{AuthorSlug: slug}, true
	}
	if slug, ok := canonicalNotesSlugForPath(normalizedPath, "/tag/"); ok {
		return notes.ListFilter{TagName: slug}, true
	}

	return notes.ListFilter{}, false
}

func canonicalNotesSlugForPath(pathValue string, prefix string) (string, bool) {
	if !strings.HasPrefix(pathValue, prefix) {
		return "", false
	}

	slug := strings.TrimSpace(strings.TrimPrefix(pathValue, prefix))
	if slug == "" || strings.Contains(slug, "/") {
		return "", false
	}

	return slug, true
}

func enforceCanonicalNotesRouteFilters(pathValue string, filter *notes.ListFilter) {
	if filter == nil {
		return
	}

	normalizedPath := frameworki18n.NormalizePath(pathValue)
	switch normalizedPath {
	case "/tales":
		filter.Type = notes.NoteTypeLong
		return
	case "/micro-tales":
		filter.Type = notes.NoteTypeShort
		return
	}

	if slug, ok := canonicalNotesSlugForPath(normalizedPath, "/author/"); ok {
		filter.AuthorSlug = slug
		return
	}
	if slug, ok := canonicalNotesSlugForPath(normalizedPath, "/tag/"); ok {
		filter.TagName = slug
	}
}

func normalizePageURL(pageURL string) (string, url.Values) {
	parsed, err := url.Parse(strings.TrimSpace(pageURL))
	if err != nil {
		return "/", make(url.Values)
	}

	pathValue := strings.TrimSpace(parsed.Path)
	if pathValue == "" {
		pathValue = "/"
	}
	if !strings.HasPrefix(pathValue, "/") {
		pathValue = "/" + pathValue
	}

	return pathValue, parsed.Query()
}

func applyStructuredDataContextForNotesView(
	view *NotesPageView,
	appCtx *Context,
	r *http.Request,
	locale string,
) {
	if view == nil {
		return
	}

	view.RootURL = resolvedRootURL(appCtx, r)
	view.AnalyticsEnabled = appCtx != nil && appCtx.LovelyEyeEnabled()
	view.CanonicalURL = canonicalURLFromRequest(appCtx, r, locale)
	view.IncludeStructuredData = shouldIncludeStructuredData(r)
}

func shouldIncludeStructuredData(r *http.Request) bool {
	if r == nil {
		return true
	}

	if strings.EqualFold(strings.TrimSpace(r.Header.Get("HX-Request")), "true") {
		return false
	}

	if r.URL == nil {
		return true
	}

	return strings.TrimSpace(r.URL.Query().Get(liveNavigationQueryKey)) != liveNavigationQueryValue
}

func canonicalURLFromRequest(appCtx *Context, r *http.Request, locale string) string {
	if appCtx == nil || r == nil {
		return ""
	}

	rootURL := resolvedRootURL(appCtx, r)
	if rootURL == "" {
		return ""
	}

	cfg, err := frameworki18n.NormalizeConfig(appCtx.I18nConfig())
	if err != nil {
		return ""
	}

	pathValue := "/"
	if r.URL != nil {
		pathValue = strings.TrimSpace(r.URL.Path)
		if pathValue == "" {
			pathValue = "/"
		}
		if strings.TrimSpace(r.URL.RawQuery) != "" {
			pathValue += "?" + strings.TrimSpace(r.URL.RawQuery)
		}
	}

	alternates, err := metagen.BuildAlternates(rootURL, cfg, locale, pathValue, nil)
	if err != nil {
		return ""
	}

	return strings.TrimSpace(alternates.Canonical)
}

func resolvedRootURL(appCtx *Context, r *http.Request) string {
	if appCtx == nil {
		return ""
	}
	return appCtx.ResolveRootURL(r)
}

func noteSiteRootURLs(appCtx *Context, resolvedRootURL string) []string {
	roots := make([]string, 0, 2)
	seen := make(map[string]struct{}, 2)

	appendRoot := func(value string) {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return
		}
		if _, ok := seen[trimmed]; ok {
			return
		}
		seen[trimmed] = struct{}{}
		roots = append(roots, trimmed)
	}

	appendRoot(resolvedRootURL)
	if appCtx != nil && appCtx.SiteResolver() != nil {
		appendRoot(appCtx.SiteResolver().CanonicalURL())
	}

	return roots
}

func parsePage(value string) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return 1
	}
	return parsed
}

func localeFromRequest(appCtx *Context, r *http.Request) string {
	requestLocale := ""
	if r != nil {
		requestLocale = frameworki18n.LocaleFromContext(r.Context())
	}
	if appCtx == nil {
		return normalizeLocaleForApp(requestLocale)
	}
	return appCtx.LocaleFromRequest(requestLocale)
}

func buildLocalizedPathWithQuery(locale string, strippedPath string, query url.Values) string {
	localizedPath := LocalizeAppPath(locale, strippedPath)
	encoded := query.Encode()
	if strings.TrimSpace(encoded) == "" {
		return localizedPath
	}
	return localizedPath + "?" + encoded
}

func sidebarModeForFilter(filter notes.ListFilter) SidebarMode {
	if strings.TrimSpace(filter.Query) != "" {
		return SidebarModeFiltered
	}

	if strings.TrimSpace(filter.AuthorSlug) != "" || strings.TrimSpace(filter.TagName) != "" {
		return SidebarModeFiltered
	}

	if notes.ParseNoteType(string(filter.Type)) != notes.NoteTypeAll {
		return SidebarModeFiltered
	}

	return SidebarModeRoot
}

func notesService(appCtx *Context) (*notes.Service, error) {
	if appCtx == nil || appCtx.service == nil {
		return nil, errNotesServiceUnavailable
	}
	return appCtx.service, nil
}

func loaderCacheKey(loaderName string, locale string, r *http.Request, fragments ...string) string {
	pathValue := "/"
	queryValue := ""
	if r != nil && r.URL != nil {
		pathValue = strings.TrimSpace(r.URL.Path)
		if pathValue == "" {
			pathValue = "/"
		}
		queryValue = strings.TrimSpace(r.URL.RawQuery)
	}

	keyParts := []string{"runtime", strings.TrimSpace(loaderName), strings.TrimSpace(locale), pathValue, queryValue}
	for _, fragment := range fragments {
		keyParts = append(keyParts, strings.TrimSpace(fragment))
	}

	return strings.Join(keyParts, "|")
}
