package runtime

import (
	"sort"
	"strings"

	"blog/internal/notes"
	webi18n "blog/web/i18n"
)

type SidebarMode string

const (
	SidebarModeRoot     SidebarMode = "root"
	SidebarModeFiltered SidebarMode = "filtered"
)

type RootLayoutView interface {
	LocaleCode() string
	MessagesMap() map[webi18n.Key]string
	LayoutPageTitle() string
	LayoutSearchQuery() string
	LovelyEyeEnabled() bool
	RSSFeedURL() string
	SidebarAuthors() []notes.Author
	SidebarTags() []notes.Tag
	SidebarCurrentAuthorSlug() string
	SidebarCurrentTagName() string
	SidebarCurrentType() notes.NoteType
	SidebarChannelsURL() string
	SidebarAllURL() string
	SidebarAnyAuthorURL() string
	SidebarAnyTagURL() string
	SidebarAnyTypeURL() string
	SidebarAuthorURL(authorSlug string) string
	SidebarTagURL(tagName string) string
	SidebarTypeURL(noteType notes.NoteType) string
}

type PaginationView struct {
	Page       int
	TotalPages int
	HasPrev    bool
	HasNext    bool
	FirstPage  int
	LastPage   int
	PrevPage   int
	NextPage   int
	FirstURL   string
	LastURL    string
	PrevURL    string
	NextURL    string
}

type NotesPageView struct {
	Locale                string
	RootURL               string
	CanonicalURL          string
	IncludeStructuredData bool
	Messages              map[webi18n.Key]string
	PageTitle             string
	Filter                notes.ListFilter
	SidebarMode           SidebarMode
	Notes                 []notes.NoteSummary
	Authors               []notes.Author
	Tags                  []notes.Tag
	ActiveAuthor          *notes.Author
	ActiveTag             *notes.Tag
	Pagination            PaginationView
	ContextTitle          string
	ContextSubtitle       string
	ContextDescription    string
	EmptyStateMessage     string
	AnalyticsEnabled      bool
}

type AuthorPageView = NotesPageView

type NotePageView struct {
	Locale                string
	RootURL               string
	CanonicalURL          string
	IncludeStructuredData bool
	Messages              map[webi18n.Key]string
	PageTitle             string
	Note                  notes.NoteDetail
	SidebarAuthorItems    []notes.Author
	SidebarTagItems       []notes.Tag
	AnalyticsEnabled      bool
}

func newFallbackView(locale string) RootLayoutView {
	locale = normalizeLocaleForApp(locale)
	messages := localizedMessages(nil, locale)

	return NotesPageView{
		Locale:      locale,
		Messages:    messages,
		PageTitle:   Message(messages, webi18n.KeyNotfoundPageTitle),
		SidebarMode: SidebarModeRoot,
		Filter: notes.ListFilter{
			Type: notes.NoteTypeAll,
		},
	}
}

func NewNotFoundView(locale string) RootLayoutView {
	return newFallbackView(locale)
}

func NewErrorView(locale string) RootLayoutView {
	return newFallbackView(locale)
}

func (v NotesPageView) LocaleCode() string {
	return normalizeLocaleForApp(v.Locale)
}

func (v NotesPageView) MessagesMap() map[webi18n.Key]string {
	return v.Messages
}

func (v NotesPageView) LayoutPageTitle() string {
	return v.PageTitle
}

func (v NotesPageView) LayoutSearchQuery() string {
	return strings.TrimSpace(v.Filter.Query)
}

func (v NotesPageView) LovelyEyeEnabled() bool {
	return v.AnalyticsEnabled
}

func (v NotesPageView) RSSFeedURL() string {
	return BuildRSSFeedURL(
		v.LocaleCode(),
		v.Filter.Page,
		v.Filter.AuthorSlug,
		v.Filter.TagName,
		v.Filter.Type,
		v.Filter.Query,
	)
}

func (v NotesPageView) SidebarAuthors() []notes.Author {
	return v.Authors
}

func (v NotesPageView) SidebarTags() []notes.Tag {
	return v.Tags
}

func (v NotesPageView) SidebarCurrentAuthorSlug() string {
	return v.Filter.AuthorSlug
}

func (v NotesPageView) SidebarCurrentTagName() string {
	return v.Filter.TagName
}

func (v NotesPageView) SidebarCurrentType() notes.NoteType {
	return v.Filter.Type
}

