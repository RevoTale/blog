package main

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var dynamicSegmentNamePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)

type templateKind string

const (
	pageTemplate   templateKind = "page"
	layoutTemplate templateKind = "layout"
)

type routeSegment struct {
	StaticName string
	ParamName  string
}

func (s routeSegment) IsParam() bool {
	return s.ParamName != ""
}

func (s routeSegment) RoutePart() string {
	if s.IsParam() {
		return "[" + s.ParamName + "]"
	}
	return s.StaticName
}

func (s routeSegment) SafePart() string {
	if s.IsParam() {
		return "param_" + strings.ToLower(s.ParamName)
	}
	static := strings.ToLower(s.StaticName)
	static = strings.ReplaceAll(static, "-", "_")
	static = strings.ReplaceAll(static, ".", "_")
	static = strings.ReplaceAll(static, " ", "_")
	if static == "" {
		return "segment"
	}
	return static
}

type templateDef struct {
	Kind       templateKind
	RouteID    string
	SourcePath string
	Segments   []routeSegment
	ModuleName string
	Package    string
	OutputDir  string
	OutputFile string
}

type liveConfig struct {
	StateType       string
	ParseStateFn    string
	LoaderFn        string
	SelectorID      string
	BadRequestError string
}

type routeBinding struct {
	ParamsType string
	ViewType   string
	LoadFn     string
	Live       *liveConfig
}

var bindings = map[string]routeBinding{
	"notes": {
		ParamsType: "framework.EmptyParams",
		ViewType:   "appcore.NotesPageView",
		LoadFn:     "appcore.LoadNotesPage",
		Live: &liveConfig{
			StateType:       "appcore.NotesSignalState",
			ParseStateFn:    "appcore.ParseNotesLiveState",
			LoaderFn:        "appcore.LoadNotesLivePage",
			SelectorID:      "notes-content",
			BadRequestError: "invalid datastar signal payload",
		},
	},
	"note/[slug]": {
		ParamsType: "framework.SlugParams",
		ViewType:   "appcore.NotePageView",
		LoadFn:     "appcore.LoadNotePage",
	},
	"author/[slug]": {
		ParamsType: "framework.SlugParams",
		ViewType:   "appcore.AuthorPageView",
		LoadFn:     "appcore.LoadAuthorPage",
		Live: &liveConfig{
			StateType:       "appcore.AuthorSignalState",
			ParseStateFn:    "appcore.ParseAuthorLiveState",
			LoaderFn:        "appcore.LoadAuthorLivePage",
			SelectorID:      "author-content",
			BadRequestError: "invalid datastar signal payload",
		},
	},
}

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "approutegen: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	appRootRel, outputRootRel, importRoot, err := resolvePaths()
	if err != nil {
		return err
	}

	templates, err := discoverTemplates(appRootRel, outputRootRel)
	if err != nil {
		return err
	}

	pages := make([]templateDef, 0, len(templates))
	layoutsByRoute := make(map[string]templateDef)
	for _, tpl := range templates {
		switch tpl.Kind {
		case pageTemplate:
			pages = append(pages, tpl)
		case layoutTemplate:
			layoutsByRoute[tpl.RouteID] = tpl
		default:
			return fmt.Errorf("unexpected template kind %q", tpl.Kind)
		}
	}

	if len(pages) == 0 {
		return errors.New("no page.templ files found in internal/web/app")
	}

	sort.Slice(pages, func(i int, j int) bool {
		return pages[i].RouteID < pages[j].RouteID
	})

	if err := validateBindings(pages); err != nil {
		return err
	}

	if err := os.RemoveAll(outputRootRel); err != nil {
		return fmt.Errorf("clear routes_gen: %w", err)
	}
	if err := os.MkdirAll(outputRootRel, 0o755); err != nil {
		return fmt.Errorf("create routes_gen root: %w", err)
	}

	for _, tpl := range templates {
		if err := writeTemplCopy(tpl); err != nil {
			return err
		}
	}

	registry, err := generateRegistry(importRoot, pages, layoutsByRoute)
	if err != nil {
		return err
	}

	registryPath := filepath.Join(outputRootRel, "registry_gen.go")
	if err := os.WriteFile(registryPath, registry, 0o644); err != nil {
		return fmt.Errorf("write registry %q: %w", registryPath, err)
	}

	return nil
}

func resolvePaths() (string, string, string, error) {
	switch {
	case pathExists("internal/web/app"):
		return "internal/web/app", "internal/web/routes_gen", "internal/web/routes_gen", nil
	case pathExists("app"):
		return "app", "routes_gen", "internal/web/routes_gen", nil
	default:
		return "", "", "", errors.New("cannot resolve app directory; expected app or internal/web/app")
	}
}

