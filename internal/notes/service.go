package notes

import (
	"context"
	"errors"
	"html/template"
	"net/url"
	"path"
	"strings"
	"time"

	"blog/internal/gql"
	md "blog/internal/markdown"
	genqlientgraphql "github.com/Khan/genqlient/graphql"
)

var ErrNotFound = errors.New("not found")

type Service struct {
	client   genqlientgraphql.Client
	pageSize int
	rootURL  string
}

type AuthorMedia struct {
	URL    string
	Alt    string
	Width  int
	Height int
}

type Author struct {
	Name   string
	Slug   string
	Bio    string
	Avatar *AuthorMedia
}

type Tag struct {
	Name  string
	Title string
}

type Attachment struct {
	URL      string
	Alt      string
	Width    int
	Height   int
	Filename string
	MIMEType string
}

type NoteSummary struct {
	ID          string
	Slug        string
	Title       string
	Excerpt     string
	PublishedAt string
	Description string
	Attachment  *Attachment
	Authors     []Author
	Tags        []Tag
}

type NoteDetail struct {
	ID          string
	Slug        string
	Title       string
	BodyHTML    template.HTML
	PublishedAt string
	Description string
	Attachment  *Attachment
	Authors     []Author
	Tags        []Tag
}

type NotesListResult struct {
	Notes      []NoteSummary
	Tags       []Tag
	ActiveTag  string
	Page       int
	TotalPages int
}

type AuthorPageResult struct {
	Author     Author
	Notes      []NoteSummary
	Page       int
	TotalPages int
}

func NewService(client genqlientgraphql.Client, pageSize int, rootURL string) *Service {
	if pageSize < 1 {
		pageSize = 12
	}

	return &Service{
		client:   client,
		pageSize: pageSize,
		rootURL:  strings.TrimSpace(rootURL),
	}
}

func (s *Service) ListNotes(ctx context.Context, page int, tagName string) (NotesListResult, error) {
	page = sanitizePage(page)
	result := NotesListResult{
		Page:       page,
		TotalPages: 1,
		ActiveTag:  tagName,
	}

	tagsResponse, err := gql.AvailableLongNoteTags(ctx, s.client)
	if err != nil {
		return NotesListResult{}, err
	}
	result.Tags = mapAvailableTags(tagsResponse)

	tagName = strings.TrimSpace(tagName)
	if tagName == "" {
		response, err := gql.ListNotes(ctx, s.client, page, s.pageSize)
		if err != nil {
			return NotesListResult{}, err
		}
		result.Notes, result.TotalPages = mapNotesList(response)
		if result.TotalPages < 1 {
			result.TotalPages = 1
		}
		return result, nil
	}

	tagIDs, err := s.findTagIDs(ctx, []string{tagName})
	if err != nil {
		return NotesListResult{}, err
	}
	if len(tagIDs) == 0 {
		result.Notes = []NoteSummary{}
		result.TotalPages = 1
		return result, nil
	}

	response, err := gql.ListNotesByTagIDs(ctx, s.client, page, s.pageSize, tagIDs)
	if err != nil {
		return NotesListResult{}, err
	}
	result.Notes, result.TotalPages = mapNotesListByTags(response)
	if result.TotalPages < 1 {
		result.TotalPages = 1
	}

	return result, nil
}

func (s *Service) GetNoteBySlug(ctx context.Context, slug string) (*NoteDetail, error) {
	response, err := gql.NoteBySlug(ctx, s.client, slug)
	if err != nil {
		return nil, err
	}

	if response == nil || response.Micro_posts == nil || len(response.Micro_posts.Docs) == 0 {
		return nil, ErrNotFound
	}

	doc := response.Micro_posts.Docs[0]
	translateLinks := noteTranslateLinks(doc)
	note := NoteDetail{
		ID:    doc.Id,
		Slug:  strOr(doc.Slug, slug),
		Title: pickTitle(doc.Title, doc.Slug, doc.Id),
		BodyHTML: md.ToHTML(strOr(doc.Content, ""), md.Options{
			TranslateLinks: translateLinks,
			RootURL:        s.rootURL,
		}),
		PublishedAt: formatDate(doc.PublishedAt),
		Attachment:  mapNoteAttachment(doc.Attachment),
		Authors:     mapNoteAuthors(doc.Authors),
		Tags:        mapNoteTags(doc.Tags),
	}

	if doc.Meta != nil {
		note.Description = strOr(doc.Meta.Description, "")
		if strings.TrimSpace(note.Title) == "" {
			note.Title = strOr(doc.Meta.Title, note.Slug)
		}
	}

	if strings.TrimSpace(note.Title) == "" {
		note.Title = note.Slug
	}

	return &note, nil
}