func (v NotesPageView) SidebarChannelsURL() string {
	return BuildChannelsURL(v.LocaleCode(), v.Filter.AuthorSlug, v.Filter.TagName, v.Filter.Type, v.Filter.Query)
}

func (v NotesPageView) SidebarAllURL() string {
	return BuildNotesFilterURL(v.LocaleCode(), 1, "", "", notes.NoteTypeAll, v.Filter.Query)
}

func (v NotesPageView) SidebarAnyAuthorURL() string {
	if v.SidebarMode == SidebarModeRoot {
		return BuildNotesFilterURL(v.LocaleCode(), 1, "", "", notes.NoteTypeAll, v.Filter.Query)
	}

	return BuildNotesFilterURL(v.LocaleCode(), 1, "", v.Filter.TagName, v.Filter.Type, v.Filter.Query)
}

func (v NotesPageView) SidebarAnyTagURL() string {
	if v.SidebarMode == SidebarModeRoot {
		return BuildNotesFilterURL(v.LocaleCode(), 1, "", "", notes.NoteTypeAll, v.Filter.Query)
	}

	return BuildNotesFilterURL(v.LocaleCode(), 1, v.Filter.AuthorSlug, "", v.Filter.Type, v.Filter.Query)
}

func (v NotesPageView) SidebarAnyTypeURL() string {
	if v.SidebarMode == SidebarModeRoot {
		return BuildNotesFilterURL(v.LocaleCode(), 1, "", "", notes.NoteTypeAll, v.Filter.Query)
	}

	return BuildNotesFilterURL(v.LocaleCode(), 1, v.Filter.AuthorSlug, v.Filter.TagName, notes.NoteTypeAll, v.Filter.Query)
}

func (v NotesPageView) SidebarAuthorURL(authorSlug string) string {
	authorSlug = strings.TrimSpace(authorSlug)
	if authorSlug == "" {
		return v.SidebarAnyAuthorURL()
	}

	if v.SidebarMode == SidebarModeRoot {
		return BuildAuthorURL(v.LocaleCode(), authorSlug, 1)
	}

	return BuildNotesFilterURL(v.LocaleCode(), 1, authorSlug, v.Filter.TagName, v.Filter.Type, v.Filter.Query)
}

func (v NotesPageView) SidebarTagURL(tagName string) string {
	tagName = strings.TrimSpace(tagName)
	if tagName == "" {
		return v.SidebarAnyTagURL()
	}

	if v.SidebarMode == SidebarModeRoot {
		return BuildTagURL(v.LocaleCode(), tagName)
	}

	return BuildNotesFilterURL(v.LocaleCode(), 1, v.Filter.AuthorSlug, tagName, v.Filter.Type, v.Filter.Query)
}

func (v NotesPageView) SidebarTypeURL(noteType notes.NoteType) string {
	noteType = notes.ParseNoteType(string(noteType))
	if noteType == notes.NoteTypeAll {
		return v.SidebarAnyTypeURL()
	}

	if v.SidebarMode == SidebarModeRoot {
		if noteType == notes.NoteTypeLong {
			return BuildTalesURL(v.LocaleCode(), 1, "", "")
		}

		if noteType == notes.NoteTypeShort {
			return BuildMicroTalesURL(v.LocaleCode(), 1, "", "")
		}
	}

	return BuildNotesFilterURL(v.LocaleCode(), 1, v.Filter.AuthorSlug, v.Filter.TagName, noteType, v.Filter.Query)
}

func (v NotePageView) LocaleCode() string {
	return normalizeLocaleForApp(v.Locale)
}

func (v NotePageView) MessagesMap() map[webi18n.Key]string {
	return v.Messages
}

func (v NotePageView) LayoutPageTitle() string {
	return v.PageTitle
}

func (v NotePageView) LayoutSearchQuery() string {
	return ""
}

func (v NotePageView) LovelyEyeEnabled() bool {
	return v.AnalyticsEnabled
}

func (v NotePageView) RSSFeedURL() string {
	return BuildRSSFeedURL(v.LocaleCode(), 1, "", "", notes.NoteTypeAll, "")
}

func (v NotePageView) SidebarAuthors() []notes.Author {
	return v.SidebarAuthorItems
}

func (v NotePageView) SidebarTags() []notes.Tag {
	return v.SidebarTagItems
}

func (v NotePageView) SidebarCurrentAuthorSlug() string {
	return ""
}

func (v NotePageView) SidebarCurrentTagName() string {
	return ""
}