func pathExists(target string) bool {
	_, err := os.Stat(target)
	return err == nil
}

func validateBindings(pages []templateDef) error {
	seen := make(map[string]struct{}, len(pages))
	for _, page := range pages {
		if _, ok := bindings[page.RouteID]; !ok {
			return fmt.Errorf("route %q has no binding config in approutegen", page.RouteID)
		}
		seen[page.RouteID] = struct{}{}
	}

	for routeID := range bindings {
		if _, ok := seen[routeID]; ok {
			continue
		}
		return fmt.Errorf("binding config for route %q has no matching app page.templ", routeID)
	}

	return nil
}

func discoverTemplates(appRoot string, outputRoot string) ([]templateDef, error) {
	templates := make([]templateDef, 0, 8)

	walkErr := filepath.WalkDir(appRoot, func(filePath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}

		base := path.Base(filepath.ToSlash(filePath))
		var kind templateKind
		switch base {
		case "page.templ":
			kind = pageTemplate
		case "layout.templ":
			kind = layoutTemplate
		default:
			return nil
		}

		relPath, relErr := filepath.Rel(appRoot, filePath)
		if relErr != nil {
			return fmt.Errorf("resolve relative path for %q: %w", filePath, relErr)
		}
		relPath = filepath.ToSlash(relPath)

		routeDir := path.Dir(relPath)
		if routeDir == "." {
			routeDir = ""
		}

		segments, parseErr := parseRouteSegments(routeDir)
		if parseErr != nil {
			return fmt.Errorf("parse route in %q: %w", relPath, parseErr)
		}

		routeID := routeIDFromSegments(segments)
		moduleName := moduleNameFor(kind, segments)
		templates = append(templates, templateDef{
			Kind:       kind,
			RouteID:    routeID,
			SourcePath: filepath.ToSlash(filePath),
			Segments:   segments,
			ModuleName: moduleName,
			Package:    moduleName,
			OutputDir:  filepath.ToSlash(filepath.Join(outputRoot, moduleName)),
			OutputFile: string(kind) + ".templ",
		})

		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("walk app templates: %w", walkErr)
	}

	sort.Slice(templates, func(i int, j int) bool {
		left := templates[i]
		right := templates[j]
		if left.RouteID != right.RouteID {
			return left.RouteID < right.RouteID
		}
		return left.Kind < right.Kind
	})

	return templates, nil
}

func parseRouteSegments(routeDir string) ([]routeSegment, error) {
	if strings.TrimSpace(routeDir) == "" {
		return []routeSegment{}, nil
	}

	parts := strings.Split(routeDir, "/")
	segments := make([]routeSegment, 0, len(parts))
	for _, part := range parts {
		segment, err := parseRouteSegment(part)
		if err != nil {
			return nil, err
		}
		segments = append(segments, segment)
	}

	return segments, nil
}

func parseRouteSegment(part string) (routeSegment, error) {
	trimmed := strings.TrimSpace(part)
	if trimmed == "" {
		return routeSegment{}, errors.New("route segment cannot be empty")
	}

	if strings.HasPrefix(trimmed, "[") || strings.HasSuffix(trimmed, "]") {
		if !strings.HasPrefix(trimmed, "[") || !strings.HasSuffix(trimmed, "]") {
			return routeSegment{}, fmt.Errorf("invalid wildcard segment %q", part)
		}
		name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(trimmed, "["), "]"))
		if !dynamicSegmentNamePattern.MatchString(name) {
			return routeSegment{}, fmt.Errorf("invalid wildcard name %q", name)
		}
		return routeSegment{ParamName: name}, nil
	}

	if strings.HasPrefix(trimmed, "_") {
		name := strings.TrimSpace(strings.TrimPrefix(trimmed, "_"))
		if !dynamicSegmentNamePattern.MatchString(name) {
			return routeSegment{}, fmt.Errorf("invalid wildcard name %q", name)
		}
		return routeSegment{ParamName: name}, nil
	}

	if strings.ContainsAny(trimmed, "[]") {
		return routeSegment{}, fmt.Errorf("invalid static segment %q", part)
	}

	return routeSegment{StaticName: trimmed}, nil
}

func routeIDFromSegments(segments []routeSegment) string {
	if len(segments) == 0 {
		return ""
	}
	parts := make([]string, 0, len(segments))
	for _, segment := range segments {
		parts = append(parts, segment.RoutePart())
	}
	return strings.Join(parts, "/")
}

