package notes

import (
	"context"
	"errors"
	"html/template"
	"net/url"
	"path"
	"sort"
	"strings"
	"time"

	"blog/internal/gql"
	md "blog/internal/markdown"
	genqlientgraphql "github.com/Khan/genqlient/graphql"
)

var ErrNotFound = errors.New("not found")

type NoteType string

const (
	NoteTypeAll   NoteType = "all"
	NoteTypeLong  NoteType = "long"
	NoteTypeShort NoteType = "short"
)

type ListFilter struct {
	Page       int
	AuthorSlug string
	TagName    string
	Type       NoteType
}

type ListOptions struct {
	RequireAuthor bool
	RequireTag    bool
}

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
	Notes        []NoteSummary
	Authors      []Author
	Tags         []Tag
	ActiveFilter ListFilter
	ActiveAuthor *Author
	ActiveTag    *Tag
	Page         int
	TotalPages   int
}

type AuthorPageResult struct {
	Author     Author
	Notes      []NoteSummary
	Page       int
	TotalPages int
	Filter     ListFilter
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

func ParseNoteType(raw string) NoteType {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "long":
		return NoteTypeLong
	case "short":
		return NoteTypeShort
	default:
		return NoteTypeAll
	}
}

func (t NoteType) QueryValue() string {
	if t == NoteTypeLong || t == NoteTypeShort {
		return string(t)
	}

	return ""
}

func (s *Service) ListNotes(ctx context.Context, filter ListFilter, options ListOptions) (NotesListResult, error) {
	filter = normalizeFilter(filter)
	result := NotesListResult{
		ActiveFilter: filter,
		Page:         filter.Page,
		TotalPages:   1,
	}

	authorsResponse, err := gql.AvailableAuthors(ctx, s.client, 200)
	if err != nil {
		return NotesListResult{}, err
	}
	result.Authors = mapAvailableAuthors(authorsResponse)

	tagsResponse, err := gql.AvailableTagsByPostType(ctx, s.client, postTypeFilterArg(filter.Type))
	if err != nil {
		return NotesListResult{}, err
	}
	result.Tags = mapAvailableTags(tagsResponse)

	if filter.AuthorSlug != "" {
		author, authorErr := s.GetAuthorBySlug(ctx, filter.AuthorSlug)
		if authorErr != nil {
			if errors.Is(authorErr, ErrNotFound) && !options.RequireAuthor {
				result.Notes = []NoteSummary{}
				result.TotalPages = 1
				return result, nil
			}

			return NotesListResult{}, authorErr
		}
		result.ActiveAuthor = author
		result.Authors = mergeAuthor(result.Authors, *author)
	}

	tagIDs := []string{}
	if filter.TagName != "" {
		tag, tagErr := s.GetTagByName(ctx, filter.TagName)
		if tagErr != nil {
			if errors.Is(tagErr, ErrNotFound) && !options.RequireTag {
				result.Notes = []NoteSummary{}
				result.TotalPages = 1
				return result, nil
			}

			return NotesListResult{}, tagErr
		}
		result.ActiveTag = tag
		result.Tags = mergeTag(result.Tags, *tag)

		tagIDs, err = s.findTagIDs(ctx, []string{filter.TagName})
		if err != nil {
			return NotesListResult{}, err
		}
		if len(tagIDs) == 0 {
			if options.RequireTag {
				return NotesListResult{}, ErrNotFound
			}

			result.Notes = []NoteSummary{}
			result.TotalPages = 1
			return result, nil
		}
	}

	notes, totalPages, err := s.listNotesByFilter(ctx, filter, tagIDs)
	if err != nil {
		return NotesListResult{}, err
	}
	if totalPages < 1 {
		totalPages = 1
	}

	result.Notes = notes
	result.TotalPages = totalPages

	if result.ActiveTag == nil && filter.TagName != "" {
		result.ActiveTag = findTagByName(result.Tags, filter.TagName)
	}

	result.Authors = mergeAuthorsFromNotes(result.Authors, notes)
	result.Tags = mergeTagsFromNotes(result.Tags, notes)

	return result, nil
}