func (v NotePageView) SidebarCurrentType() notes.NoteType {
	return notes.NoteTypeAll
}

func (v NotePageView) SidebarChannelsURL() string {
	return LocalizeAppPath(v.LocaleCode(), "/channels")
}

func (v NotePageView) SidebarAllURL() string {
	return LocalizeAppPath(v.LocaleCode(), "/")
}

func (v NotePageView) SidebarAnyAuthorURL() string {
	return LocalizeAppPath(v.LocaleCode(), "/")
}

func (v NotePageView) SidebarAnyTagURL() string {
	return LocalizeAppPath(v.LocaleCode(), "/")
}

func (v NotePageView) SidebarAnyTypeURL() string {
	return LocalizeAppPath(v.LocaleCode(), "/")
}

func (v NotePageView) SidebarAuthorURL(authorSlug string) string {
	return BuildAuthorURL(v.LocaleCode(), authorSlug, 1)
}

func (v NotePageView) SidebarTagURL(tagName string) string {
	return BuildTagURL(v.LocaleCode(), tagName)
}

func (v NotePageView) SidebarTypeURL(noteType notes.NoteType) string {
	noteType = notes.ParseNoteType(string(noteType))
	if noteType == notes.NoteTypeLong {
		return BuildTalesURL(v.LocaleCode(), 1, "", "")
	}
	if noteType == notes.NoteTypeShort {
		return BuildMicroTalesURL(v.LocaleCode(), 1, "", "")
	}

	return LocalizeAppPath(v.LocaleCode(), "/")
}

func newNotesPageView(
	locale string,
	messages map[webi18n.Key]string,
	result notes.NotesListResult,
	mode SidebarMode,
) NotesPageView {
	view := NotesPageView{
		Locale:      locale,
		Messages:    messages,
		PageTitle:   notesPageTitle(messages, result),
		Filter:      result.ActiveFilter,
		SidebarMode: mode,
		Notes:       result.Notes,
		Authors:     uniqueSortedAuthors(result.Authors),
		Tags:        uniqueSortedTags(result.Tags),
		ActiveAuthor: func() *notes.Author {
			if result.ActiveAuthor == nil {
				return nil
			}
			copy := *result.ActiveAuthor
			return &copy
		}(),
		ActiveTag: func() *notes.Tag {
			if result.ActiveTag == nil {
				return nil
			}
			copy := *result.ActiveTag
			return &copy
		}(),
		Pagination: newPaginationView(locale, result.ActiveFilter, result.TotalPages),
	}

	applyContext(&view)
	return view
}

func notesPageTitle(messages map[webi18n.Key]string, result notes.NotesListResult) string {
	if result.ActiveAuthor != nil {
		return result.ActiveAuthor.Name
	}
	if result.ActiveTag != nil {
		return "#" + result.ActiveTag.Title
	}
	if result.ActiveFilter.Type == notes.NoteTypeLong {
		return Message(messages, webi18n.KeyLayoutTitleTales)
	}
	if result.ActiveFilter.Type == notes.NoteTypeShort {
		return Message(messages, webi18n.KeyLayoutTitleMicroTales)
	}

	return Message(messages, webi18n.KeyLayoutTitleNotes)
}

func applyContext(view *NotesPageView) {
	if view == nil {
		return
	}

	switch {
	case view.ActiveAuthor != nil:
		view.ContextTitle = view.ActiveAuthor.Name
		view.ContextSubtitle = "@" + view.ActiveAuthor.Slug
		view.ContextDescription = view.ActiveAuthor.Bio
	case view.ActiveTag != nil:
		view.ContextTitle = "#" + view.ActiveTag.Title
		view.ContextSubtitle = Message(view.Messages, webi18n.KeyContextTagSubtitle)
		view.ContextDescription = Message(view.Messages, webi18n.KeyContextTagDescription)
	case view.Filter.Type == notes.NoteTypeLong:
		view.ContextTitle = Message(view.Messages, webi18n.KeyLayoutTitleTales)
		view.ContextSubtitle = Message(view.Messages, webi18n.KeyContextTypeSubtitle)
		view.ContextDescription = Message(view.Messages, webi18n.KeyContextLongDescription)
	case view.Filter.Type == notes.NoteTypeShort:
		view.ContextTitle = Message(view.Messages, webi18n.KeyLayoutTitleMicroTales)
		view.ContextSubtitle = Message(view.Messages, webi18n.KeyContextTypeSubtitle)
		view.ContextDescription = Message(view.Messages, webi18n.KeyContextShortDescription)
	default:
		view.ContextTitle = Message(view.Messages, webi18n.KeyLayoutTitleNotes)
		view.ContextSubtitle = Message(view.Messages, webi18n.KeyContextFeed)
		view.ContextDescription = ""
	}
}

