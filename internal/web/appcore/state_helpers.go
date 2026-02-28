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

func LiveNavigationScriptTag() string {
	return `<script>
(() => {
  const liveMarkerKey = "__live";
  const liveMarkerValue = "navigation";

  function hasLiveContainer() {
    return document.querySelector("#notes-content") !== null;
  }

  function toLiveURL(rawURL) {
    const url = new URL(rawURL, window.location.origin);
    const pathname = url.pathname === "/" ? "/.live/" : "/.live" + url.pathname;
    url.searchParams.set(liveMarkerKey, liveMarkerValue);
    const query = url.searchParams.toString();
    if (query === "") {
      return pathname;
    }
    return pathname + "?" + query;
  }

  function scrollToTop() {
    window.scrollTo({ top: 0, left: 0, behavior: "smooth" });
  }

  function patchElement(selector, mode, html) {
    const target = document.querySelector(selector);
    if (!target) {
      return false;
    }
    if (mode === "inner") {
      target.innerHTML = html;
      return true;
    }

    const tpl = document.createElement("template");
    tpl.innerHTML = html.trim();
    if (!tpl.content.firstChild) {
      return false;
    }
    target.replaceWith(tpl.content);
    return true;
  }

  function applyLivePatch(streamText) {
    const blocks = streamText.split(/\r?\n\r?\n+/);
    let patched = false;

    for (const block of blocks) {
      const lines = block.split(/\r?\n/);
      let eventName = "";
      let selector = "";
      let mode = "outer";
      const elementLines = [];

      for (const line of lines) {
        if (line.startsWith("event:")) {
          eventName = line.slice("event:".length).trim();
          continue;
        }
        if (!line.startsWith("data:")) {
          continue;
        }
        const dataLine = line.slice("data:".length).trimStart();
        if (dataLine.startsWith("selector ")) {
          selector = dataLine.slice("selector ".length).trim();
          continue;
        }
        if (dataLine.startsWith("mode ")) {
          mode = dataLine.slice("mode ".length).trim();
          continue;
        }
        if (dataLine.startsWith("elements ")) {
          elementLines.push(dataLine.slice("elements ".length));
        }
      }

      if (eventName !== "datastar-patch-elements") {
        continue;
      }
      if (selector === "" || elementLines.length === 0) {
        continue;
      }
      if (patchElement(selector, mode, elementLines.join("\n"))) {
        patched = true;
      }
    }

    return patched;
  }

  async function requestLivePatch(liveURL) {
    if (!hasLiveContainer()) {
      return;
    }

    try {
      const res = await fetch(liveURL, {
        method: "GET",
        credentials: "same-origin",
        headers: {
          Accept: "text/event-stream"
        }
      });
      if (!res.ok) {
        window.location.assign(window.location.href);
        return;
      }

      const payload = await res.text();
      if (!applyLivePatch(payload)) {
        window.location.assign(window.location.href);
        return;
      }
      scrollToTop();
    } catch (_err) {
      window.location.assign(window.location.href);
    }
  }

  document.addEventListener("click", (event) => {
    if (event.defaultPrevented) {
      return;
    }

    const target = event.target;
    if (!(target instanceof Element)) {
      return;
    }

    const link = target.closest("a[data-live-nav-url]");
    if (!link) {
      return;
    }
    if (!(link instanceof HTMLAnchorElement)) {
      return;
    }
    if (event.button !== 0) {
      return;
    }
    if (event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) {
      return;
    }

    const href = link.getAttribute("href");
    const liveURL = link.getAttribute("data-live-nav-url");
    if (!href || !liveURL) {
      return;
    }

    event.preventDefault();
    window.history.pushState({}, "", href);
    void requestLivePatch(liveURL);
  });

  window.addEventListener("popstate", () => {
    if (!hasLiveContainer()) {
      return;
    }
    void requestLivePatch(toLiveURL(window.location.pathname + window.location.search));
  });
})();
</script>`
}
