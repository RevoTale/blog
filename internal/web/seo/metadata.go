package seo

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"blog/internal/notes"
	"blog/internal/web/appcore"
	webi18n "blog/internal/web/i18n"
	"github.com/RevoTale/no-js/framework"
	frameworki18n "github.com/RevoTale/no-js/framework/i18n"
	"github.com/RevoTale/no-js/framework/metagen"
)

func MetaGenRootPage(
	ctx context.Context,
	appCtx *appcore.Context,
	r *http.Request,
) (metagen.Metadata, error) {
	view, err := appcore.LoadNotesPage(ctx, appCtx, r, framework.EmptyParams{})
	if err != nil {
		return metagen.Metadata{}, err
	}
	cardTitle := localizeSEO(
		appCtx,
		view.LocaleCode(),
		webi18n.KeySeoRootTitle,
		"Notes - Quick Coding, Experience, Open Source, SEO & Science Insights",
		nil,
	)
	description := localizeSEO(
		appCtx,
		view.LocaleCode(),
		webi18n.KeySeoRootDescription,
		"Dive into concise notes packed with actionable tips on coding, web-performance, SEO, "+
			"AI workflows, book takeaways and more-updated regularly on RevoTale.",
		nil,
	)
	return notesListingMetadata(
		appCtx,
		r,
		view,
		cardTitle,
		description,
		"website",
		&metagen.Robots{Index: metagen.Bool(true), Follow: metagen.Bool(true)},
		true,
	)
}

func MetaGenTalesPage(
	ctx context.Context,
	appCtx *appcore.Context,
	r *http.Request,
) (metagen.Metadata, error) {
	view, err := appcore.LoadNotesTalesPage(ctx, appCtx, r, framework.EmptyParams{})
	if err != nil {
		return metagen.Metadata{}, err
	}
	description := localizeSEO(
		appCtx,
		view.LocaleCode(),
		webi18n.KeySeoTalesDescription,
		"Read long-form tales from the blog feed.",
		nil,
	)
	return notesListingMetadata(
		appCtx,
		r,
		view,
		view.PageTitle,
		description,
		"website",
		&metagen.Robots{Index: metagen.Bool(true), Follow: metagen.Bool(true)},
		false,
	)
}

func MetaGenMicroTalesPage(
	ctx context.Context,
	appCtx *appcore.Context,
	r *http.Request,
) (metagen.Metadata, error) {
	view, err := appcore.LoadNotesMicroTalesPage(ctx, appCtx, r, framework.EmptyParams{})
	if err != nil {
		return metagen.Metadata{}, err
	}
	description := localizeSEO(
		appCtx,
		view.LocaleCode(),
		webi18n.KeySeoMicroTalesDescription,
		"Read short-form micro-tales from the blog feed.",
		nil,
	)
	return notesListingMetadata(
		appCtx,
		r,
		view,
		view.PageTitle,
		description,
		"website",
		&metagen.Robots{Index: metagen.Bool(true), Follow: metagen.Bool(true)},
		false,
	)
}

func MetaGenTagPage(
	ctx context.Context,
	appCtx *appcore.Context,
	r *http.Request,
	slug string,
) (metagen.Metadata, error) {
	view, err := appcore.LoadTagPage(ctx, appCtx, r, framework.SlugParams{Slug: slug})
	if err != nil {
		return metagen.Metadata{}, err
	}
	description := localizeSEO(
		appCtx,
		view.LocaleCode(),
		webi18n.KeySeoTagDescription,
		"Browse notes for this tag.",
		map[string]any{
			"Tag": strings.TrimSpace(strings.TrimPrefix(view.PageTitle, "#")),
		},
	)
	return notesListingMetadata(
		appCtx,
		r,
		view,
		view.PageTitle,
		description,
		"website",
		&metagen.Robots{Index: metagen.Bool(true), Follow: metagen.Bool(true)},
		false,
	)
}

func MetaGenChannelsPage(
	ctx context.Context,
	appCtx *appcore.Context,
	r *http.Request,
) (metagen.Metadata, error) {
	view, err := appcore.LoadChannelsPage(ctx, appCtx, r, framework.EmptyParams{})
	if err != nil {
		return metagen.Metadata{}, err
	}
	description := localizeSEO(
		appCtx,
		view.LocaleCode(),
		webi18n.KeySeoChannelsDescription,
		"Browse available channels and filters for the blog feed.",
		nil,
	)
	return notesListingMetadata(
		appCtx,
		r,
		view,
		view.PageTitle,
		description,
		"website",
		&metagen.Robots{Index: metagen.Bool(false), Follow: metagen.Bool(true)},
		false,
	)
}

