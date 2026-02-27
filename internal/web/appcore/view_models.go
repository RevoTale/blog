package appcore

import (
	"sort"
	"strings"

	"blog/internal/notes"
)

type RootLayoutView interface {
	LayoutPageTitle() string
	SidebarAuthors() []notes.Author
	SidebarCurrentAuthorSlug() string
	SidebarTags() []notes.Tag
	SidebarCurrentTagName() string
}

type AuthorLayoutView interface {
	AuthorDetails() notes.Author
}

type PaginationView struct {
	Page       int
	TotalPages int
	HasPrev    bool
	HasNext    bool
	PrevPage   int
	NextPage   int
	PrevURL    string
	NextURL    string
}

type NotesPageView struct {
	PageTitle          string
	Tag                string
	Notes              []notes.NoteSummary
	Tags               []notes.Tag
	Pagination         PaginationView
	SidebarAuthorItems []notes.Author
	SidebarTagItems    []notes.Tag
}

type NotePageView struct {
	PageTitle          string
	Note               notes.NoteDetail
	SidebarAuthorItems []notes.Author
	SidebarTagItems    []notes.Tag
}

type AuthorPageView struct {
	PageTitle          string
	Author             notes.Author
	Notes              []notes.NoteSummary
	Pagination         PaginationView
	SidebarAuthorItems []notes.Author
	SidebarTagItems    []notes.Tag
}

func (v NotesPageView) LayoutPageTitle() string {
	return v.PageTitle
}

func (v NotesPageView) SidebarAuthors() []notes.Author {
	return v.SidebarAuthorItems
}

func (v NotesPageView) SidebarCurrentAuthorSlug() string {
	return ""
}

func (v NotesPageView) SidebarTags() []notes.Tag {
	return v.SidebarTagItems
}

func (v NotesPageView) SidebarCurrentTagName() string {
	return v.Tag
}

func (v NotePageView) LayoutPageTitle() string {
	return v.PageTitle
}

func (v NotePageView) SidebarAuthors() []notes.Author {
	return v.SidebarAuthorItems
}

func (v NotePageView) SidebarCurrentAuthorSlug() string {
	return ""
}

func (v NotePageView) SidebarTags() []notes.Tag {
	return v.SidebarTagItems
}

func (v NotePageView) SidebarCurrentTagName() string {
	return ""
}

func (v AuthorPageView) LayoutPageTitle() string {
	return v.PageTitle
}

func (v AuthorPageView) SidebarAuthors() []notes.Author {
	return v.SidebarAuthorItems
}

func (v AuthorPageView) SidebarCurrentAuthorSlug() string {
	return v.Author.Slug
}

func (v AuthorPageView) SidebarTags() []notes.Tag {
	return v.SidebarTagItems
}

func (v AuthorPageView) SidebarCurrentTagName() string {
	return ""
}

func (v AuthorPageView) AuthorDetails() notes.Author {
	return v.Author
}

func newNotesPageView(result notes.NotesListResult) NotesPageView {
	return NotesPageView{
		PageTitle:          "Notes",
		Tag:                result.ActiveTag,
		Notes:              result.Notes,
		Tags:               result.Tags,
		SidebarAuthorItems: collectAuthorsFromNotes(result.Notes),
		SidebarTagItems:    uniqueSortedTags(result.Tags),
		Pagination: newPaginationView(
			result.Page,
			result.TotalPages,
			BuildNotesURL(result.Page-1, result.ActiveTag),
			BuildNotesURL(result.Page+1, result.ActiveTag),
		),
	}
}

func newAuthorPageView(result *notes.AuthorPageResult) AuthorPageView {
	sidebarAuthors := collectAuthorsFromNotes(result.Notes)
	sidebarAuthors = uniqueSortedAuthors(append(sidebarAuthors, result.Author))

	return AuthorPageView{
		PageTitle:          result.Author.Name,
		Author:             result.Author,
		Notes:              result.Notes,
		SidebarAuthorItems: sidebarAuthors,
		SidebarTagItems:    collectTagsFromNotes(result.Notes),
		Pagination: newPaginationView(
			result.Page,
			result.TotalPages,
			BuildAuthorURL(result.Author.Slug, result.Page-1),
			BuildAuthorURL(result.Author.Slug, result.Page+1),
		),
	}
}

func collectAuthorsFromNotes(noteItems []notes.NoteSummary) []notes.Author {
	authors := make([]notes.Author, 0)
	for _, note := range noteItems {
		authors = append(authors, note.Authors...)
	}

	return uniqueSortedAuthors(authors)
}

func collectTagsFromNotes(noteItems []notes.NoteSummary) []notes.Tag {
	tags := make([]notes.Tag, 0)
	for _, note := range noteItems {
		tags = append(tags, note.Tags...)
	}

	return uniqueSortedTags(tags)
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

func newPaginationView(page int, totalPages int, prevURL string, nextURL string) PaginationView {
	if totalPages < 1 {
		totalPages = 1
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
		PrevPage:   prevPage,
		NextPage:   nextPage,
		PrevURL:    prevURL,
		NextURL:    nextURL,
	}
}

func sanitizePage(page int) int {
	if page < 1 {
		return 1
	}

	return page
}
