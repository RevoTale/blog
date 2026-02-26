package web

import "blog/internal/notes"

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
	PageTitle  string
	Tag        string
	Notes      []notes.NoteSummary
	Tags       []notes.Tag
	Pagination PaginationView
}

type NotePageView struct {
	PageTitle string
	Note      notes.NoteDetail
}

type AuthorPageView struct {
	PageTitle  string
	Author     notes.Author
	Notes      []notes.NoteSummary
	Pagination PaginationView
}

func newNotesPageView(result notes.NotesListResult) NotesPageView {
	return NotesPageView{
		PageTitle: "Notes",
		Tag:       result.ActiveTag,
		Notes:     result.Notes,
		Tags:      result.Tags,
		Pagination: newPaginationView(
			result.Page,
			result.TotalPages,
			buildNotesURL(result.Page-1, result.ActiveTag),
			buildNotesURL(result.Page+1, result.ActiveTag),
		),
	}
}

func newAuthorPageView(result *notes.AuthorPageResult) AuthorPageView {
	return AuthorPageView{
		PageTitle: result.Author.Name,
		Author:    result.Author,
		Notes:     result.Notes,
		Pagination: newPaginationView(
			result.Page,
			result.TotalPages,
			buildAuthorURL(result.Author.Slug, result.Page-1),
			buildAuthorURL(result.Author.Slug, result.Page+1),
		),
	}
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
