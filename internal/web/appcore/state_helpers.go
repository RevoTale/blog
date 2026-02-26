package appcore

import (
	"encoding/json"
	"strconv"

	"blog/internal/markdown"
)

type NotesSignalState struct {
	Page int    `json:"page"`
	Tag  string `json:"tag"`
}

type AuthorSignalState struct {
	Page int `json:"page"`
}

func NotesSignalsJSON(view NotesPageView) string {
	return marshalSignals(NotesSignalState{
		Page: view.Pagination.Page,
		Tag:  view.Tag,
	})
}

func AuthorSignalsJSON(view AuthorPageView) string {
	return marshalSignals(AuthorSignalState{Page: view.Pagination.Page})
}

func marshalSignals[T interface{}](value T) string {
	payload, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}

	return string(payload)
}

func TagClass(active bool) string {
	if active {
		return "tag active"
	}
	return "tag"
}

func NoteCardClass(hasAttachment bool) string {
	if hasAttachment {
		return "panel note-card has-attachment"
	}
	return "panel note-card"
}

func PagerStatusText(p PaginationView) string {
	return "page " + strconv.Itoa(p.Page) + " / " + strconv.Itoa(p.TotalPages)
}

func NotesTagAction(tag string) string {
	return "$tag=" + strconv.Quote(tag) + "; $page=1; @get('/notes/live')"
}

func NotesPageAction(page int) string {
	return "$page=" + strconv.Itoa(sanitizePage(page)) + "; @get('/notes/live')"
}

func AuthorPageAction(slug string, page int) string {
	return "$page=" + strconv.Itoa(sanitizePage(page)) + "; @get('" + BuildAuthorLiveURL(slug) + "')"
}

func AttachmentAltText(alt string, fallbackTitle string) string {
	if alt != "" {
		return alt
	}
	if fallbackTitle != "" {
		return fallbackTitle + " attachment"
	}
	return "attachment"
}

func AttachmentLabel(filename string) string {
	if filename != "" {
		return filename
	}
	return "open file"
}

func ChromaStyleTag() string {
	return "<style>" + string(markdown.ChromaCSS()) + "</style>"
}
