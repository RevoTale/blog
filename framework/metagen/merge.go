package metagen

import "strings"

// Merge combines parent and child metadata.
// Child values override parent values when non-zero, and DangerRawHead appends.
func Merge(parent Metadata, child Metadata) Metadata {
	merged := Normalize(parent)
	child = Normalize(child)

	if strings.TrimSpace(child.Title) != "" {
		merged.Title = child.Title
	}
	if strings.TrimSpace(child.Description) != "" {
		merged.Description = child.Description
	}
	if strings.TrimSpace(child.Alternates.Canonical) != "" ||
		len(child.Alternates.Languages) > 0 ||
		len(child.Alternates.Types) > 0 {
		merged.Alternates = child.Alternates
	}
	if child.Robots != nil {
		merged.Robots = child.Robots
	}
	if child.OpenGraph != nil {
		merged.OpenGraph = child.OpenGraph
	}
	if child.Twitter != nil {
		merged.Twitter = child.Twitter
	}
	if len(child.Authors) > 0 {
		merged.Authors = child.Authors
	}
	if strings.TrimSpace(child.Publisher) != "" {
		merged.Publisher = child.Publisher
	}
	if child.Pinterest != nil {
		merged.Pinterest = child.Pinterest
	}

	if len(child.DangerRawHead) > 0 {
		merged.DangerRawHead = append(merged.DangerRawHead, child.DangerRawHead...)
	}

	return Normalize(merged)
}

// MergeAll merges multiple metadata layers in order.
func MergeAll(layers ...Metadata) Metadata {
	merged := Metadata{}
	for _, layer := range layers {
		merged = Merge(merged, layer)
	}
	return merged
}
