package appcore

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"blog/framework"
	frameworki18n "blog/framework/i18n"
	"blog/internal/notes"
	webi18n "blog/internal/web/i18n"
)

const liveNavigationQueryKey = "__live"
const liveNavigationQueryValue = "navigation"

func LoadNotesPage(
	ctx context.Context,
	appCtx *Context,
	r *http.Request,
	_ framework.EmptyParams,
) (NotesPageView, error) {
	locale := localeFromRequest(appCtx, r)
	filter := listFilterFromQuery(r, notes.ListFilter{})
	view, err := loadNotesListPage(
		ctx,
		appCtx,
		locale,
		filter,
		notes.ListOptions{},
		sidebarModeForFilter(filter),
	)
	if err != nil {
		return NotesPageView{}, err
	}
	view.EmptyStateMessage = Message(view.Messages, webi18n.KeyEmptyRoot)
	return view, nil
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

	view, err := loadNotesListPage(
		ctx,
		appCtx,
		locale,
		filter,
		notes.ListOptions{RequireAuthor: true},
		SidebarModeFiltered,
	)
	if err != nil {
		return AuthorPageView{}, err
	}

	view.EmptyStateMessage = Message(view.Messages, webi18n.KeyEmptyAuthor)
	return view, nil
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

	view, err := loadNotesListPage(
		ctx,
		appCtx,
		locale,
		filter,
		notes.ListOptions{RequireTag: true},
		SidebarModeFiltered,
	)
	if err != nil {
		return NotesPageView{}, err
	}
	view.EmptyStateMessage = Message(view.Messages, webi18n.KeyEmptyTag)
	return view, nil
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

	view, err := loadNotesListPage(ctx, appCtx, locale, filter, notes.ListOptions{}, SidebarModeFiltered)
	if err != nil {
		return NotesPageView{}, err
	}
	view.EmptyStateMessage = Message(view.Messages, webi18n.KeyEmptyTales)
	return view, nil
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

	view, err := loadNotesListPage(ctx, appCtx, locale, filter, notes.ListOptions{}, SidebarModeFiltered)
	if err != nil {
		return NotesPageView{}, err
	}
	view.EmptyStateMessage = Message(view.Messages, webi18n.KeyEmptyMicro)
	return view, nil
}

func LoadChannelsPage(
	ctx context.Context,
	appCtx *Context,
	r *http.Request,
	_ framework.EmptyParams,
) (NotesPageView, error) {
	locale := localeFromRequest(appCtx, r)
	filter := listFilterFromQuery(r, notes.ListFilter{})
	view, err := loadNotesListPage(ctx, appCtx, locale, filter, notes.ListOptions{}, sidebarModeForFilter(filter))
	if err != nil {
		return NotesPageView{}, err
	}

	view.PageTitle = Message(view.Messages, webi18n.KeyChannelsPageTitle)
	return view, nil
}

func loadNotesListPage(
	ctx context.Context,
	appCtx *Context,
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

	return newNotesPageView(locale, localizedMessages(appCtx, locale), result, mode), nil
}

func LoadNotePage(
	ctx context.Context,
	appCtx *Context,
	r *http.Request,
	params framework.SlugParams,
) (NotePageView, error) {
	locale := localeFromRequest(appCtx, r)
	service, err := notesService(appCtx)
	if err != nil {
		return NotePageView{}, err
	}

	note, err := service.GetNoteBySlug(ctx, locale, params.Slug)
	if err != nil {
		return NotePageView{}, err
	}
	messages := localizedMessages(appCtx, locale)
	pageTitle := strings.TrimSpace(note.Title)
	if pageTitle == "" {
		pageTitle = Message(messages, webi18n.KeyNoteTitleFallback)
	}

	return NotePageView{
		Locale:             locale,
		Messages:           messages,
		PageTitle:          pageTitle,
		Note:               *note,
		SidebarAuthorItems: uniqueSortedAuthors(note.Authors),
		SidebarTagItems:    uniqueSortedTags(note.Tags),
	}, nil
}

func listFilterFromQuery(r *http.Request, defaults notes.ListFilter) notes.ListFilter {
	if defaults.Page < 1 {
		defaults.Page = 1
	}

	query := url.Values{}
	if r != nil && r.URL != nil {
		query = r.URL.Query()
	}

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

func BuildNotesURL(locale string, page int, tag string, searchQuery string) string {
	return BuildNotesFilterURL(locale, page, "", tag, notes.NoteTypeAll, searchQuery)
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
