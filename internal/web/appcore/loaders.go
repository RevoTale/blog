package appcore

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"blog/framework"
	"blog/internal/notes"
	"github.com/starfederation/datastar-go/datastar"
)

func LoadNotesPage(
	ctx context.Context,
	appCtx *Context,
	r *http.Request,
	_ framework.EmptyParams,
) (NotesPageView, error) {
	filter := listFilterFromQuery(r, notes.ListFilter{})
	return loadNotesListPage(ctx, appCtx, filter, notes.ListOptions{}, sidebarModeForFilter(filter))
}

func LoadNotesLivePage(
	ctx context.Context,
	appCtx *Context,
	r *http.Request,
	_ framework.EmptyParams,
	state NotesSignalState,
) (NotesPageView, error) {
	fallback := listFilterFromQuery(r, notes.ListFilter{})
	filter := notes.ListFilter{
		Page:       sanitizePage(state.Page),
		AuthorSlug: cleanOrFallback(state.Author, fallback.AuthorSlug),
		TagName:    cleanOrFallback(state.Tag, fallback.TagName),
		Type:       notes.ParseNoteType(cleanOrFallback(state.Type, string(fallback.Type))),
	}

	return loadNotesListPage(ctx, appCtx, filter, notes.ListOptions{}, sidebarModeForFilter(filter))
}

func LoadAuthorPage(
	ctx context.Context,
	appCtx *Context,
	r *http.Request,
	params framework.SlugParams,
) (AuthorPageView, error) {
	defaults := notes.ListFilter{AuthorSlug: params.Slug}
	filter := listFilterFromQuery(r, defaults)
	filter.AuthorSlug = strings.TrimSpace(params.Slug)

	view, err := loadNotesListPage(ctx, appCtx, filter, notes.ListOptions{RequireAuthor: true}, SidebarModeFiltered)
	if err != nil {
		return AuthorPageView{}, err
	}

	return view, nil
}

func LoadAuthorLivePage(
	ctx context.Context,
	appCtx *Context,
	r *http.Request,
	params framework.SlugParams,
	state AuthorSignalState,
) (AuthorPageView, error) {
	fallback := listFilterFromQuery(r, notes.ListFilter{AuthorSlug: params.Slug})
	filter := notes.ListFilter{
		Page:       sanitizePage(state.Page),
		AuthorSlug: params.Slug,
		TagName:    cleanOrFallback(state.Tag, fallback.TagName),
		Type:       notes.ParseNoteType(cleanOrFallback(state.Type, string(fallback.Type))),
	}

	view, err := loadNotesListPage(ctx, appCtx, filter, notes.ListOptions{RequireAuthor: true}, SidebarModeFiltered)
	if err != nil {
		return AuthorPageView{}, err
	}

	return view, nil
}

func LoadTagPage(
	ctx context.Context,
	appCtx *Context,
	r *http.Request,
	params framework.SlugParams,
) (NotesPageView, error) {
	defaults := notes.ListFilter{TagName: params.Slug}
	filter := listFilterFromQuery(r, defaults)
	filter.TagName = strings.TrimSpace(params.Slug)

	return loadNotesListPage(ctx, appCtx, filter, notes.ListOptions{RequireTag: true}, SidebarModeFiltered)
}

func LoadNotesTalesPage(
	ctx context.Context,
	appCtx *Context,
	r *http.Request,
	_ framework.EmptyParams,
) (NotesPageView, error) {
	defaults := notes.ListFilter{Type: notes.NoteTypeLong}
	filter := listFilterFromQuery(r, defaults)
	filter.Type = notes.NoteTypeLong

	return loadNotesListPage(ctx, appCtx, filter, notes.ListOptions{}, SidebarModeFiltered)
}

func LoadNotesMicroTalesPage(
	ctx context.Context,
	appCtx *Context,
	r *http.Request,
	_ framework.EmptyParams,
) (NotesPageView, error) {
	defaults := notes.ListFilter{Type: notes.NoteTypeShort}
	filter := listFilterFromQuery(r, defaults)
	filter.Type = notes.NoteTypeShort

	return loadNotesListPage(ctx, appCtx, filter, notes.ListOptions{}, SidebarModeFiltered)
}

func LoadChannelsPage(
	ctx context.Context,
	appCtx *Context,
	r *http.Request,
	_ framework.EmptyParams,
) (NotesPageView, error) {
	filter := listFilterFromQuery(r, notes.ListFilter{})
	view, err := loadNotesListPage(ctx, appCtx, filter, notes.ListOptions{}, sidebarModeForFilter(filter))
	if err != nil {
		return NotesPageView{}, err
	}

	view.PageTitle = "Channels"
	return view, nil
}

func loadNotesListPage(
	ctx context.Context,
	appCtx *Context,
	filter notes.ListFilter,
	options notes.ListOptions,
	mode SidebarMode,
) (NotesPageView, error) {
	service, err := notesService(appCtx)
	if err != nil {
		return NotesPageView{}, err
	}

	result, err := service.ListNotes(ctx, filter, options)
	if err != nil {
		return NotesPageView{}, err
	}

	return newNotesPageView(result, mode), nil
}