func (s *Service) listNotesByFilter(
	ctx context.Context,
	filter ListFilter,
	tagIDs []string,
) ([]NoteSummary, int, error) {
	hasAuthor := filter.AuthorSlug != ""
	hasTag := len(tagIDs) > 0
	hasType := filter.Type == NoteTypeLong || filter.Type == NoteTypeShort

	postType, _ := toPostTypeInput(filter.Type)

	switch {
	case hasAuthor && hasTag && hasType:
		response, err := gql.ListNotesByAuthorTagIDsAndType(
			ctx,
			s.client,
			filter.AuthorSlug,
			filter.Page,
			s.pageSize,
			tagIDs,
			postType,
		)
		if err != nil {
			return nil, 0, err
		}
		notes, totalPages := mapNotesListByAuthorTagIDsAndType(response)
		return notes, totalPages, nil

	case hasAuthor && hasTag:
		response, err := gql.ListNotesByAuthorAndTagIDs(ctx, s.client, filter.AuthorSlug, filter.Page, s.pageSize, tagIDs)
		if err != nil {
			return nil, 0, err
		}
		notes, totalPages := mapNotesListByAuthorAndTagIDs(response)
		return notes, totalPages, nil

	case hasAuthor && hasType:
		response, err := gql.NotesByAuthorSlugAndType(
			ctx,
			s.client,
			filter.AuthorSlug,
			filter.Page,
			s.pageSize,
			postType,
		)
		if err != nil {
			return nil, 0, err
		}
		notes, totalPages := mapNotesByAuthorSlugAndType(response)
		return notes, totalPages, nil

	case hasAuthor:
		response, err := gql.NotesByAuthorSlug(ctx, s.client, filter.AuthorSlug, filter.Page, s.pageSize)
		if err != nil {
			return nil, 0, err
		}
		notes, totalPages := mapNotesByAuthorSlug(response)
		return notes, totalPages, nil

	case hasTag && hasType:
		response, err := gql.ListNotesByTagIDsAndType(ctx, s.client, filter.Page, s.pageSize, tagIDs, postType)
		if err != nil {
			return nil, 0, err
		}
		notes, totalPages := mapNotesListByTagIDsAndType(response)
		return notes, totalPages, nil

	case hasTag:
		response, err := gql.ListNotesByTagIDs(ctx, s.client, filter.Page, s.pageSize, tagIDs)
		if err != nil {
			return nil, 0, err
		}
		notes, totalPages := mapNotesListByTags(response)
		return notes, totalPages, nil

	case hasType:
		response, err := gql.ListNotesByType(ctx, s.client, filter.Page, s.pageSize, postType)
		if err != nil {
			return nil, 0, err
		}
		notes, totalPages := mapNotesListByType(response)
		return notes, totalPages, nil

	default:
		response, err := gql.ListNotes(ctx, s.client, filter.Page, s.pageSize)
		if err != nil {
			return nil, 0, err
		}
		notes, totalPages := mapNotesList(response)
		return notes, totalPages, nil
	}
}

func (s *Service) GetAuthorBySlug(ctx context.Context, slug string) (*Author, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil, ErrNotFound
	}

	response, err := gql.AuthorBySlug(ctx, s.client, slug)
	if err != nil {
		return nil, err
	}
	if response == nil || response.Authors == nil || len(response.Authors.Docs) == 0 {
		return nil, ErrNotFound
	}

	author := mapAuthorFromAuthorDoc(response.Authors.Docs[0])
	if strings.TrimSpace(author.Slug) == "" {
		author.Slug = slug
	}

	return &author, nil
}

func (s *Service) GetTagByName(ctx context.Context, name string) (*Tag, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrNotFound
	}

	response, err := gql.TagByName(ctx, s.client, name)
	if err != nil {
		return nil, err
	}
	if response == nil || response.Tags == nil || len(response.Tags.Docs) == 0 {
		return nil, ErrNotFound
	}

	tag := mapTagFromTagDoc(response.Tags.Docs[0])
	if strings.TrimSpace(tag.Name) == "" {
		tag.Name = name
	}

	return &tag, nil
}