func moduleNameFor(kind templateKind, segments []routeSegment) string {
	parts := make([]string, 0, len(segments)+2)
	parts = append(parts, "r", string(kind))
	if len(segments) == 0 {
		parts = append(parts, "root")
	} else {
		for _, segment := range segments {
			parts = append(parts, segment.SafePart())
		}
	}
	return strings.Join(parts, "_")
}

func writeTemplCopy(tpl templateDef) error {
	source, err := os.ReadFile(tpl.SourcePath)
	if err != nil {
		return fmt.Errorf("read %q: %w", tpl.SourcePath, err)
	}

	rewritten, err := rewritePackageDeclaration(source, tpl.Package)
	if err != nil {
		return fmt.Errorf("rewrite package for %q: %w", tpl.SourcePath, err)
	}

	if err := os.MkdirAll(tpl.OutputDir, 0o755); err != nil {
		return fmt.Errorf("create output dir %q: %w", tpl.OutputDir, err)
	}

	target := filepath.Join(tpl.OutputDir, tpl.OutputFile)
	if err := os.WriteFile(target, rewritten, 0o644); err != nil {
		return fmt.Errorf("write generated template %q: %w", target, err)
	}

	return nil
}

func rewritePackageDeclaration(source []byte, packageName string) ([]byte, error) {
	lines := strings.Split(string(source), "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "package ") {
			continue
		}

		lines[i] = "package " + packageName
		return []byte(strings.Join(lines, "\n")), nil
	}

	return nil, errors.New("template missing package declaration")
}

func generateRegistry(importRoot string, pages []templateDef, layouts map[string]templateDef) ([]byte, error) {
	layoutImportSet := make(map[string]templateDef)
	for _, layout := range layouts {
		layoutImportSet[layout.ModuleName] = layout
	}

	pageImports := make(map[string]templateDef)
	for _, page := range pages {
		pageImports[page.ModuleName] = page
	}

	importLines := []string{
		"\"strings\"",
		"\"blog/framework\"",
		"\"blog/framework/router\"",
		"\"blog/internal/web/appcore\"",
		"\"github.com/a-h/templ\"",
	}

	moduleImports := make([]string, 0, len(pageImports)+len(layoutImportSet))
	for _, page := range pages {
		moduleImports = append(moduleImports, fmt.Sprintf(
			"%s \"blog/%s/%s\"",
			page.ModuleName,
			filepath.ToSlash(importRoot),
			page.ModuleName,
		))
	}
	layoutKeys := make([]string, 0, len(layoutImportSet))
	for key := range layoutImportSet {
		layoutKeys = append(layoutKeys, key)
	}
	sort.Strings(layoutKeys)
	for _, key := range layoutKeys {
		if _, ok := pageImports[key]; ok {
			continue
		}
		moduleImports = append(moduleImports, fmt.Sprintf(
			"%s \"blog/%s/%s\"",
			key,
			filepath.ToSlash(importRoot),
			key,
		))
	}
	sort.Strings(moduleImports)
	importLines = append(importLines, moduleImports...)

	buffer := &bytes.Buffer{}
	buffer.WriteString("// Code generated by cmd/approutegen. DO NOT EDIT.\n")
	buffer.WriteString("package routes_gen\n\n")
	buffer.WriteString("import (\n")
	for _, line := range importLines {
		buffer.WriteString("\t" + line + "\n")
	}
	buffer.WriteString(")\n\n")

	buffer.WriteString("func Handlers() []framework.RouteHandler[*appcore.Context] {\n")
	buffer.WriteString("\treturn []framework.RouteHandler[*appcore.Context]{\n")
	for _, page := range pages {
		binding := bindings[page.RouteID]
		if binding.Live == nil {
			writef(
				buffer,
				"\t\tframework.PageOnlyRouteHandler[*appcore.Context, %s, %s]{\n",
				binding.ParamsType,
				binding.ViewType,
			)
			writePageModule(buffer, page, binding, layouts)
			buffer.WriteString("\t\t},\n")
			continue
		}

		writef(
			buffer,
			"\t\tframework.PageAndLiveRouteHandler[*appcore.Context, %s, %s, %s]{\n",
			binding.ParamsType,
			binding.ViewType,
			binding.Live.StateType,
		)
		writePageModule(buffer, page, binding, layouts)
		writeLiveModule(buffer, page, binding)
		buffer.WriteString("\t\t},\n")
	}
	buffer.WriteString("\t}\n")
	buffer.WriteString("}\n\n")

	parserFuncs := make(map[string]string)
	for _, page := range pages {
		parserName := parserFuncName(page.RouteID)
		if _, ok := parserFuncs[parserName]; !ok {
			parserFuncs[parserName] = page.RouteID
		}

		binding := bindings[page.RouteID]
		if binding.Live != nil {
			liveRouteID := page.RouteID + "/live"
			liveParserName := parserFuncName(liveRouteID)
			if _, ok := parserFuncs[liveParserName]; !ok {
				parserFuncs[liveParserName] = liveRouteID
			}
		}
	}

	parserNames := make([]string, 0, len(parserFuncs))
	for name := range parserFuncs {
		parserNames = append(parserNames, name)
	}
	sort.Strings(parserNames)
	for _, parserName := range parserNames {
		routeID := parserFuncs[parserName]
		writeParserFunc(buffer, parserName, routeID)
	}

	wrapperNames := make([]string, 0)
	for _, page := range pages {
		binding := bindings[page.RouteID]
		wrappers := layoutWrappersFor(page, binding, layouts)
		wrapperNames = append(wrapperNames, wrappers...)
	}
	wrapperNames = dedupeSorted(wrapperNames)
	for _, wrapperName := range wrapperNames {
		writeWrapperFunc(buffer, wrapperName, pages, layouts)
	}

	formatted, err := format.Source(buffer.Bytes())
	if err != nil {
		return nil, fmt.Errorf("format registry source: %w", err)
	}
	return formatted, nil
}

