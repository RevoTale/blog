package web

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strings"
)

const pageTemplateName = "page.templ"

var dynamicSegmentNamePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)

type pathSegment struct {
	name    string
	isParam bool
}

type appRoute struct {
	id          string
	segments    []pathSegment
	staticCount int
	patternKey  string
}

type AppRouteMatch struct {
	ID     string
	Params map[string]string
}

func (m AppRouteMatch) Param(name string) (string, bool) {
	if m.Params == nil {
		return "", false
	}

	value, ok := m.Params[name]
	return value, ok
}

type AppRouter struct {
	routes []appRoute
}

func NewAppRouter(embedded fs.FS, root string) (*AppRouter, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, errors.New("app router root cannot be empty")
	}

	routes := make([]appRoute, 0, 8)
	seenPattern := make(map[string]string)

	walkErr := fs.WalkDir(embedded, root, func(filePath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if path.Base(filePath) != pageTemplateName {
			return nil
		}

		relPath := strings.TrimPrefix(filePath, root+"/")
		if relPath == filePath {
			return fmt.Errorf("compute route path for %q under root %q", filePath, root)
		}

		route, parseErr := parseAppRoute(relPath)
		if parseErr != nil {
			return parseErr
		}

		if existing, ok := seenPattern[route.patternKey]; ok {
			return fmt.Errorf("route pattern conflict: %q and %q", existing, route.id)
		}
		seenPattern[route.patternKey] = route.id
		routes = append(routes, route)
		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("walk app directory: %w", walkErr)
	}

	if len(routes) == 0 {
		return nil, errors.New("no page.templ routes found")
	}

	sort.Slice(routes, func(i int, j int) bool {
		left := routes[i]
		right := routes[j]

		if left.staticCount != right.staticCount {
			return left.staticCount > right.staticCount
		}
		if len(left.segments) != len(right.segments) {
			return len(left.segments) > len(right.segments)
		}
		return left.id < right.id
	})

	return &AppRouter{routes: routes}, nil
}

func parseAppRoute(relPath string) (appRoute, error) {
	cleaned := path.Clean(strings.TrimSpace(relPath))
	if cleaned == "" || cleaned == "." {
		return appRoute{}, errors.New("route path cannot be empty")
	}

	id := ""
	if cleaned != pageTemplateName {
		suffix := "/" + pageTemplateName
		if !strings.HasSuffix(cleaned, suffix) {
			return appRoute{}, fmt.Errorf("route file %q must end with %q", relPath, suffix)
		}
		id = strings.TrimSuffix(cleaned, suffix)
	}

	parts := []string{}
	if id != "" {
		parts = strings.Split(id, "/")
	}

	segments := make([]pathSegment, 0, len(parts))
	patternParts := make([]string, 0, len(parts))
	normalizedIDParts := make([]string, 0, len(parts))
	staticCount := 0

	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			return appRoute{}, fmt.Errorf("route file %q has empty path segment", relPath)
		}

		name, isParam, normalizedIDPart, err := parseWildcardSegment(part)
		if err != nil {
			return appRoute{}, fmt.Errorf("route file %q: %w", relPath, err)
		}

		if isParam {
			segments = append(segments, pathSegment{name: name, isParam: true})
			patternParts = append(patternParts, ":")
			normalizedIDParts = append(normalizedIDParts, normalizedIDPart)
			continue
		}

		segments = append(segments, pathSegment{name: part, isParam: false})
		patternParts = append(patternParts, part)
		normalizedIDParts = append(normalizedIDParts, part)
		staticCount++
	}

	normalizedID := strings.Join(normalizedIDParts, "/")

	patternKey := "/"
	if len(patternParts) > 0 {
		patternKey = "/" + strings.Join(patternParts, "/")
	}

	return appRoute{
		id:          normalizedID,
		segments:    segments,
		staticCount: staticCount,
		patternKey:  patternKey,
	}, nil
}

func parseWildcardSegment(segment string) (string, bool, string, error) {
	if strings.HasPrefix(segment, "_") {
		name := strings.TrimSpace(strings.TrimPrefix(segment, "_"))
		if !dynamicSegmentNamePattern.MatchString(name) {
			return "", false, "", fmt.Errorf("invalid wildcard name %q", name)
		}
		return name, true, "[" + name + "]", nil
	}

	if strings.HasPrefix(segment, "[") || strings.HasSuffix(segment, "]") {
		if !strings.HasPrefix(segment, "[") || !strings.HasSuffix(segment, "]") {
			return "", false, "", fmt.Errorf("invalid wildcard segment %q", segment)
		}

		name := strings.TrimSpace(segment[1 : len(segment)-1])
		if !dynamicSegmentNamePattern.MatchString(name) {
			return "", false, "", fmt.Errorf("invalid wildcard name %q", name)
		}

		return name, true, "[" + name + "]", nil
	}

	if strings.ContainsAny(segment, "[]") {
		return "", false, "", fmt.Errorf("invalid static segment %q", segment)
	}
	if strings.TrimSpace(segment) == "_" {
		return "", false, "", fmt.Errorf("invalid wildcard segment %q", segment)
	}

	return "", false, "", nil
}

func (router *AppRouter) Match(requestPath string) (AppRouteMatch, bool) {
	requestSegments := splitPathSegments(requestPath)

	for _, route := range router.routes {
		if len(route.segments) != len(requestSegments) {
			continue
		}

		params := make(map[string]string, 2)
		matched := true

		for idx, segment := range route.segments {
			requestValue := requestSegments[idx]
			if segment.isParam {
				params[segment.name] = requestValue
				continue
			}
			if segment.name != requestValue {
				matched = false
				break
			}
		}

		if !matched {
			continue
		}

		if len(params) == 0 {
			return AppRouteMatch{ID: route.id}, true
		}
		return AppRouteMatch{ID: route.id, Params: params}, true
	}

	return AppRouteMatch{}, false
}

func matchPathPattern(pattern string, requestPath string) (map[string]string, bool) {
	patternSegments := splitPathSegments(pattern)
	requestSegments := splitPathSegments(requestPath)
	if len(patternSegments) != len(requestSegments) {
		return nil, false
	}

	params := make(map[string]string, 2)
	for idx, patternSegment := range patternSegments {
		name, isParam, _, err := parseWildcardSegment(patternSegment)
		if err != nil {
			return nil, false
		}

		requestSegment := requestSegments[idx]
		if !isParam {
			if patternSegment != requestSegment {
				return nil, false
			}
			continue
		}

		params[name] = requestSegment
	}

	return params, true
}

func splitPathSegments(raw string) []string {
	cleaned := path.Clean("/" + strings.TrimSpace(raw))
	if cleaned == "/" {
		return []string{}
	}

	trimmed := strings.Trim(cleaned, "/")
	if trimmed == "" {
		return []string{}
	}

	return strings.Split(trimmed, "/")
}
