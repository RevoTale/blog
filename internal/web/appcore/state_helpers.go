package appcore

import (
	"strconv"
	"strings"

	"blog/internal/markdown"
	"blog/internal/notes"
)

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

func FirstAuthor(authors []notes.Author) *notes.Author {
	if len(authors) == 0 {
		return nil
	}

	author := authors[0]
	return &author
}

func FirstAuthorName(authors []notes.Author) string {
	author := FirstAuthor(authors)
	if author == nil {
		return ""
	}

	return strings.TrimSpace(author.Name)
}

func FirstAuthorSlug(authors []notes.Author) string {
	author := FirstAuthor(authors)
	if author == nil {
		return ""
	}

	return strings.TrimSpace(author.Slug)
}

func FirstAuthorAvatar(authors []notes.Author) *notes.AuthorMedia {
	author := FirstAuthor(authors)
	if author == nil {
		return nil
	}

	return author.Avatar
}

func HasFirstAuthorAvatar(authors []notes.Author) bool {
	return FirstAuthorAvatar(authors) != nil
}

func FirstAuthorAvatarURL(authors []notes.Author) string {
	avatar := FirstAuthorAvatar(authors)
	if avatar == nil {
		return ""
	}

	return strings.TrimSpace(avatar.URL)
}

func FirstAuthorAvatarAlt(authors []notes.Author) string {
	avatar := FirstAuthorAvatar(authors)
	if avatar == nil {
		return ""
	}

	return strings.TrimSpace(avatar.Alt)
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
		notes.ParseNoteType(string(view.SidebarCurrentType())) == notes.NoteTypeAll &&
		strings.TrimSpace(view.LayoutSearchQuery()) == ""
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

func HTMXNavigationScriptTag() string {
	return `<script>
(() => {
  document.addEventListener("htmx:afterSettle", (event) => {
    const detail = event && event.detail;
    const target = detail && detail.target;
    if (!(target instanceof HTMLElement)) {
      return;
    }
    if (target.id !== "notes-content") {
      return;
    }
    window.scrollTo({ top: 0, left: 0, behavior: "smooth" });
  });
})();
</script>`
}