func writePageModule(buffer *bytes.Buffer, page templateDef, binding routeBinding, layouts map[string]templateDef) {
	parserName := parserFuncName(page.RouteID)
	writef(
		buffer,
		"\t\t\tPage: framework.PageModule[*appcore.Context, %s, %s]{\n",
		binding.ParamsType,
		binding.ViewType,
	)
	writef(buffer, "\t\t\t\tPattern:     \"/%s\",\n", page.RouteID)
	writef(buffer, "\t\t\t\tParseParams: %s,\n", parserName)
	writef(buffer, "\t\t\t\tLoad:        %s,\n", binding.LoadFn)
	writef(buffer, "\t\t\t\tRender:      %s.Page,\n", page.ModuleName)

	wrappers := layoutWrappersFor(page, binding, layouts)
	if len(wrappers) == 0 {
		writef(buffer, "\t\t\t\tLayouts:     []framework.LayoutRenderer[%s]{},\n", binding.ViewType)
	} else {
		writef(buffer, "\t\t\t\tLayouts: []framework.LayoutRenderer[%s]{\n", binding.ViewType)
		for _, wrapper := range wrappers {
			writef(buffer, "\t\t\t\t\t%s,\n", wrapper)
		}
		buffer.WriteString("\t\t\t\t},\n")
	}
	buffer.WriteString("\t\t\t},\n")
}

func writeLiveModule(buffer *bytes.Buffer, page templateDef, binding routeBinding) {
	if binding.Live == nil {
		return
	}
	writef(
		buffer,
		"\t\t\tLive: framework.LiveModule[*appcore.Context, %s, %s, %s]{\n",
		binding.ParamsType,
		binding.ViewType,
		binding.Live.StateType,
	)
	writef(buffer, "\t\t\t\tPattern:           \"/%s/live\",\n", page.RouteID)
	writef(
		buffer,
		"\t\t\t\tParseParams:       %s,\n",
		parserFuncName(page.RouteID+"/live"),
	)
	writef(buffer, "\t\t\t\tParseState:        %s,\n", binding.Live.ParseStateFn)
	writef(buffer, "\t\t\t\tLoad:              %s,\n", binding.Live.LoaderFn)
	writef(buffer, "\t\t\t\tRender:            %s.Page,\n", page.ModuleName)
	writef(buffer, "\t\t\t\tSelectorID:        \"%s\",\n", binding.Live.SelectorID)
	writef(buffer, "\t\t\t\tBadRequestMessage: \"%s\",\n", binding.Live.BadRequestError)
	buffer.WriteString("\t\t\t},\n")
}

func parserFuncName(routeID string) string {
	if routeID == "" {
		return "parseRootParams"
	}

	parts := strings.Split(routeID, "/")
	builder := strings.Builder{}
	builder.WriteString("parse")
	for _, part := range parts {
		if strings.HasPrefix(part, "[") && strings.HasSuffix(part, "]") {
			part = strings.TrimSuffix(strings.TrimPrefix(part, "["), "]")
		}
		part = strings.ReplaceAll(part, "-", " ")
		part = strings.ReplaceAll(part, "_", " ")
		tokens := strings.Fields(part)
		if len(tokens) == 0 {
			continue
		}
		for _, token := range tokens {
			builder.WriteString(strings.ToUpper(token[:1]))
			if len(token) > 1 {
				builder.WriteString(token[1:])
			}
		}
	}
	builder.WriteString("Params")
	return builder.String()
}