func (s *Service) GetAuthorPage(ctx context.Context, slug string, page int) (*AuthorPageResult, error) {
	page = sanitizePage(page)

	authorResponse, err := gql.AuthorBySlug(ctx, s.client, slug)
	if err != nil {
		return nil, err
	}
	if authorResponse == nil || authorResponse.Authors == nil || len(authorResponse.Authors.Docs) == 0 {
		return nil, ErrNotFound
	}

	author := mapAuthorFromAuthorDoc(authorResponse.Authors.Docs[0])
	if strings.TrimSpace(author.Slug) == "" {
		author.Slug = slug
	}

	notesResponse, err := gql.NotesByAuthorSlug(ctx, s.client, slug, page, s.pageSize)
	if err != nil {
		return nil, err
	}

	notes, totalPages := mapNotesByAuthorSlug(notesResponse)
	if totalPages < 1 {
		totalPages = 1
	}

	return &AuthorPageResult{
		Author:     author,
		Notes:      notes,
		Page:       page,
		TotalPages: totalPages,
	}, nil
}

func (s *Service) findTagIDs(ctx context.Context, tagNames []string) ([]string, error) {
	if len(tagNames) == 0 {
		return nil, nil
	}

	response, err := gql.TagIDsByNames(ctx, s.client, tagNames)
	if err != nil {
		return nil, err
	}

	if response == nil || response.Tags == nil {
		return nil, nil
	}

	tagIDs := make([]string, 0, len(response.Tags.Docs))
	for _, tag := range response.Tags.Docs {
		tagIDs = append(tagIDs, tag.Id)
	}

	return tagIDs, nil
}

func noteTranslateLinks(doc gql.NoteBySlugMicro_postsDocsMicro_post) map[string]string {
	links := make(map[string]string, len(doc.ExternalLinks)+len(doc.LinkedMicroPosts))

	for _, external := range doc.ExternalLinks {
		if strings.TrimSpace(external.Target_url) == "" {
			continue
		}
		links[external.Id] = external.Target_url
	}

	for _, linked := range doc.LinkedMicroPosts {
		if linked.Slug == nil || strings.TrimSpace(*linked.Slug) == "" {
			continue
		}
		links[linked.Id] = "/note/" + strings.TrimSpace(*linked.Slug)
	}

	return links
}

func mapAvailableTags(response *gql.AvailableLongNoteTagsResponse) []Tag {
	if response == nil {
		return nil
	}

	out := make([]Tag, 0, len(response.AvailableTagsByMicroPostType))
	for _, item := range response.AvailableTagsByMicroPostType {
		title := strOr(item.Title, item.Name)
		out = append(out, Tag{
			Name:  item.Name,
			Title: title,
		})
	}

	return out
}

func mapNotesList(response *gql.ListNotesResponse) ([]NoteSummary, int) {
	if response == nil || response.Micro_posts == nil {
		return []NoteSummary{}, 1
	}

	items := make([]NoteSummary, 0, len(response.Micro_posts.Docs))
	for _, doc := range response.Micro_posts.Docs {
		description := ""
		if doc.Meta != nil {
			description = strOr(doc.Meta.Description, "")
		}
		items = append(items, summaryFromListDoc(
			doc.Id,
			doc.Slug,
			doc.Title,
			doc.Content,
			doc.PublishedAt,
			description,
			mapListAttachment(doc.Attachment),
			mapListAuthors(doc.Authors),
			mapListTags(doc.Tags),
		))
	}

	return items, response.Micro_posts.TotalPages
}

func mapNotesListByTags(response *gql.ListNotesByTagIDsResponse) ([]NoteSummary, int) {
	if response == nil || response.Micro_posts == nil {
		return []NoteSummary{}, 1
	}

	items := make([]NoteSummary, 0, len(response.Micro_posts.Docs))
	for _, doc := range response.Micro_posts.Docs {
		description := ""
		if doc.Meta != nil {
			description = strOr(doc.Meta.Description, "")
		}
		items = append(items, summaryFromListDoc(
			doc.Id,
			doc.Slug,
			doc.Title,
			doc.Content,
			doc.PublishedAt,
			description,
			mapTagListAttachment(doc.Attachment),
			mapTagListAuthors(doc.Authors),
			mapTagListTags(doc.Tags),
		))
	}

	return items, response.Micro_posts.TotalPages
}

