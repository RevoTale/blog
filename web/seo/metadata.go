package seo

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"blog/internal/notes"
	i18nkeys "blog/web/generated/i18nkeys"
	"blog/web/view"
	"github.com/RevoTale/no-js/framework"
	frameworki18n "github.com/RevoTale/no-js/framework/i18n"
	"github.com/RevoTale/no-js/framework/metagen"
)

func MetaGenRootPage(
	meta framework.MetaContext[*runtime.Context],
) (metagen.Metadata, error) {
	view, err := runtime.LoadNotesPage(meta.Context(), meta.App(), meta.Request(), framework.EmptyParams{})
	if err != nil {
		return metagen.Metadata{}, err
	}
	cardTitle := i18nkeys.TSeoRootTitle(meta.App().I18n(meta.Request()))
	description := i18nkeys.TSeoRootDescription(meta.App().I18n(meta.Request()))
	return notesListingMetadata(
		meta,
		view,
		cardTitle,
		description,
		"website",
		&metagen.Robots{Index: metagen.Bool(true), Follow: metagen.Bool(true)},
		true,
	)
}

func MetaGenTalesPage(
	meta framework.MetaContext[*runtime.Context],
) (metagen.Metadata, error) {
	view, err := runtime.LoadNotesTalesPage(meta.Context(), meta.App(), meta.Request(), framework.EmptyParams{})
	if err != nil {
		return metagen.Metadata{}, err
	}
	description := i18nkeys.TSeoTalesDescription(meta.App().I18n(meta.Request()))
	return notesListingMetadata(
		meta,
		view,
		view.PageTitle,
		description,
		"website",
		&metagen.Robots{Index: metagen.Bool(true), Follow: metagen.Bool(true)},
		false,
	)
}

func MetaGenMicroTalesPage(
	meta framework.MetaContext[*runtime.Context],
) (metagen.Metadata, error) {
	view, err := runtime.LoadNotesMicroTalesPage(meta.Context(), meta.App(), meta.Request(), framework.EmptyParams{})
	if err != nil {
		return metagen.Metadata{}, err
	}
	description := i18nkeys.TSeoMicroTalesDescription(meta.App().I18n(meta.Request()))
	return notesListingMetadata(
		meta,
		view,
		view.PageTitle,
		description,
		"website",
		&metagen.Robots{Index: metagen.Bool(true), Follow: metagen.Bool(true)},
		false,
	)
}

func MetaGenTagPage(
	meta framework.MetaContext[*runtime.Context],
	slug string,
) (metagen.Metadata, error) {
	view, err := runtime.LoadTagPage(meta.Context(), meta.App(), meta.Request(), framework.SlugParams{Slug: slug})
	if err != nil {
		return metagen.Metadata{}, err
	}
	description := i18nkeys.TSeoTagDescription(meta.App().I18n(meta.Request()), i18nkeys.SeoTagDescriptionArgs{
		Tag: strings.TrimSpace(strings.TrimPrefix(view.PageTitle, "#")),
	})
	return notesListingMetadata(
		meta,
		view,
		view.PageTitle,
		description,
		"website",
		&metagen.Robots{Index: metagen.Bool(true), Follow: metagen.Bool(true)},
		false,
	)
}

func MetaGenChannelsPage(
	meta framework.MetaContext[*runtime.Context],
) (metagen.Metadata, error) {
	view, err := runtime.LoadChannelsPage(meta.Context(), meta.App(), meta.Request(), framework.EmptyParams{})
	if err != nil {
		return metagen.Metadata{}, err
	}
	description := i18nkeys.TSeoChannelsDescription(meta.App().I18n(meta.Request()))
	return notesListingMetadata(
		meta,
		view,
		view.PageTitle,
		description,
		"website",
		&metagen.Robots{Index: metagen.Bool(false), Follow: metagen.Bool(true)},
		false,
	)
}