func writeParserFunc(buffer *bytes.Buffer, parserName string, routeID string) {
	pattern := "/" + routeID
	if routeID == "" {
		pattern = "/"
	}

	if strings.Contains(routeID, "[slug]") {
		writef(buffer, "func %s(requestPath string) (framework.SlugParams, bool) {\n", parserName)
		writef(buffer, "\tparams, ok := router.MatchPathPattern(\"%s\", requestPath)\n", pattern)
		buffer.WriteString("\tif !ok {\n")
		buffer.WriteString("\t\treturn framework.SlugParams{}, false\n")
		buffer.WriteString("\t}\n")
		buffer.WriteString("\tslug := strings.TrimSpace(params[\"slug\"])\n")
		buffer.WriteString("\tif !router.IsValidSlug(slug) {\n")
		buffer.WriteString("\t\treturn framework.SlugParams{}, false\n")
		buffer.WriteString("\t}\n")
		buffer.WriteString("\treturn framework.SlugParams{Slug: slug}, true\n")
		buffer.WriteString("}\n\n")
		return
	}

	writef(buffer, "func %s(requestPath string) (framework.EmptyParams, bool) {\n", parserName)
	writef(buffer, "\t_, ok := router.MatchPathPattern(\"%s\", requestPath)\n", pattern)
	buffer.WriteString("\tif !ok {\n")
	buffer.WriteString("\t\treturn framework.EmptyParams{}, false\n")
	buffer.WriteString("\t}\n")
	buffer.WriteString("\treturn framework.EmptyParams{}, true\n")
	buffer.WriteString("}\n\n")
}

func layoutWrappersFor(page templateDef, binding routeBinding, layouts map[string]templateDef) []string {
	chain := layoutChain(page.RouteID, layouts)
	wrappers := make([]string, 0, len(chain))
	for _, layout := range chain {
		wrappers = append(wrappers, wrapperFuncName(page, binding, layout))
	}
	return wrappers
}

func layoutChain(routeID string, layouts map[string]templateDef) []templateDef {
	segments := []string{}
	if routeID != "" {
		segments = strings.Split(routeID, "/")
	}

	candidates := make([]string, 0, len(segments)+1)
	candidates = append(candidates, "")
	for idx := 1; idx <= len(segments); idx++ {
		candidates = append(candidates, strings.Join(segments[:idx], "/"))
	}

	chain := make([]templateDef, 0, len(candidates))
	for _, candidate := range candidates {
		layout, ok := layouts[candidate]
		if !ok {
			continue
		}
		chain = append(chain, layout)
	}
	return chain
}

func wrapperFuncName(page templateDef, binding routeBinding, layout templateDef) string {
	base := parserFuncName(page.RouteID)
	base = strings.TrimSuffix(strings.TrimPrefix(base, "parse"), "Params")
	layoutKey := parserFuncName(layout.RouteID)
	layoutKey = strings.TrimSuffix(strings.TrimPrefix(layoutKey, "parse"), "Params")
	return fmt.Sprintf("wrap%sWith%sLayout", base, layoutKey)
}

func writeWrapperFunc(buffer *bytes.Buffer, wrapperName string, pages []templateDef, layouts map[string]templateDef) {
	for _, page := range pages {
		binding := bindings[page.RouteID]
		chain := layoutChain(page.RouteID, layouts)
		for _, layout := range chain {
			if wrapperFuncName(page, binding, layout) != wrapperName {
				continue
			}
			writef(buffer, "func %s(view %s, child templ.Component) templ.Component {\n", wrapperName, binding.ViewType)
			writef(buffer, "\treturn %s.Layout(view, child)\n", layout.ModuleName)
			buffer.WriteString("}\n\n")
			return
		}
	}
}

func writef(buffer *bytes.Buffer, pattern string, args ...interface{}) {
	_, _ = fmt.Fprintf(buffer, pattern, args...)
}

func dedupeSorted(values []string) []string {
	sort.Strings(values)
	if len(values) == 0 {
		return values
	}

	out := make([]string, 0, len(values))
	previous := ""
	for idx, value := range values {
		if idx > 0 && value == previous {
			continue
		}
		out = append(out, value)
		previous = value
	}

	return out
}
