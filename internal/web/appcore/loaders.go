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
	service, err := notesService(appCtx)
	if err != nil {
		return NotesPageView{}, err
	}

	state := NotesSignalState{
		Page: parsePage(r.URL.Query().Get("page")),
		Tag:  strings.TrimSpace(r.URL.Query().Get("tag")),
	}

	result, err := service.ListNotes(ctx, state.Page, state.Tag)
	if err != nil {
		return NotesPageView{}, err
	}

	return newNotesPageView(result), nil
}

func LoadNotesLivePage(
	ctx context.Context,
	appCtx *Context,
	_ *http.Request,
	_ framework.EmptyParams,
	state NotesSignalState,
) (NotesPageView, error) {
	service, err := notesService(appCtx)
	if err != nil {
		return NotesPageView{}, err
	}

	result, err := service.ListNotes(ctx, sanitizePage(state.Page), strings.TrimSpace(state.Tag))
	if err != nil {
		return NotesPageView{}, err
	}

	return newNotesPageView(result), nil
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

	return NotePageView{
		PageTitle:          note.Title,
		Note:               *note,
		SidebarAuthorItems: uniqueSortedAuthors(note.Authors),
		SidebarTagItems:    uniqueSortedTags(note.Tags),
	}, nil
}

func LoadAuthorPage(
	ctx context.Context,
	appCtx *Context,
	r *http.Request,
	params framework.SlugParams,
) (AuthorPageView, error) {
	service, err := notesService(appCtx)
	if err != nil {
		return AuthorPageView{}, err
	}

	result, err := service.GetAuthorPage(ctx, params.Slug, parsePage(r.URL.Query().Get("page")))
	if err != nil {
		return AuthorPageView{}, err
	}

	return newAuthorPageView(result), nil
}

func LoadAuthorLivePage(
	ctx context.Context,
	appCtx *Context,
	_ *http.Request,
	params framework.SlugParams,
	state AuthorSignalState,
) (AuthorPageView, error) {
	service, err := notesService(appCtx)
	if err != nil {
		return AuthorPageView{}, err
	}

	result, err := service.GetAuthorPage(ctx, params.Slug, sanitizePage(state.Page))
	if err != nil {
		return AuthorPageView{}, err
	}

	return newAuthorPageView(result), nil
}

func ParseNotesLiveState(r *http.Request) (NotesSignalState, error) {
	fallback := NotesSignalState{
		Page: parsePage(r.URL.Query().Get("page")),
		Tag:  strings.TrimSpace(r.URL.Query().Get("tag")),
	}

	state, err := readDatastarState(r, fallback)
	if err != nil {
		return NotesSignalState{}, err
	}
	state.Page = sanitizePage(state.Page)
	state.Tag = strings.TrimSpace(state.Tag)

	return state, nil
}

func ParseAuthorLiveState(r *http.Request) (AuthorSignalState, error) {
	fallback := AuthorSignalState{Page: parsePage(r.URL.Query().Get("page"))}
	state, err := readDatastarState(r, fallback)
	if err != nil {
		return AuthorSignalState{}, err
	}
	state.Page = sanitizePage(state.Page)

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

func BuildNotesURL(page int, tag string) string {
	if page < 1 {
		page = 1
	}

	q := make(url.Values)
	if page > 1 {
		q.Set("page", strconv.Itoa(page))
	}
	if strings.TrimSpace(tag) != "" {
		q.Set("tag", tag)
	}

	encoded := q.Encode()
	if encoded == "" {
		return "/notes"
	}
	return "/notes?" + encoded
}

func BuildAuthorURL(slug string, page int) string {
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

func notesService(appCtx *Context) (*notes.Service, error) {
	if appCtx == nil || appCtx.service == nil {
		return nil, errNotesServiceUnavailable
	}
	return appCtx.service, nil
}