func MetaGenAuthorPage(
	meta framework.MetaContext[*runtime.Context],
	slug string,
) (metagen.Metadata, error) {
	view, err := runtime.LoadAuthorPage(meta.Context(), meta.App(), meta.Request(), framework.SlugParams{Slug: slug})
	if err != nil {
		return metagen.Metadata{}, err
	}

	site := siteInfo(meta.App().I18n(meta.Request()))

	authorName := ""
	authorSlug := ""
	var image *metagen.OpenGraphImage
	if view.ActiveAuthor != nil {
		authorName = strings.TrimSpace(view.ActiveAuthor.Name)
		authorSlug = strings.TrimSpace(view.ActiveAuthor.Slug)
		image = authorAvatarImage(view.RootURL, view.ActiveAuthor)
	}

	contentTitle := strings.TrimSpace(authorName)
	if contentTitle == "" {
		contentTitle = strings.TrimSpace(view.PageTitle)
	}
	if contentTitle != "" {
		contentTitle = contentTitle + " | Author"
	} else {
		contentTitle = "Author"
	}
	title := titleWithSite(contentTitle, site.Name)

	description := i18nkeys.TSeoAuthorDescription(meta.App().I18n(meta.Request()), i18nkeys.SeoAuthorDescriptionArgs{
		Author: strings.TrimSpace(view.PageTitle),
	})
	if view.ActiveAuthor != nil && strings.TrimSpace(view.ActiveAuthor.Bio) != "" {
		description = strings.TrimSpace(view.ActiveAuthor.Bio)
	}

	alternates, alternatesErr := buildAlternates(meta, view.LocaleCode(), nil)
	if alternatesErr != nil {
		return metagen.Metadata{}, alternatesErr
	}
	canonicalURL := strings.TrimSpace(alternates.Canonical)

	openGraph := &metagen.OpenGraph{
		Type:        "profile",
		URL:         canonicalURL,
		SiteName:    site.Name,
		Title:       contentTitle,
		Description: description,
		Locale:      view.LocaleCode(),
	}
	twitter := &metagen.Twitter{
		Card:        "summary",
		Site:        "@RevoTale",
		Title:       contentTitle,
		Description: description,
	}
	if image != nil {
		openGraph.Images = []metagen.OpenGraphImage{*image}
		twitter.Card = "summary_large_image"
		twitter.Images = []string{image.URL}
	}

	authors := []metagen.Author{}
	if authorName != "" {
		authors = append(authors, metagen.Author{
			Name: authorName,
			URL:  urlString(meta.LocalizedURL(view.LocaleCode(), "/author/"+authorSlug)),
		})
	}

	return metagen.Normalize(metagen.Metadata{
		Title:       title,
		Description: description,
		Alternates:  alternates,
		Robots: notesListingRobots(
			meta.Request(),
			view.Filter,
			&metagen.Robots{Index: metagen.Bool(true), Follow: metagen.Bool(true)},
		),
		OpenGraph: openGraph,
		Twitter:   twitter,
		Authors:   authors,
		Publisher: site.Publisher,
		Pinterest: &metagen.Pinterest{RichPin: metagen.Bool(true)},
	}), nil
}