func mapNotesByAuthorSlug(response *gql.NotesByAuthorSlugResponse) ([]NoteSummary, int) {
	if response == nil || response.Micro_posts == nil {
		return []NoteSummary{}, 1
	}

	items := make([]NoteSummary, 0, len(response.Micro_posts.Docs))
	for _, doc := range response.Micro_posts.Docs {
		description := ""
		if doc.Meta != nil {
			description = strOr(doc.Meta.Description, "")
		}
		items = append(items, summaryFromListDoc(
			doc.Id,
			doc.Slug,
			doc.Title,
			doc.Content,
			doc.PublishedAt,
			description,
			mapAuthorListAttachment(doc.Attachment),
			mapAuthorListAuthors(doc.Authors),
			mapAuthorListTags(doc.Tags),
		))
	}

	return items, response.Micro_posts.TotalPages
}

func summaryFromListDoc(
	id string,
	slug *string,
	title *string,
	content *string,
	publishedAt *string,
	description string,
	attachment *Attachment,
	authors []Author,
	tags []Tag,
) NoteSummary {
	contentText := strOr(content, "")
	if description == "" {
		description = md.Excerpt(contentText, 220)
	}

	return NoteSummary{
		ID:          id,
		Slug:        strOr(slug, id),
		Title:       pickTitle(title, slug, id),
		Excerpt:     md.Excerpt(contentText, 260),
		PublishedAt: formatDate(publishedAt),
		Description: description,
		Attachment:  attachment,
		Authors:     authors,
		Tags:        tags,
	}
}

func mapNoteAuthors(authors []gql.NoteBySlugMicro_postsDocsMicro_postAuthorsAuthor) []Author {
	out := make([]Author, 0, len(authors))
	for _, item := range authors {
		var avatar *AuthorMedia
		if item.Avatar != nil {
			avatar = newAvatar(item.Avatar.Url, item.Avatar.Alt, item.Avatar.Width, item.Avatar.Height)
		}
		out = append(out, Author{
			Name:   strOr(item.Name, item.Slug),
			Slug:   item.Slug,
			Bio:    strOr(item.Bio, ""),
			Avatar: avatar,
		})
	}

	return out
}

func mapListAuthors(authors []gql.ListNotesMicro_postsDocsMicro_postAuthorsAuthor) []Author {
	out := make([]Author, 0, len(authors))
	for _, item := range authors {
		var avatar *AuthorMedia
		if item.Avatar != nil {
			avatar = newAvatar(item.Avatar.Url, item.Avatar.Alt, item.Avatar.Width, item.Avatar.Height)
		}
		out = append(out, Author{
			Name:   strOr(item.Name, item.Slug),
			Slug:   item.Slug,
			Bio:    strOr(item.Bio, ""),
			Avatar: avatar,
		})
	}

	return out
}

func mapTagListAuthors(authors []gql.ListNotesByTagIDsMicro_postsDocsMicro_postAuthorsAuthor) []Author {
	out := make([]Author, 0, len(authors))
	for _, item := range authors {
		var avatar *AuthorMedia
		if item.Avatar != nil {
			avatar = newAvatar(item.Avatar.Url, item.Avatar.Alt, item.Avatar.Width, item.Avatar.Height)
		}
		out = append(out, Author{
			Name:   strOr(item.Name, item.Slug),
			Slug:   item.Slug,
			Bio:    strOr(item.Bio, ""),
			Avatar: avatar,
		})
	}

	return out
}

func mapAuthorListAuthors(authors []gql.NotesByAuthorSlugMicro_postsDocsMicro_postAuthorsAuthor) []Author {
	out := make([]Author, 0, len(authors))
	for _, item := range authors {
		var avatar *AuthorMedia
		if item.Avatar != nil {
			avatar = newAvatar(item.Avatar.Url, item.Avatar.Alt, item.Avatar.Width, item.Avatar.Height)
		}
		out = append(out, Author{
			Name:   strOr(item.Name, item.Slug),
			Slug:   item.Slug,
			Bio:    strOr(item.Bio, ""),
			Avatar: avatar,
		})
	}

	return out
}

func mapNoteAttachment(attachment *gql.NoteBySlugMicro_postsDocsMicro_postAttachmentMedia) *Attachment {
	if attachment == nil {
		return nil
	}

	return newAttachment(
		attachment.Url,
		attachment.Alt,
		attachment.Filename,
		attachment.MimeType,
		attachment.Width,
		attachment.Height,
	)
}

func mapListAttachment(attachment *gql.ListNotesMicro_postsDocsMicro_postAttachmentMedia) *Attachment {
	if attachment == nil {
		return nil
	}

	return newAttachment(
		attachment.Url,
		attachment.Alt,
		attachment.Filename,
		attachment.MimeType,
		attachment.Width,
		attachment.Height,
	)
}

