package web

import (
	"encoding/json"
	"strconv"

	"blog/internal/markdown"
)

type notesSignalState struct {
	Page int    `json:"page"`
	Tag  string `json:"tag"`
}

type authorSignalState struct {
	Page int `json:"page"`
}

func notesSignalsJSON(view NotesPageView) string {
	return marshalSignals(notesSignalState{
		Page: view.Pagination.Page,
		Tag:  view.Tag,
	})
}

func authorSignalsJSON(view AuthorPageView) string {
	return marshalSignals(authorSignalState{Page: view.Pagination.Page})
}

func marshalSignals[T interface{}](value T) string {
	payload, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}

	return string(payload)
}

func tagClass(active bool) string {
	if active {
		return "tag active"
	}
	return "tag"
}

func noteCardClass(hasAttachment bool) string {
	if hasAttachment {
		return "panel note-card has-attachment"
	}
	return "panel note-card"
}

func pagerStatusText(p PaginationView) string {
	return "page " + strconv.Itoa(p.Page) + " / " + strconv.Itoa(p.TotalPages)
}

func notesTagAction(tag string) string {
	return "$tag=" + strconv.Quote(tag) + "; $page=1; @get('/notes/live')"
}

func notesPageAction(page int) string {
	return "$page=" + strconv.Itoa(sanitizePage(page)) + "; @get('/notes/live')"
}

func authorPageAction(slug string, page int) string {
	return "$page=" + strconv.Itoa(sanitizePage(page)) + "; @get('" + buildAuthorLiveURL(slug) + "')"
}

func attachmentAltText(alt string, fallbackTitle string) string {
	if alt != "" {
		return alt
	}
	if fallbackTitle != "" {
		return fallbackTitle + " attachment"
	}
	return "attachment"
}

func attachmentLabel(filename string) string {
	if filename != "" {
		return filename
	}
	return "open file"
}

func chromaStyleTag() string {
	return "<style>" + string(markdown.ChromaCSS()) + "</style>"
}