func MetaGenNotePage(
	meta framework.MetaContext[*runtime.Context],
	slug string,
) (metagen.Metadata, error) {
	view, err := runtime.LoadNotePage(meta.Context(), meta.App(), meta.Request(), framework.SlugParams{Slug: slug})
	if err != nil {
		return metagen.Metadata{}, err
	}

	site := siteInfo(meta.App().I18n(meta.Request()))
	contentTitle := strings.TrimSpace(view.Note.MetaTitle)
	if contentTitle == "" {
		contentTitle = strings.TrimSpace(view.Note.Title)
	}
	title := titleWithSite(contentTitle, site.Name)
	description := strings.TrimSpace(view.Note.Description)

	alternates, alternatesErr := buildAlternates(meta, view.LocaleCode(), nil)
	if alternatesErr != nil {
		return metagen.Metadata{}, alternatesErr
	}
	canonicalURL := strings.TrimSpace(alternates.Canonical)

	image := noteImage(view.RootURL, view.Note.MetaImage, view.Note.Attachment)
	openGraph := &metagen.OpenGraph{
		Type:        "article",
		URL:         canonicalURL,
		SiteName:    site.Name,
		Title:       contentTitle,
		Description: description,
		Locale:      view.LocaleCode(),
	}
	twitter := &metagen.Twitter{
		Card:        "summary",
		Site:        "@RevoTale",
		Title:       contentTitle,
		Description: description,
	}
	if image != nil {
		openGraph.Images = []metagen.OpenGraphImage{*image}
		twitter.Card = "summary_large_image"
		twitter.Images = []string{image.URL}
	}

	authors := make([]metagen.Author, 0, len(view.Note.Authors))
	openGraphAuthors := make([]string, 0, len(view.Note.Authors))
	for _, author := range view.Note.Authors {
		authorName := strings.TrimSpace(author.Name)
		authorSlug := strings.TrimSpace(author.Slug)
		if authorName == "" {
			continue
		}
		authorURL := urlString(meta.LocalizedURL(view.LocaleCode(), "/author/"+authorSlug))
		authors = append(authors, metagen.Author{
			Name: authorName,
			URL:  authorURL,
		})
		openGraphAuthors = append(openGraphAuthors, authorURL)
	}

	openGraphTags := make([]string, 0, len(view.Note.Tags))
	for _, tag := range view.Note.Tags {
		tagName := strings.TrimSpace(tag.Title)
		if tagName == "" {
			tagName = strings.TrimSpace(tag.Name)
		}
		if tagName == "" {
			continue
		}
		openGraphTags = append(openGraphTags, tagName)
	}

	openGraph.PublishedTime = strings.TrimSpace(view.Note.PublishedAtISO)
	openGraph.Authors = openGraphAuthors
	openGraph.Tags = openGraphTags

	return metagen.Normalize(metagen.Metadata{
		Title:       title,
		Description: description,
		Alternates:  alternates,
		Robots: robotsWithQueryNoIndex(meta.Request(), &metagen.Robots{
			Index:  metagen.Bool(true),
			Follow: metagen.Bool(true),
		}),
		OpenGraph: openGraph,
		Twitter:   twitter,
		Authors:   authors,
		Publisher: site.Publisher,
		Pinterest: &metagen.Pinterest{RichPin: metagen.Bool(true)},
	}), nil
}

func notesListingMetadata(
	meta framework.MetaContext[*runtime.Context],
	view runtime.NotesPageView,
	cardTitle string,
	description string,
	openGraphType string,
	robots *metagen.Robots,
	includeRSS bool,
) (metagen.Metadata, error) {
	site := siteInfo(meta.App().I18n(meta.Request()))
	contentTitle := strings.TrimSpace(cardTitle)
	if contentTitle == "" {
		contentTitle = strings.TrimSpace(view.PageTitle)
	}
	title := titleWithSite(contentTitle, site.Name)

	alternateTypes := map[string]string(nil)
	if includeRSS {
		alternateTypes = notesRSSAlternateTypes(meta, view.LocaleCode())
	}

	alternates, err := buildAlternates(meta, view.LocaleCode(), alternateTypes)
	if err != nil {
		return metagen.Metadata{}, err
	}
	canonicalURL := strings.TrimSpace(alternates.Canonical)

	image := firstListingImage(view.RootURL, view.Notes)
	openGraph := &metagen.OpenGraph{
		Type:        strings.TrimSpace(openGraphType),
		URL:         canonicalURL,
		SiteName:    site.Name,
		Title:       contentTitle,
		Description: description,
		Locale:      view.LocaleCode(),
	}
	twitter := &metagen.Twitter{
		Card:        "summary",
		Site:        "@RevoTale",
		Title:       contentTitle,
		Description: description,
	}
	if image != nil {
		openGraph.Images = []metagen.OpenGraphImage{*image}
		twitter.Card = "summary_large_image"
		twitter.Images = []string{image.URL}
	}

	return metagen.Normalize(metagen.Metadata{
		Title:       title,
		Description: description,
		Alternates:  alternates,
		Robots:      notesListingRobots(meta.Request(), view.Filter, robots),
		OpenGraph:   openGraph,
		Twitter:     twitter,
		Publisher:   site.Publisher,
	}), nil
}

