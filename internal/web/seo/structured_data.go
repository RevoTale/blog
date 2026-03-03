package seo

import (
	"context"
	"encoding/json"
	"io"
	"net/url"
	"strings"

	"blog/internal/notes"
	"blog/internal/web/appcore"
	webi18n "blog/internal/web/i18n"
	"github.com/a-h/templ"
)

type jsonLDComponentFunc func(ctx context.Context, w io.Writer) error

func (f jsonLDComponentFunc) Render(ctx context.Context, w io.Writer) error {
	return f(ctx, w)
}

func JSONLDScript(doc any) templ.Component {
	return jsonLDComponentFunc(func(_ context.Context, w io.Writer) error {
		if doc == nil {
			return nil
		}

		encoded, err := json.Marshal(doc)
		if err != nil {
			return err
		}

		_, err = io.WriteString(w, `<script type="application/ld+json">`+string(encoded)+`</script>`)
		return err
	})
}

func BuildOrganizationJSONLD(rootURL string) map[string]any {
	canonicalRoot := joinRootAndPath(rootURL, "/")
	return map[string]any{
		"@context": "https://schema.org",
		"@type":    "Organization",
		"brand":    "RevoTale",
		"name":     "RevoTale",
		"logo":     joinRootAndPath(rootURL, "/apple-touch-icon.png"),
		"url":      canonicalRoot,
		"sameAs": []string{
			"https://twitter.com/RevoTale",
			"https://github.com/RevoTale",
			"https://www.npmjs.com/~grisaia",
			"https://packagist.org/users/grisaia/",
		},
	}
}

func BuildAuthorJSONLD(view appcore.AuthorPageView) map[string]any {
	if view.ActiveAuthor == nil {
		return nil
	}

	author := view.ActiveAuthor
	doc := map[string]any{
		"@context": "https://schema.org",
		"@type":    "Person",
		"name":     strings.TrimSpace(author.Name),
		"url":      authorCanonicalURL(view, author.Slug),
	}
	if image := authorAvatarImageObject(view.RootURL, author.Avatar); image != nil {
		doc["image"] = image
	}
	return doc
}

func BuildNoteJSONLD(view appcore.NotePageView) map[string]any {
	canonicalURL := strings.TrimSpace(view.CanonicalURL)
	if canonicalURL == "" {
		canonicalURL = absoluteLocalizedURLForRoot(
			view.RootURL,
			view.LocaleCode(),
			"/note/"+strings.TrimSpace(view.Note.Slug),
		)
	}

	authors := make([]map[string]any, 0, len(view.Note.Authors))
	for _, author := range view.Note.Authors {
		name := strings.TrimSpace(author.Name)
		if name == "" {
			continue
		}

		person := map[string]any{
			"@context": "https://schema.org",
			"@type":    "Person",
			"name":     name,
			"url":      absoluteLocalizedURLForRoot(view.RootURL, view.LocaleCode(), "/author/"+strings.TrimSpace(author.Slug)),
		}
		if image := authorAvatarImageObject(view.RootURL, author.Avatar); image != nil {
			person["image"] = image
		}
		authors = append(authors, person)
	}

	doc := map[string]any{
		"@context":    "https://schema.org",
		"@type":       "BlogPosting",
		"@id":         canonicalURL,
		"headline":    pickNoteHeadline(view.Note.Title, view.Note.MetaTitle),
		"url":         canonicalURL,
		"author":      authors,
		"publisher":   BuildOrganizationJSONLD(view.RootURL),
		"description": strings.TrimSpace(view.Note.Description),
		"mainEntityOfPage": map[string]any{
			"@type": "WebPage",
			"@id":   canonicalURL,
		},
		"inLanguage": view.LocaleCode(),
	}
	if datePublished := strings.TrimSpace(view.Note.PublishedAtISO); datePublished != "" {
		doc["datePublished"] = datePublished
	}
	if image := pickNoteImageObject(view.RootURL, view.Note.MetaImage, view.Note.Attachment); image != nil {
		doc["image"] = image
	}
	if mentions := structuredDataMentions(view.RootURL, view.LocaleCode(), view.Note.Mentions); len(mentions) > 0 {
		doc["mentions"] = mentions
	}

	return doc
}