func LoadNotePage(
	ctx context.Context,
	appCtx *Context,
	_ *http.Request,
	params framework.SlugParams,
) (NotePageView, error) {
	service, err := notesService(appCtx)
	if err != nil {
		return NotePageView{}, err
	}

	note, err := service.GetNoteBySlug(ctx, params.Slug)
	if err != nil {
		return NotePageView{}, err
	}
	pageTitle := strings.TrimSpace(note.Title)
	if pageTitle == "" {
		pageTitle = "Note"
	}

	return NotePageView{
		PageTitle:          pageTitle,
		Note:               *note,
		SidebarAuthorItems: uniqueSortedAuthors(note.Authors),
		SidebarTagItems:    uniqueSortedTags(note.Tags),
	}, nil
}

func ParseNotesLiveState(r *http.Request) (NotesSignalState, error) {
	fallbackFilter := listFilterFromQuery(r, notes.ListFilter{})
	fallback := NotesSignalState{
		Page:   fallbackFilter.Page,
		Author: fallbackFilter.AuthorSlug,
		Tag:    fallbackFilter.TagName,
		Type:   string(fallbackFilter.Type),
	}

	state, err := readDatastarState(r, fallback)
	if err != nil {
		return NotesSignalState{}, err
	}
	state.Page = sanitizePage(state.Page)
	state.Author = strings.TrimSpace(state.Author)
	state.Tag = strings.TrimSpace(state.Tag)
	state.Type = string(notes.ParseNoteType(state.Type))

	return state, nil
}

func ParseAuthorLiveState(r *http.Request) (AuthorSignalState, error) {
	fallbackFilter := listFilterFromQuery(r, notes.ListFilter{})
	fallback := AuthorSignalState{
		Page: fallbackFilter.Page,
		Tag:  fallbackFilter.TagName,
		Type: string(fallbackFilter.Type),
	}

	state, err := readDatastarState(r, fallback)
	if err != nil {
		return AuthorSignalState{}, err
	}
	state.Page = sanitizePage(state.Page)
	state.Tag = strings.TrimSpace(state.Tag)
	state.Type = string(notes.ParseNoteType(state.Type))

	return state, nil
}

func readDatastarState[T interface{}](r *http.Request, fallback T) (T, error) {
	if r.Method == http.MethodGet && strings.TrimSpace(r.URL.Query().Get(datastar.DatastarKey)) == "" {
		return fallback, nil
	}

	parsed := fallback
	if err := datastar.ReadSignals(r, &parsed); err != nil {
		return fallback, err
	}

	return parsed, nil
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

	return filter
}

func cleanOrFallback(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return strings.TrimSpace(fallback)
	}

	return trimmed
}

func BuildNotesURL(page int, tag string) string {
	return BuildNotesFilterURL(page, "", tag, notes.NoteTypeAll)
}

func BuildNotesFilterURL(page int, authorSlug string, tagName string, noteType notes.NoteType) string {
	if page < 1 {
		page = 1
	}

	noteType = notes.ParseNoteType(string(noteType))
	authorSlug = strings.TrimSpace(authorSlug)
	tagName = strings.TrimSpace(tagName)

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

	encoded := q.Encode()
	if encoded == "" {
		return "/notes"
	}

	return "/notes?" + encoded
}

func BuildChannelsURL(authorSlug string, tagName string, noteType notes.NoteType) string {
	noteType = notes.ParseNoteType(string(noteType))
	authorSlug = strings.TrimSpace(authorSlug)
	tagName = strings.TrimSpace(tagName)

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

	encoded := q.Encode()
	if encoded == "" {
		return "/channels"
	}

	return "/channels?" + encoded
}

func BuildAuthorURL(slug string, page int) string {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return "/notes"
	}

	if page < 1 {
		page = 1
	}

	if page == 1 {
		return "/author/" + slug
	}

	q := make(url.Values)
	q.Set("page", strconv.Itoa(page))
	return "/author/" + slug + "?" + q.Encode()
}

func BuildTagURL(tagSlug string) string {
	tagSlug = strings.TrimSpace(tagSlug)
	if tagSlug == "" {
		return "/notes"
	}

	return "/tag/" + tagSlug
}

func BuildTalesURL(page int, authorSlug string, tagName string) string {
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
		return "/notes/tales"
	}

	return "/notes/tales?" + encoded
}

func BuildMicroTalesURL(page int, authorSlug string, tagName string) string {
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
		return "/notes/micro-tales"
	}

	return "/notes/micro-tales?" + encoded
}

func BuildAuthorLiveURL(slug string) string {
	return "/author/" + slug + "/live"
}

func parsePage(value string) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return 1
	}
	return parsed
}

func sanitizePage(page int) int {
	if page < 1 {
		return 1
	}

	return page
}

func sidebarModeForFilter(filter notes.ListFilter) SidebarMode {
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