func notesListingRobots(r *http.Request, filter notes.ListFilter, base *metagen.Robots) *metagen.Robots {
	robots := base
	if robots == nil {
		robots = &metagen.Robots{}
	}

	if shouldNoIndexListingRequest(r, filter) {
		robots.Index = metagen.Bool(false)
		if robots.Follow == nil {
			robots.Follow = metagen.Bool(true)
		}
	}

	return robots
}

func shouldNoIndexListingRequest(r *http.Request, filter notes.ListFilter) bool {
	if strings.TrimSpace(filter.Query) != "" || activeListingFilterCount(filter) > 1 {
		return true
	}

	return requestHasUnknownListingQuery(r)
}

func activeListingFilterCount(filter notes.ListFilter) int {
	count := 0
	if strings.TrimSpace(filter.AuthorSlug) != "" {
		count++
	}
	if strings.TrimSpace(filter.TagName) != "" {
		count++
	}
	if notes.ParseNoteType(string(filter.Type)) != notes.NoteTypeAll {
		count++
	}

	return count
}

func requestHasUnknownListingQuery(r *http.Request) bool {
	if r == nil || r.URL == nil {
		return false
	}

	for key := range r.URL.Query() {
		switch strings.TrimSpace(key) {
		case "", "page", "author", "tag", "type", "q":
			continue
		default:
			return true
		}
	}

	return false
}

func robotsWithQueryNoIndex(r *http.Request, base *metagen.Robots) *metagen.Robots {
	robots := base
	if robots == nil {
		robots = &metagen.Robots{}
	}
	if requestHasQuery(r) {
		robots.Index = metagen.Bool(false)
		if robots.Follow == nil {
			robots.Follow = metagen.Bool(true)
		}
	}
	return robots
}

func requestHasQuery(r *http.Request) bool {
	if r == nil || r.URL == nil {
		return false
	}
	return strings.TrimSpace(r.URL.RawQuery) != ""
}

type siteMetadata struct {
	Name        string
	Description string
	Publisher   string
}

func siteInfo(i18n frameworki18n.Context[i18nkeys.Key]) siteMetadata {
	name := strings.TrimSpace(i18nkeys.TSeoSiteName(i18n))
	description := strings.TrimSpace(i18nkeys.TSeoSiteDescription(i18n))
	publisher := strings.TrimSpace(i18nkeys.TSeoPublisherName(i18n))

	return siteMetadata{
		Name:        name,
		Description: description,
		Publisher:   publisher,
	}
}

func titleWithSite(pageTitle string, siteName string) string {
	trimmedPage := strings.TrimSpace(pageTitle)
	trimmedSite := strings.TrimSpace(siteName)
	if trimmedSite == "" {
		trimmedSite = "RevoTale"
	}
	if trimmedPage == "" {
		return trimmedSite
	}
	return trimmedPage + " | " + trimmedSite
}

func buildAlternates(
	meta framework.MetaContext[*runtime.Context],
	locale string,
	alternateTypes map[string]string,
) (metagen.Alternates, error) {
	if meta == nil {
		return metagen.Alternates{}, fmt.Errorf("metadata context is required")
	}
	return meta.Alternates(locale, alternateTypes)
}