func BuildNotesBlogJSONLD(view appcore.NotesPageView) map[string]any {
	canonicalURL := strings.TrimSpace(view.CanonicalURL)
	if canonicalURL == "" {
		canonicalURL = absoluteLocalizedURLForRoot(view.RootURL, view.LocaleCode(), "/")
	}

	blogPosts := make([]map[string]any, 0, len(view.Notes))
	for _, note := range view.Notes {
		noteURL := absoluteLocalizedURLForRoot(view.RootURL, view.LocaleCode(), "/note/"+strings.TrimSpace(note.Slug))
		authors := make([]map[string]any, 0, len(note.Authors))
		for _, author := range note.Authors {
			name := strings.TrimSpace(author.Name)
			if name == "" {
				continue
			}
			authors = append(authors, map[string]any{
				"@context": "https://schema.org",
				"@type":    "Person",
				"name":     name,
				"url":      absoluteLocalizedURLForRoot(view.RootURL, view.LocaleCode(), "/author/"+strings.TrimSpace(author.Slug)),
			})
		}

		post := map[string]any{
			"@context":    "https://schema.org",
			"@type":       "BlogPosting",
			"@id":         noteURL,
			"headline":    pickNoteHeadline(note.Title, note.MetaTitle),
			"url":         noteURL,
			"author":      authors,
			"publisher":   BuildOrganizationJSONLD(view.RootURL),
			"description": strings.TrimSpace(note.Description),
			"mainEntityOfPage": map[string]any{
				"@type": "WebPage",
				"@id":   noteURL,
			},
			"inLanguage": view.LocaleCode(),
		}
		if datePublished := strings.TrimSpace(note.PublishedAtISO); datePublished != "" {
			post["datePublished"] = datePublished
		}
		if image := pickNoteImageObject(view.RootURL, note.MetaImage, note.Attachment); image != nil {
			post["image"] = image
		}
		if mentions := structuredDataMentions(view.RootURL, view.LocaleCode(), note.Mentions); len(mentions) > 0 {
			post["mentions"] = mentions
		}
		blogPosts = append(blogPosts, post)
	}

	name := strings.TrimSpace(appcore.Message(view.MessagesMap(), webi18n.KeySeoNotesJSONLDName))
	if name == "" || name == string(webi18n.KeySeoNotesJSONLDName) {
		name = "Notes"
	}
	description := strings.TrimSpace(appcore.Message(view.MessagesMap(), webi18n.KeySeoNotesJSONLDDescription))
	if description == "" || description == string(webi18n.KeySeoNotesJSONLDDescription) {
		description = "Explore a collection of notes on coding, web performance, SEO, AI workflows, and book takeaways."
	}

	return map[string]any{
		"@context":    "https://schema.org",
		"@type":       "Blog",
		"name":        name,
		"url":         canonicalURL,
		"description": description,
		"inLanguage":  view.LocaleCode(),
		"publisher":   BuildOrganizationJSONLD(view.RootURL),
		"blogPost":    blogPosts,
	}
}

func pickNoteHeadline(title string, metaTitle string) string {
	out := strings.TrimSpace(title)
	if out != "" {
		return out
	}
	return strings.TrimSpace(metaTitle)
}

func pickNoteImageObject(rootURL string, metaImage *notes.Attachment, attachment *notes.Attachment) map[string]any {
	if image := attachmentToImageObject(rootURL, metaImage); image != nil {
		return image
	}
	return attachmentToImageObject(rootURL, attachment)
}

func authorAvatarImageObject(rootURL string, avatar *notes.AuthorMedia) map[string]any {
	if avatar == nil {
		return nil
	}
	thumbURL, thumbWidth, thumbHeight := appcore.ImageThumb(
		strings.TrimSpace(avatar.URL),
		avatar.Width,
		avatar.Height,
	)
	imageURL := absoluteMediaURLForRoot(rootURL, thumbURL)
	if imageURL == "" {
		return nil
	}
	image := map[string]any{
		"@context": "https://schema.org",
		"@type":    "ImageObject",
		"url":      imageURL,
	}
	if thumbWidth > 0 {
		image["width"] = thumbWidth
	}
	if thumbHeight > 0 {
		image["height"] = thumbHeight
	}
	if alt := strings.TrimSpace(avatar.Alt); alt != "" {
		image["description"] = alt
	}
	return image
}

func attachmentToImageObject(rootURL string, attachment *notes.Attachment) map[string]any {
	if attachment == nil {
		return nil
	}
	thumbURL, thumbWidth, thumbHeight := appcore.ImageThumb(
		strings.TrimSpace(attachment.URL),
		attachment.Width,
		attachment.Height,
	)
	imageURL := absoluteMediaURLForRoot(rootURL, thumbURL)
	if imageURL == "" {
		return nil
	}

	image := map[string]any{
		"@context": "https://schema.org",
		"@type":    "ImageObject",
		"url":      imageURL,
	}
	if thumbWidth > 0 {
		image["width"] = thumbWidth
	}
	if thumbHeight > 0 {
		image["height"] = thumbHeight
	}
	if desc := strings.TrimSpace(attachment.Alt); desc != "" {
		image["description"] = desc
	}
	return image
}

func structuredDataMentions(rootURL string, locale string, mentions []notes.NoteMention) []map[string]any {
	if len(mentions) == 0 {
		return nil
	}

	out := make([]map[string]any, 0, len(mentions))
	for _, mention := range mentions {
		target := absoluteMentionURL(rootURL, locale, mention.URL)
		if target == "" {
			continue
		}
		out = append(out, map[string]any{"@id": target})
	}
	return out
}

func absoluteMentionURL(rootURL string, locale string, rawURL string) string {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return ""
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return ""
	}
	if parsed.IsAbs() && strings.TrimSpace(parsed.Host) != "" {
		return parsed.String()
	}

	localizedPath := parsed.Path
	if strings.TrimSpace(localizedPath) != "" {
		localizedPath = appcore.LocalizeAppPath(locale, localizedPath)
	}
	return joinRootAndPath(rootURL, localizedPath)
}

func absoluteMediaURLForRoot(rootURL string, rawURL string) string {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return ""
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return ""
	}
	if parsed.IsAbs() && strings.TrimSpace(parsed.Host) != "" {
		return parsed.String()
	}
	return joinRootAndPath(rootURL, parsed.Path)
}

func absoluteLocalizedURLForRoot(rootURL string, locale string, strippedPath string) string {
	localizedPath := appcore.LocalizeAppPath(locale, strippedPath)
	return joinRootAndPath(rootURL, localizedPath)
}

func authorCanonicalURL(view appcore.AuthorPageView, authorSlug string) string {
	if canonical := strings.TrimSpace(view.CanonicalURL); canonical != "" {
		return canonical
	}
	return absoluteLocalizedURLForRoot(view.RootURL, view.LocaleCode(), "/author/"+strings.TrimSpace(authorSlug))
}