func (s *Service) GetAuthorPage(ctx context.Context, slug string, page int) (*AuthorPageResult, error) {
	filter := ListFilter{
		Page:       sanitizePage(page),
		AuthorSlug: strings.TrimSpace(slug),
		Type:       NoteTypeAll,
	}

	result, err := s.ListNotes(ctx, filter, ListOptions{RequireAuthor: true})
	if err != nil {
		return nil, err
	}
	if result.ActiveAuthor == nil {
		return nil, ErrNotFound
	}

	return &AuthorPageResult{
		Author:     *result.ActiveAuthor,
		Notes:      result.Notes,
		Page:       result.Page,
		TotalPages: result.TotalPages,
		Filter:     result.ActiveFilter,
	}, nil
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

func mapAvailableTags(response *gql.AvailableTagsByPostTypeResponse) []Tag {
	if response == nil {
		return []Tag{}
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

func mapAvailableAuthors(response *gql.AvailableAuthorsResponse) []Author {
	if response == nil || response.Authors == nil {
		return []Author{}
	}

	out := make([]Author, 0, len(response.Authors.Docs))
	for _, item := range response.Authors.Docs {
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

	sort.Slice(out, func(i int, j int) bool {
		left := strings.ToLower(strings.TrimSpace(out[i].Name))
		right := strings.ToLower(strings.TrimSpace(out[j].Name))
		if left == right {
			return out[i].Slug < out[j].Slug
		}

		return left < right
	})

	return out
}

func mapTagFromTagDoc(doc gql.TagByNameTagsDocsTag) Tag {
	return Tag{
		Name:  doc.Name,
		Title: strOr(doc.Title, doc.Name),
	}
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

func mapNotesListByType(response *gql.ListNotesByTypeResponse) ([]NoteSummary, int) {
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
			mapListByTypeAttachment(doc.Attachment),
			mapListByTypeAuthors(doc.Authors),
			mapListByTypeTags(doc.Tags),
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

func mapNotesListByTagIDsAndType(response *gql.ListNotesByTagIDsAndTypeResponse) ([]NoteSummary, int) {
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
			mapTagByTypeAttachment(doc.Attachment),
			mapTagByTypeAuthors(doc.Authors),
			mapTagByTypeTags(doc.Tags),
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

func mapNotesByAuthorSlugAndType(response *gql.NotesByAuthorSlugAndTypeResponse) ([]NoteSummary, int) {
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
			mapAuthorByTypeAttachment(doc.Attachment),
			mapAuthorByTypeAuthors(doc.Authors),
			mapAuthorByTypeTags(doc.Tags),
		))
	}

	return items, response.Micro_posts.TotalPages
}

func mapNotesListByAuthorAndTagIDs(response *gql.ListNotesByAuthorAndTagIDsResponse) ([]NoteSummary, int) {
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
			mapAuthorTagAttachment(doc.Attachment),
			mapAuthorTagAuthors(doc.Authors),
			mapAuthorTagTags(doc.Tags),
		))
	}

	return items, response.Micro_posts.TotalPages
}

func mapNotesListByAuthorTagIDsAndType(response *gql.ListNotesByAuthorTagIDsAndTypeResponse) ([]NoteSummary, int) {
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
			mapAuthorTagTypeAttachment(doc.Attachment),
			mapAuthorTagTypeAuthors(doc.Authors),
			mapAuthorTagTypeTags(doc.Tags),
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

func mapListAuthors(authors []gql.NoteListDocAuthorsAuthor) []Author {
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

func mapListByTypeAuthors(authors []gql.NoteListDocAuthorsAuthor) []Author {
	return mapListAuthors(authors)
}

func mapTagListAuthors(authors []gql.NoteListDocAuthorsAuthor) []Author {
	return mapListAuthors(authors)
}

func mapTagByTypeAuthors(authors []gql.NoteListDocAuthorsAuthor) []Author {
	return mapListAuthors(authors)
}

func mapAuthorListAuthors(authors []gql.NoteListDocAuthorsAuthor) []Author {
	return mapListAuthors(authors)
}

func mapAuthorByTypeAuthors(authors []gql.NoteListDocAuthorsAuthor) []Author {
	return mapListAuthors(authors)
}

func mapAuthorTagAuthors(authors []gql.NoteListDocAuthorsAuthor) []Author {
	return mapListAuthors(authors)
}

func mapAuthorTagTypeAuthors(authors []gql.NoteListDocAuthorsAuthor) []Author {
	return mapListAuthors(authors)
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

func mapListAttachment(attachment *gql.NoteListDocAttachmentMedia) *Attachment {
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

func mapListByTypeAttachment(attachment *gql.NoteListDocAttachmentMedia) *Attachment {
	return mapListAttachment(attachment)
}

func mapTagListAttachment(attachment *gql.NoteListDocAttachmentMedia) *Attachment {
	return mapListAttachment(attachment)
}

func mapTagByTypeAttachment(attachment *gql.NoteListDocAttachmentMedia) *Attachment {
	return mapListAttachment(attachment)
}

func mapAuthorListAttachment(attachment *gql.NoteListDocAttachmentMedia) *Attachment {
	return mapListAttachment(attachment)
}

func mapAuthorByTypeAttachment(attachment *gql.NoteListDocAttachmentMedia) *Attachment {
	return mapListAttachment(attachment)
}

func mapAuthorTagAttachment(attachment *gql.NoteListDocAttachmentMedia) *Attachment {
	return mapListAttachment(attachment)
}

func mapAuthorTagTypeAttachment(attachment *gql.NoteListDocAttachmentMedia) *Attachment {
	return mapListAttachment(attachment)
}

func mapNoteTags(tags []gql.NoteBySlugMicro_postsDocsMicro_postTagsTag) []Tag {
	out := make([]Tag, 0, len(tags))
	for _, item := range tags {
		out = append(out, Tag{Name: item.Name, Title: strOr(item.Title, item.Name)})
	}

	return out
}

func mapListTags(tags []gql.NoteListDocTagsTag) []Tag {
	out := make([]Tag, 0, len(tags))
	for _, item := range tags {
		out = append(out, Tag{Name: item.Name, Title: strOr(item.Title, item.Name)})
	}

	return out
}

func mapListByTypeTags(tags []gql.NoteListDocTagsTag) []Tag {
	return mapListTags(tags)
}

func mapTagListTags(tags []gql.NoteListDocTagsTag) []Tag {
	return mapListTags(tags)
}

func mapTagByTypeTags(tags []gql.NoteListDocTagsTag) []Tag {
	return mapListTags(tags)
}

func mapAuthorListTags(tags []gql.NoteListDocTagsTag) []Tag {
	return mapListTags(tags)
}

func mapAuthorByTypeTags(tags []gql.NoteListDocTagsTag) []Tag {
	return mapListTags(tags)
}

func mapAuthorTagTags(tags []gql.NoteListDocTagsTag) []Tag {
	return mapListTags(tags)
}

func mapAuthorTagTypeTags(tags []gql.NoteListDocTagsTag) []Tag {
	return mapListTags(tags)
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

func mergeAuthor(authors []Author, author Author) []Author {
	for _, existing := range authors {
		if existing.Slug == author.Slug {
			return authors
		}
	}

	return append(authors, author)
}

func mergeAuthorsFromNotes(authors []Author, noteItems []NoteSummary) []Author {
	out := make([]Author, 0, len(authors)+len(noteItems))
	out = append(out, authors...)

	for _, note := range noteItems {
		for _, author := range note.Authors {
			out = mergeAuthor(out, author)
		}
	}

	sort.Slice(out, func(i int, j int) bool {
		left := strings.ToLower(strings.TrimSpace(out[i].Name))
		right := strings.ToLower(strings.TrimSpace(out[j].Name))
		if left == right {
			return out[i].Slug < out[j].Slug
		}

		return left < right
	})

	return out
}

func mergeTag(tags []Tag, tag Tag) []Tag {
	for _, existing := range tags {
		if existing.Name == tag.Name {
			return tags
		}
	}

	return append(tags, tag)
}

func mergeTagsFromNotes(tags []Tag, noteItems []NoteSummary) []Tag {
	out := make([]Tag, 0, len(tags)+len(noteItems))
	out = append(out, tags...)

	for _, note := range noteItems {
		for _, tag := range note.Tags {
			out = mergeTag(out, tag)
		}
	}

	sort.Slice(out, func(i int, j int) bool {
		left := strings.ToLower(strings.TrimSpace(out[i].Title))
		right := strings.ToLower(strings.TrimSpace(out[j].Title))
		if left == right {
			return out[i].Name < out[j].Name
		}

		return left < right
	})

	return out
}

func findTagByName(tags []Tag, name string) *Tag {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}

	for _, tag := range tags {
		if tag.Name == name {
			copy := tag
			return &copy
		}
	}

	return nil
}

func normalizeFilter(filter ListFilter) ListFilter {
	filter.Page = sanitizePage(filter.Page)
	filter.AuthorSlug = strings.TrimSpace(filter.AuthorSlug)
	filter.TagName = strings.TrimSpace(filter.TagName)
	filter.Type = ParseNoteType(string(filter.Type))

	return filter
}

func postTypeFilterArg(noteType NoteType) *string {
	if noteType == NoteTypeLong || noteType == NoteTypeShort {
		value := string(noteType)
		return &value
	}

	return nil
}

func toPostTypeInput(noteType NoteType) (gql.Micro_post_post_type_Input, bool) {
	switch noteType {
	case NoteTypeLong:
		return gql.Micro_post_post_type_InputLong, true
	case NoteTypeShort:
		return gql.Micro_post_post_type_InputShort, true
	default:
		return "", false
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