func notesRSSAlternateTypes(meta framework.MetaContext[*runtime.Context], locale string) map[string]string {
	if meta == nil {
		return nil
	}

	feedURL := meta.URL("/feed.xml")
	if feedURL == nil {
		return nil
	}

	query := url.Values{}
	query.Set("locale", strings.TrimSpace(locale))
	if request := meta.Request(); request != nil && request.URL != nil {
		requestQuery := request.URL.Query()
		for _, key := range []string{"page", "author", "tag", "type", "q"} {
			value := strings.TrimSpace(requestQuery.Get(key))
			if value == "" {
				continue
			}
			query.Set(key, value)
		}
	}
	feedURL.RawQuery = query.Encode()

	return map[string]string{
		"application/rss+xml": feedURL.String(),
	}
}

func firstListingImage(rootURL string, notes []notes.NoteSummary) *metagen.OpenGraphImage {
	for _, note := range notes {
		if image := noteImage(rootURL, note.MetaImage, note.Attachment); image != nil {
			return image
		}
	}
	return nil
}

func noteImage(
	rootURL string,
	metaImage *notes.Attachment,
	attachment *notes.Attachment,
) *metagen.OpenGraphImage {
	if image := noteAttachmentImage(rootURL, metaImage); image != nil {
		return image
	}
	return noteAttachmentImage(rootURL, attachment)
}

func noteAttachmentImage(rootURL string, attachment *notes.Attachment) *metagen.OpenGraphImage {
	if attachment == nil {
		return nil
	}
	thumbURL, thumbWidth, thumbHeight := runtime.ImageThumb(
		strings.TrimSpace(attachment.URL),
		attachment.Width,
		attachment.Height,
	)
	imageURL := absoluteMediaURL(rootURL, thumbURL)
	if imageURL == "" {
		return nil
	}
	return &metagen.OpenGraphImage{
		URL:    imageURL,
		Alt:    strings.TrimSpace(attachment.Alt),
		Width:  thumbWidth,
		Height: thumbHeight,
	}
}

func authorAvatarImage(rootURL string, author *notes.Author) *metagen.OpenGraphImage {
	if author == nil || author.Avatar == nil {
		return nil
	}
	thumbURL, thumbWidth, thumbHeight := runtime.ImageThumb(
		strings.TrimSpace(author.Avatar.URL),
		author.Avatar.Width,
		author.Avatar.Height,
	)
	imageURL := absoluteMediaURL(rootURL, thumbURL)
	if imageURL == "" {
		return nil
	}
	return &metagen.OpenGraphImage{
		URL:    imageURL,
		Alt:    strings.TrimSpace(author.Avatar.Alt),
		Width:  thumbWidth,
		Height: thumbHeight,
	}
}

func absoluteMediaURL(rootURL string, rawURL string) string {
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

func joinRootAndPath(rootURL string, routePath string) string {
	trimmedPath := strings.TrimSpace(routePath)
	if trimmedPath == "" {
		trimmedPath = "/"
	}
	if !strings.HasPrefix(trimmedPath, "/") {
		trimmedPath = "/" + trimmedPath
	}

	parsedRoot, err := url.Parse(strings.TrimSpace(rootURL))
	if err != nil || !parsedRoot.IsAbs() || strings.TrimSpace(parsedRoot.Host) == "" {
		return trimmedPath
	}

	base := strings.TrimSuffix(strings.TrimSpace(parsedRoot.Path), "/")
	if trimmedPath == "/" {
		if base == "" {
			parsedRoot.Path = "/"
		} else {
			parsedRoot.Path = base
		}
		parsedRoot.RawQuery = ""
		parsedRoot.Fragment = ""
		return parsedRoot.String()
	}

	joined := path.Join(base, strings.TrimPrefix(trimmedPath, "/"))
	if !strings.HasPrefix(joined, "/") {
		joined = "/" + joined
	}
	parsedRoot.Path = joined
	parsedRoot.RawQuery = ""
	parsedRoot.Fragment = ""
	return parsedRoot.String()
}

func urlString(value *url.URL) string {
	if value == nil {
		return ""
	}
	return value.String()
}
