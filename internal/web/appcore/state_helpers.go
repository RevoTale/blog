package appcore

import (
	"encoding/json"
	"strconv"
	"strings"

	"blog/internal/markdown"
	"blog/internal/notes"
)

type NotesSignalState struct {
	Page   int    `json:"page"`
	Author string `json:"author"`
	Tag    string `json:"tag"`
	Type   string `json:"type"`
}

type AuthorSignalState struct {
	Page int    `json:"page"`
	Tag  string `json:"tag"`
	Type string `json:"type"`
}

func NotesSignalsJSON(view NotesPageView) string {
	return marshalSignals(NotesSignalState{
		Page:   view.Pagination.Page,
		Author: view.Filter.AuthorSlug,
		Tag:    view.Filter.TagName,
		Type:   string(view.Filter.Type),
	})
}

func AuthorSignalsJSON(view AuthorPageView) string {
	return marshalSignals(AuthorSignalState{
		Page: view.Pagination.Page,
		Tag:  view.Filter.TagName,
		Type: string(view.Filter.Type),
	})
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

func ChannelLinkClass(active bool) string {
	if active {
		return "channel-link active"
	}

	return "channel-link"
}

func AuthorChannelLabel(author notes.Author) string {
	label := strings.TrimSpace(author.Name)
	if label == "" {
		label = strings.TrimSpace(author.Slug)
	}

	return "@" + label
}

func TagChannelLabel(tag notes.Tag) string {
	label := strings.TrimSpace(tag.Title)
	if label == "" {
		label = strings.TrimSpace(tag.Name)
	}

	return "#" + label
}

func TypeChannelLabel(noteType notes.NoteType) string {
	switch notes.ParseNoteType(string(noteType)) {
	case notes.NoteTypeLong:
		return "Tales"
	case notes.NoteTypeShort:
		return "Micro-tales"
	default:
		return "ANY"
	}
}

func SidebarAllActive(view RootLayoutView) bool {
	if view == nil {
		return true
	}

	return strings.TrimSpace(view.SidebarCurrentAuthorSlug()) == "" &&
		strings.TrimSpace(view.SidebarCurrentTagName()) == "" &&
		notes.ParseNoteType(string(view.SidebarCurrentType())) == notes.NoteTypeAll
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