func mapTagListAttachment(attachment *gql.ListNotesByTagIDsMicro_postsDocsMicro_postAttachmentMedia) *Attachment {
	if attachment == nil {
		return nil
	}

	return newAttachment(
		attachment.Url,
		attachment.Alt,
		attachment.Filename,
		attachment.MimeType,
		attachment.Width,
		attachment.Height,
	)
}

func mapAuthorListAttachment(attachment *gql.NotesByAuthorSlugMicro_postsDocsMicro_postAttachmentMedia) *Attachment {
	if attachment == nil {
		return nil
	}

	return newAttachment(
		attachment.Url,
		attachment.Alt,
		attachment.Filename,
		attachment.MimeType,
		attachment.Width,
		attachment.Height,
	)
}

func mapNoteTags(tags []gql.NoteBySlugMicro_postsDocsMicro_postTagsTag) []Tag {
	out := make([]Tag, 0, len(tags))
	for _, item := range tags {
		out = append(out, Tag{
			Name:  item.Name,
			Title: strOr(item.Title, item.Name),
		})
	}

	return out
}

func mapListTags(tags []gql.ListNotesMicro_postsDocsMicro_postTagsTag) []Tag {
	out := make([]Tag, 0, len(tags))
	for _, item := range tags {
		out = append(out, Tag{
			Name:  item.Name,
			Title: strOr(item.Title, item.Name),
		})
	}

	return out
}

func mapTagListTags(tags []gql.ListNotesByTagIDsMicro_postsDocsMicro_postTagsTag) []Tag {
	out := make([]Tag, 0, len(tags))
	for _, item := range tags {
		out = append(out, Tag{
			Name:  item.Name,
			Title: strOr(item.Title, item.Name),
		})
	}

	return out
}

func mapAuthorListTags(tags []gql.NotesByAuthorSlugMicro_postsDocsMicro_postTagsTag) []Tag {
	out := make([]Tag, 0, len(tags))
	for _, item := range tags {
		out = append(out, Tag{
			Name:  item.Name,
			Title: strOr(item.Title, item.Name),
		})
	}

	return out
}

func mapAuthorFromAuthorDoc(doc gql.AuthorBySlugAuthorsDocsAuthor) Author {
	var avatar *AuthorMedia
	if doc.Avatar != nil {
		avatar = newAvatar(doc.Avatar.Url, doc.Avatar.Alt, doc.Avatar.Width, doc.Avatar.Height)
	}
	return Author{
		Name:   strOr(doc.Name, doc.Slug),
		Slug:   doc.Slug,
		Bio:    strOr(doc.Bio, ""),
		Avatar: avatar,
	}
}

func newAvatar(url *string, alt *string, width *float64, height *float64) *AuthorMedia {
	if url == nil || strings.TrimSpace(*url) == "" {
		return nil
	}

	return &AuthorMedia{
		URL:    strOr(url, ""),
		Alt:    strOr(alt, ""),
		Width:  int(floatOr(width, 0)),
		Height: int(floatOr(height, 0)),
	}
}

func newAttachment(
	urlValue *string,
	alt *string,
	filename *string,
	mimeType *string,
	width *float64,
	height *float64,
) *Attachment {
	urlString := strOr(urlValue, "")
	if urlString == "" {
		return nil
	}

	name := strOr(filename, "")
	if name == "" {
		name = filenameFromURL(urlString)
	}

	return &Attachment{
		URL:      urlString,
		Alt:      strOr(alt, ""),
		Width:    int(floatOr(width, 0)),
		Height:   int(floatOr(height, 0)),
		Filename: name,
		MIMEType: strOr(mimeType, ""),
	}
}

func filenameFromURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	base := strings.TrimSpace(path.Base(parsed.Path))
	if base == "." || base == "/" {
		return ""
	}

	return base
}

func pickTitle(title *string, slug *string, fallback string) string {
	if v := strings.TrimSpace(strOr(title, "")); v != "" {
		return v
	}
	if v := strings.TrimSpace(strOr(slug, "")); v != "" {
		return v
	}
	return fallback
}

func formatDate(raw *string) string {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return ""
	}

	parsed, err := time.Parse(time.RFC3339, *raw)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339Nano, *raw)
		if err != nil {
			return *raw
		}
	}

	return parsed.Format("2006-01-02")
}

func strOr(value *string, fallback string) string {
	if value == nil {
		return fallback
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return fallback
	}

	return trimmed
}

func floatOr(value *float64, fallback float64) float64 {
	if value == nil {
		return fallback
	}

	return *value
}

func sanitizePage(page int) int {
	if page < 1 {
		return 1
	}

	return page
}