func newPaginationView(locale string, filter notes.ListFilter, totalPages int) PaginationView {
	if totalPages < 1 {
		totalPages = 1
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}

	hasPrev := page > 1
	hasNext := page < totalPages

	prevPage := page - 1
	if prevPage < 1 {
		prevPage = 1
	}

	nextPage := page + 1
	if nextPage < 1 {
		nextPage = 1
	}

	return PaginationView{
		Page:       page,
		TotalPages: totalPages,
		HasPrev:    hasPrev,
		HasNext:    hasNext,
		FirstPage:  1,
		LastPage:   totalPages,
		PrevPage:   prevPage,
		NextPage:   nextPage,
		FirstURL:   BuildNotesFilterURL(locale, 1, filter.AuthorSlug, filter.TagName, filter.Type, filter.Query),
		LastURL:    BuildNotesFilterURL(locale, totalPages, filter.AuthorSlug, filter.TagName, filter.Type, filter.Query),
		PrevURL:    BuildNotesFilterURL(locale, prevPage, filter.AuthorSlug, filter.TagName, filter.Type, filter.Query),
		NextURL:    BuildNotesFilterURL(locale, nextPage, filter.AuthorSlug, filter.TagName, filter.Type, filter.Query),
	}
}

func uniqueSortedAuthors(authors []notes.Author) []notes.Author {
	if len(authors) == 0 {
		return []notes.Author{}
	}

	authorBySlug := make(map[string]notes.Author, len(authors))
	for _, author := range authors {
		slug := strings.TrimSpace(author.Slug)
		if slug == "" {
			continue
		}

		existing, exists := authorBySlug[slug]
		if !exists {
			authorBySlug[slug] = author
			continue
		}

		if existing.Avatar == nil && author.Avatar != nil {
			existing.Avatar = author.Avatar
		}
		if strings.TrimSpace(existing.Bio) == "" && strings.TrimSpace(author.Bio) != "" {
			existing.Bio = author.Bio
		}
		if strings.TrimSpace(existing.Name) == "" && strings.TrimSpace(author.Name) != "" {
			existing.Name = author.Name
		}
		authorBySlug[slug] = existing
	}

	if len(authorBySlug) == 0 {
		return []notes.Author{}
	}

	out := make([]notes.Author, 0, len(authorBySlug))
	for _, author := range authorBySlug {
		out = append(out, author)
	}

	sort.Slice(out, func(i int, j int) bool {
		left := strings.ToLower(authorSortKey(out[i]))
		right := strings.ToLower(authorSortKey(out[j]))
		if left == right {
			return out[i].Slug < out[j].Slug
		}

		return left < right
	})

	return out
}

func authorSortKey(author notes.Author) string {
	name := strings.TrimSpace(author.Name)
	if name != "" {
		return name
	}

	return strings.TrimSpace(author.Slug)
}

func uniqueSortedTags(tags []notes.Tag) []notes.Tag {
	if len(tags) == 0 {
		return []notes.Tag{}
	}

	tagByName := make(map[string]notes.Tag, len(tags))
	for _, tag := range tags {
		name := strings.TrimSpace(tag.Name)
		if name == "" {
			continue
		}

		existing, exists := tagByName[name]
		if !exists {
			tagByName[name] = tag
			continue
		}

		if strings.TrimSpace(existing.Title) == "" && strings.TrimSpace(tag.Title) != "" {
			existing.Title = tag.Title
			tagByName[name] = existing
		}
	}

	if len(tagByName) == 0 {
		return []notes.Tag{}
	}

	out := make([]notes.Tag, 0, len(tagByName))
	for _, tag := range tagByName {
		out = append(out, tag)
	}

	sort.Slice(out, func(i int, j int) bool {
		left := strings.ToLower(tagSortKey(out[i]))
		right := strings.ToLower(tagSortKey(out[j]))
		if left == right {
			return out[i].Name < out[j].Name
		}

		return left < right
	})

	return out
}

func tagSortKey(tag notes.Tag) string {
	title := strings.TrimSpace(tag.Title)
	if title != "" {
		return title
	}

	return strings.TrimSpace(tag.Name)
}
