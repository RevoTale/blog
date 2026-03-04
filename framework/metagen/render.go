package metagen

import (
	"context"
	"html"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/a-h/templ"
)

const managedHeadAttribute = `data-metagen-managed="true"`

type componentFunc func(ctx context.Context, w io.Writer) error

func (f componentFunc) Render(ctx context.Context, w io.Writer) error {
	return f(ctx, w)
}

func Head(meta Metadata) templ.Component {
	return componentFunc(func(_ context.Context, w io.Writer) error {
		normalized := Normalize(meta)
		_, headHTML, err := renderManagedHead(normalized, true)
		if err != nil {
			return err
		}
		_, err = io.WriteString(w, headHTML)
		return err
	})
}

func renderManagedHead(meta Metadata, includeTitle bool) (string, string, error) {
	builder := strings.Builder{}

	if includeTitle && strings.TrimSpace(meta.Title) != "" {
		writeTag(&builder, "<title "+managedHeadAttribute+">"+escapeText(meta.Title)+"</title>")
	}

	if strings.TrimSpace(meta.Description) != "" {
		writeTag(&builder, metaTag("description", meta.Description))
	}

	writeAlternates(&builder, meta.Alternates)
	writeRobots(&builder, meta.Robots)
	writeAuthors(&builder, meta.Authors)

	if strings.TrimSpace(meta.Publisher) != "" {
		writeTag(&builder, metaTag("publisher", meta.Publisher))
	}

	writeOpenGraph(&builder, meta.OpenGraph)
	writeTwitter(&builder, meta.Twitter)
	writePinterest(&builder, meta.Pinterest)
	writeRawHead(&builder, meta.DangerRawHead)

	return meta.Title, builder.String(), nil
}

func writeRawHead(builder *strings.Builder, raw []string) {
	for _, value := range raw {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		writeTag(builder, trimmed)
	}
}

func writeAlternates(builder *strings.Builder, alternates Alternates) {
	if strings.TrimSpace(alternates.Canonical) != "" {
		writeTag(
			builder,
			`<link `+managedHeadAttribute+` rel="canonical" href="`+escapeAttr(alternates.Canonical)+`">`,
		)
	}

	languageKeys := sortedMapKeys(alternates.Languages)
	for _, language := range languageKeys {
		href := strings.TrimSpace(alternates.Languages[language])
		if href == "" {
			continue
		}
		writeTag(
			builder,
			`<link `+managedHeadAttribute+` rel="alternate" hreflang="`+
				escapeAttr(language)+`" href="`+escapeAttr(href)+`">`,
		)
	}

	typeKeys := sortedMapKeys(alternates.Types)
	for _, mediaType := range typeKeys {
		href := strings.TrimSpace(alternates.Types[mediaType])
		if href == "" {
			continue
		}
		writeTag(
			builder,
			`<link `+managedHeadAttribute+` rel="alternate" type="`+
				escapeAttr(mediaType)+`" href="`+escapeAttr(href)+`">`,
		)
	}
}

func writeRobots(builder *strings.Builder, robots *Robots) {
	if robots == nil {
		return
	}

	directives := make([]string, 0, 2)
	if robots.Index != nil {
		if *robots.Index {
			directives = append(directives, "index")
		} else {
			directives = append(directives, "noindex")
		}
	}
	if robots.Follow != nil {
		if *robots.Follow {
			directives = append(directives, "follow")
		} else {
			directives = append(directives, "nofollow")
		}
	}
	if len(directives) == 0 {
		return
	}

	writeTag(builder, metaTag("robots", strings.Join(directives, ", ")))
}

func writeOpenGraph(builder *strings.Builder, graph *OpenGraph) {
	if graph == nil {
		return
	}

	writeOpenGraphProperty(builder, "og:type", graph.Type)
	writeOpenGraphProperty(builder, "og:url", graph.URL)
	writeOpenGraphProperty(builder, "og:site_name", graph.SiteName)
	writeOpenGraphProperty(builder, "og:title", graph.Title)
	writeOpenGraphProperty(builder, "og:description", graph.Description)
	writeOpenGraphProperty(builder, "og:locale", graph.Locale)
	writeOpenGraphProperty(builder, "article:published_time", graph.PublishedTime)
	for _, author := range graph.Authors {
		writeOpenGraphProperty(builder, "article:author", author)
	}
	for _, tag := range graph.Tags {
		writeOpenGraphProperty(builder, "article:tag", tag)
	}

	for _, image := range graph.Images {
		writeOpenGraphProperty(builder, "og:image", image.URL)
		writeOpenGraphProperty(builder, "og:image:alt", image.Alt)
		if image.Width > 0 {
			writeOpenGraphProperty(builder, "og:image:width", strconv.Itoa(image.Width))
		}
		if image.Height > 0 {
			writeOpenGraphProperty(builder, "og:image:height", strconv.Itoa(image.Height))
		}
	}
}

func writeTwitter(builder *strings.Builder, twitter *Twitter) {
	if twitter == nil {
		return
	}

	writeTwitterName(builder, "twitter:card", twitter.Card)
	writeTwitterName(builder, "twitter:site", twitter.Site)
	writeTwitterName(builder, "twitter:creator", twitter.Creator)
	writeTwitterName(builder, "twitter:title", twitter.Title)
	writeTwitterName(builder, "twitter:description", twitter.Description)
	for _, image := range twitter.Images {
		writeTwitterName(builder, "twitter:image", image)
	}
}

func writeAuthors(builder *strings.Builder, authors []Author) {
	for _, author := range authors {
		if strings.TrimSpace(author.Name) != "" {
			writeTag(builder, metaTag("author", author.Name))
		}
		if strings.TrimSpace(author.URL) != "" {
			writeTag(
				builder,
				`<link `+managedHeadAttribute+` rel="author" href="`+escapeAttr(author.URL)+`">`,
			)
		}
	}
}

func writePinterest(builder *strings.Builder, pinterest *Pinterest) {
	if pinterest == nil || pinterest.RichPin == nil {
		return
	}

	writeTag(builder, metaTag("pinterest-rich-pin", strconv.FormatBool(*pinterest.RichPin)))
}

func writeOpenGraphProperty(builder *strings.Builder, property string, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	writeTag(
		builder,
		`<meta `+managedHeadAttribute+` property="`+escapeAttr(property)+`" content="`+escapeAttr(value)+`">`,
	)
}

func writeTwitterName(builder *strings.Builder, name string, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	writeTag(builder, metaTag(name, value))
}

func metaTag(name string, value string) string {
	return `<meta ` + managedHeadAttribute + ` name="` + escapeAttr(name) + `" content="` + escapeAttr(value) + `">`
}

func writeTag(builder *strings.Builder, tag string) {
	if builder.Len() > 0 {
		builder.WriteByte('\n')
	}
	builder.WriteString(tag)
}

func escapeAttr(value string) string {
	return html.EscapeString(strings.TrimSpace(value))
}

func escapeText(value string) string {
	return html.EscapeString(strings.TrimSpace(value))
}

func sortedMapKeys(values map[string]string) []string {
	if len(values) == 0 {
		return nil
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		keys = append(keys, trimmed)
	}
	sort.Strings(keys)
	return keys
}
