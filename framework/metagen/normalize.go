package metagen

import (
	"sort"
	"strings"
)

func Normalize(meta Metadata) Metadata {
	normalized := Metadata{
		Title:         strings.TrimSpace(meta.Title),
		Description:   strings.TrimSpace(meta.Description),
		Alternates:    normalizeAlternates(meta.Alternates),
		Authors:       normalizeAuthors(meta.Authors),
		Publisher:     strings.TrimSpace(meta.Publisher),
		DangerRawHead: normalizeRawHead(meta.DangerRawHead),
	}

	if robots := normalizeRobots(meta.Robots); robots != nil {
		normalized.Robots = robots
	}
	if graph := normalizeOpenGraph(meta.OpenGraph); graph != nil {
		if strings.TrimSpace(graph.Title) == "" {
			graph.Title = normalized.Title
		}
		if strings.TrimSpace(graph.Description) == "" {
			graph.Description = normalized.Description
		}
		if strings.TrimSpace(graph.URL) == "" {
			graph.URL = normalized.Alternates.Canonical
		}
		normalized.OpenGraph = graph
	}
	if twitter := normalizeTwitter(meta.Twitter); twitter != nil {
		if strings.TrimSpace(twitter.Title) == "" {
			twitter.Title = normalized.Title
		}
		if strings.TrimSpace(twitter.Description) == "" {
			twitter.Description = normalized.Description
		}
		normalized.Twitter = twitter
	}
	if pinterest := normalizePinterest(meta.Pinterest); pinterest != nil {
		normalized.Pinterest = pinterest
	}

	return normalized
}

func normalizeAlternates(alternates Alternates) Alternates {
	normalized := Alternates{
		Canonical: strings.TrimSpace(alternates.Canonical),
	}

	if len(alternates.Languages) > 0 {
		normalized.Languages = make(map[string]string, len(alternates.Languages))
		for language, href := range alternates.Languages {
			trimmedLanguage := strings.ToLower(strings.TrimSpace(language))
			trimmedHref := strings.TrimSpace(href)
			if trimmedLanguage == "" || trimmedHref == "" {
				continue
			}
			normalized.Languages[trimmedLanguage] = trimmedHref
		}
	}

	if len(alternates.Types) > 0 {
		normalized.Types = make(map[string]string, len(alternates.Types))
		for mediaType, href := range alternates.Types {
			trimmedType := strings.TrimSpace(mediaType)
			trimmedHref := strings.TrimSpace(href)
			if trimmedType == "" || trimmedHref == "" {
				continue
			}
			normalized.Types[trimmedType] = trimmedHref
		}
	}

	return normalized
}

func normalizeRobots(robots *Robots) *Robots {
	if robots == nil {
		return nil
	}

	normalized := &Robots{
		Index:  normalizeBoolPointer(robots.Index),
		Follow: normalizeBoolPointer(robots.Follow),
	}
	if normalized.Index == nil && normalized.Follow == nil {
		return nil
	}

	return normalized
}

func normalizeOpenGraph(graph *OpenGraph) *OpenGraph {
	if graph == nil {
		return nil
	}

	normalized := &OpenGraph{
		Type:          strings.TrimSpace(graph.Type),
		URL:           strings.TrimSpace(graph.URL),
		SiteName:      strings.TrimSpace(graph.SiteName),
		Title:         strings.TrimSpace(graph.Title),
		Description:   strings.TrimSpace(graph.Description),
		Locale:        strings.ToLower(strings.TrimSpace(graph.Locale)),
		PublishedTime: strings.TrimSpace(graph.PublishedTime),
		Authors:       normalizeOpenGraphStrings(graph.Authors),
		Tags:          normalizeOpenGraphStrings(graph.Tags),
		Images:        normalizeOpenGraphImages(graph.Images),
	}

	if normalized.Type == "" &&
		normalized.URL == "" &&
		normalized.SiteName == "" &&
		normalized.Title == "" &&
		normalized.Description == "" &&
		normalized.Locale == "" &&
		normalized.PublishedTime == "" &&
		len(normalized.Authors) == 0 &&
		len(normalized.Tags) == 0 &&
		len(normalized.Images) == 0 {
		return nil
	}

	return normalized
}

func normalizeTwitter(twitter *Twitter) *Twitter {
	if twitter == nil {
		return nil
	}

	normalized := &Twitter{
		Card:        strings.TrimSpace(twitter.Card),
		Site:        strings.TrimSpace(twitter.Site),
		Creator:     strings.TrimSpace(twitter.Creator),
		Title:       strings.TrimSpace(twitter.Title),
		Description: strings.TrimSpace(twitter.Description),
		Images:      normalizeTwitterImages(twitter.Images),
	}
	if normalized.Card == "" &&
		normalized.Site == "" &&
		normalized.Creator == "" &&
		normalized.Title == "" &&
		normalized.Description == "" &&
		len(normalized.Images) == 0 {
		return nil
	}

	return normalized
}

func normalizePinterest(pinterest *Pinterest) *Pinterest {
	if pinterest == nil || pinterest.RichPin == nil {
		return nil
	}

	return &Pinterest{RichPin: normalizeBoolPointer(pinterest.RichPin)}
}

func normalizeAuthors(authors []Author) []Author {
	if len(authors) == 0 {
		return nil
	}

	normalized := make([]Author, 0, len(authors))
	for _, author := range authors {
		trimmed := Author{
			Name: strings.TrimSpace(author.Name),
			URL:  strings.TrimSpace(author.URL),
		}
		if trimmed.Name == "" && trimmed.URL == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}

	sort.Slice(normalized, func(i int, j int) bool {
		leftName := strings.ToLower(normalized[i].Name)
		rightName := strings.ToLower(normalized[j].Name)
		if leftName != rightName {
			return leftName < rightName
		}
		return normalized[i].URL < normalized[j].URL
	})

	return normalized
}

func normalizeOpenGraphImages(images []OpenGraphImage) []OpenGraphImage {
	if len(images) == 0 {
		return nil
	}

	normalized := make([]OpenGraphImage, 0, len(images))
	for _, image := range images {
		trimmed := OpenGraphImage{
			URL:    strings.TrimSpace(image.URL),
			Alt:    strings.TrimSpace(image.Alt),
			Width:  image.Width,
			Height: image.Height,
		}
		if trimmed.URL == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}

	sort.Slice(normalized, func(i int, j int) bool {
		if normalized[i].URL != normalized[j].URL {
			return normalized[i].URL < normalized[j].URL
		}
		if normalized[i].Alt != normalized[j].Alt {
			return normalized[i].Alt < normalized[j].Alt
		}
		if normalized[i].Width != normalized[j].Width {
			return normalized[i].Width < normalized[j].Width
		}
		return normalized[i].Height < normalized[j].Height
	})

	return normalized
}

func normalizeOpenGraphStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}

	sort.Strings(normalized)
	return compactSortedStrings(normalized)
}

func normalizeTwitterImages(images []string) []string {
	if len(images) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(images))
	for _, image := range images {
		trimmed := strings.TrimSpace(image)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	sort.Strings(normalized)
	return compactSortedStrings(normalized)
}

func compactSortedStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	out := values[:0]
	var previous string
	for _, value := range values {
		if len(out) > 0 && value == previous {
			continue
		}
		out = append(out, value)
		previous = value
	}
	return out
}

func normalizeBoolPointer(value *bool) *bool {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func normalizeRawHead(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}

	if len(normalized) == 0 {
		return nil
	}
	return normalized
}