func MetaGenAuthorPage(
	ctx context.Context,
	appCtx *appcore.Context,
	r *http.Request,
	slug string,
) (metagen.Metadata, error) {
	view, err := appcore.LoadAuthorPage(ctx, appCtx, r, framework.SlugParams{Slug: slug})
	if err != nil {
		return metagen.Metadata{}, err
	}

	site := siteInfo(appCtx, view.LocaleCode())

	authorName := ""
	authorSlug := ""
	var image *metagen.OpenGraphImage
	if view.ActiveAuthor != nil {
		authorName = strings.TrimSpace(view.ActiveAuthor.Name)
		authorSlug = strings.TrimSpace(view.ActiveAuthor.Slug)
		image = authorAvatarImage(appCtx, view.ActiveAuthor)
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

	description := localizeSEO(
		appCtx,
		view.LocaleCode(),
		webi18n.KeySeoAuthorDescription,
		"Browse notes by this author.",
		map[string]any{
			"Author": strings.TrimSpace(view.PageTitle),
		},
	)
	if view.ActiveAuthor != nil && strings.TrimSpace(view.ActiveAuthor.Bio) != "" {
		description = strings.TrimSpace(view.ActiveAuthor.Bio)
	}

	alternates, alternatesErr := buildAlternates(appCtx, r, view.LocaleCode(), nil)
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
			URL:  absoluteLocalizedURL(appCtx, view.LocaleCode(), "/author/"+authorSlug),
		})
	}

	return metagen.Normalize(metagen.Metadata{
		Title:       title,
		Description: description,
		Alternates:  alternates,
		Robots: notesListingRobots(
			r,
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
	ctx context.Context,
	appCtx *appcore.Context,
	r *http.Request,
	slug string,
) (metagen.Metadata, error) {
	view, err := appcore.LoadNotePage(ctx, appCtx, r, framework.SlugParams{Slug: slug})
	if err != nil {
		return metagen.Metadata{}, err
	}

	site := siteInfo(appCtx, view.LocaleCode())
	contentTitle := strings.TrimSpace(view.Note.MetaTitle)
	if contentTitle == "" {
		contentTitle = strings.TrimSpace(view.Note.Title)
	}
	title := titleWithSite(contentTitle, site.Name)
	description := strings.TrimSpace(view.Note.Description)

	alternates, alternatesErr := buildAlternates(appCtx, r, view.LocaleCode(), nil)
	if alternatesErr != nil {
		return metagen.Metadata{}, alternatesErr
	}
	canonicalURL := strings.TrimSpace(alternates.Canonical)

	image := noteImage(appCtx, view.Note.MetaImage, view.Note.Attachment)
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
		authorURL := absoluteLocalizedURL(appCtx, view.LocaleCode(), "/author/"+authorSlug)
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
		Robots:      robotsWithQueryNoIndex(r, &metagen.Robots{Index: metagen.Bool(true), Follow: metagen.Bool(true)}),
		OpenGraph:   openGraph,
		Twitter:     twitter,
		Authors:     authors,
		Publisher:   site.Publisher,
		Pinterest:   &metagen.Pinterest{RichPin: metagen.Bool(true)},
	}), nil
}

func notesListingMetadata(
	appCtx *appcore.Context,
	r *http.Request,
	view appcore.NotesPageView,
	cardTitle string,
	description string,
	openGraphType string,
	robots *metagen.Robots,
	includeRSS bool,
) (metagen.Metadata, error) {
	site := siteInfo(appCtx, view.LocaleCode())
	contentTitle := strings.TrimSpace(cardTitle)
	if contentTitle == "" {
		contentTitle = strings.TrimSpace(view.PageTitle)
	}
	title := titleWithSite(contentTitle, site.Name)

	alternateTypes := map[string]string(nil)
	if includeRSS {
		alternateTypes = notesRSSAlternateTypes(appCtx, r, view.LocaleCode())
	}

	alternates, err := buildAlternates(appCtx, r, view.LocaleCode(), alternateTypes)
	if err != nil {
		return metagen.Metadata{}, err
	}
	canonicalURL := strings.TrimSpace(alternates.Canonical)

	image := firstListingImage(appCtx, view.Notes)
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
		Robots:      notesListingRobots(r, view.Filter, robots),
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
	RootURL     string
}

func siteInfo(appCtx *appcore.Context, locale string) siteMetadata {
	name := localizeSEO(appCtx, locale, webi18n.KeySeoSiteName, "RevoTale", nil)
	description := localizeSEO(
		appCtx,
		locale,
		webi18n.KeySeoSiteDescription,
		"A multilingual note feed with tales and micro-tales.",
		nil,
	)
	publisher := localizeSEO(appCtx, locale, webi18n.KeySeoPublisherName, "RevoTale", nil)

	rootURL := ""
	if appCtx != nil {
		rootURL = strings.TrimSpace(appCtx.RootURL())
	}

	return siteMetadata{
		Name:        name,
		Description: description,
		Publisher:   publisher,
		RootURL:     rootURL,
	}
}

func localizeSEO(
	appCtx *appcore.Context,
	locale string,
	key webi18n.Key,
	fallback string,
	data map[string]any,
) string {
	normalizedKey := webi18n.Key(strings.TrimSpace(string(key)))
	if appCtx == nil {
		return strings.TrimSpace(fallback)
	}
	value := strings.TrimSpace(appCtx.T(locale, normalizedKey, data))
	if value == "" || value == string(normalizedKey) {
		return strings.TrimSpace(fallback)
	}
	return value
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
	appCtx *appcore.Context,
	r *http.Request,
	locale string,
	alternateTypes map[string]string,
) (metagen.Alternates, error) {
	if appCtx == nil {
		return metagen.Alternates{}, fmt.Errorf("app context is required")
	}

	rootURL := strings.TrimSpace(appCtx.RootURL())
	if rootURL == "" {
		return metagen.Alternates{}, fmt.Errorf("BLOG_ROOT_URL is required for metadata alternates")
	}

	cfg, err := frameworki18n.NormalizeConfig(appCtx.I18nConfig())
	if err != nil {
		return metagen.Alternates{}, fmt.Errorf("normalize i18n config: %w", err)
	}

	return metagen.BuildAlternates(rootURL, cfg, locale, requestPathWithQuery(r), alternateTypes)
}

func requestPathWithQuery(r *http.Request) string {
	if r == nil || r.URL == nil {
		return "/"
	}
	pathValue := strings.TrimSpace(r.URL.Path)
	if pathValue == "" {
		pathValue = "/"
	}
	if strings.TrimSpace(r.URL.RawQuery) == "" {
		return pathValue
	}
	return pathValue + "?" + strings.TrimSpace(r.URL.RawQuery)
}

func notesRSSAlternateTypes(appCtx *appcore.Context, r *http.Request, locale string) map[string]string {
	if appCtx == nil {
		return nil
	}

	rootURL := strings.TrimSpace(appCtx.RootURL())
	if rootURL == "" {
		return nil
	}

	feedURL := joinRootAndPath(rootURL, "/feed.xml")
	if strings.TrimSpace(feedURL) == "" {
		return nil
	}

	query := url.Values{}
	query.Set("locale", strings.TrimSpace(locale))
	if r != nil && r.URL != nil {
		requestQuery := r.URL.Query()
		for _, key := range []string{"page", "author", "tag", "type", "q"} {
			value := strings.TrimSpace(requestQuery.Get(key))
			if value == "" {
				continue
			}
			query.Set(key, value)
		}
	}

	return map[string]string{
		"application/rss+xml": feedURL + "?" + query.Encode(),
	}
}

func firstListingImage(appCtx *appcore.Context, notes []notes.NoteSummary) *metagen.OpenGraphImage {
	for _, note := range notes {
		if image := noteImage(appCtx, note.MetaImage, note.Attachment); image != nil {
			return image
		}
	}
	return nil
}

func noteImage(
	appCtx *appcore.Context,
	metaImage *notes.Attachment,
	attachment *notes.Attachment,
) *metagen.OpenGraphImage {
	if image := noteAttachmentImage(appCtx, metaImage); image != nil {
		return image
	}
	return noteAttachmentImage(appCtx, attachment)
}

func noteAttachmentImage(appCtx *appcore.Context, attachment *notes.Attachment) *metagen.OpenGraphImage {
	if attachment == nil {
		return nil
	}
	thumbURL, thumbWidth, thumbHeight := appcore.ImageThumb(
		strings.TrimSpace(attachment.URL),
		attachment.Width,
		attachment.Height,
	)
	imageURL := absoluteMediaURL(appCtx, thumbURL)
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

func authorAvatarImage(appCtx *appcore.Context, author *notes.Author) *metagen.OpenGraphImage {
	if author == nil || author.Avatar == nil {
		return nil
	}
	thumbURL, thumbWidth, thumbHeight := appcore.ImageThumb(
		strings.TrimSpace(author.Avatar.URL),
		author.Avatar.Width,
		author.Avatar.Height,
	)
	imageURL := absoluteMediaURL(appCtx, thumbURL)
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

func absoluteMediaURL(appCtx *appcore.Context, rawURL string) string {
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

	root := ""
	if appCtx != nil {
		root = strings.TrimSpace(appCtx.RootURL())
	}
	return joinRootAndPath(root, parsed.Path)
}

func absoluteLocalizedURL(appCtx *appcore.Context, locale string, strippedPath string) string {
	localizedPath := appcore.LocalizeAppPath(locale, strippedPath)
	root := ""
	if appCtx != nil {
		root = strings.TrimSpace(appCtx.RootURL())
	}
	return joinRootAndPath(root, localizedPath)
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
