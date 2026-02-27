package approutegen

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var dynamicSegmentNamePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)
var liveContainerPattern = regexp.MustCompile(
	`(?s)<[^>]*\bid\s*=\s*"([A-Za-z0-9_-]+)"[^>]*\bdata-signals\b` +
		`|<[^>]*\bdata-signals\b[^>]*\bid\s*=\s*"([A-Za-z0-9_-]+)"`,
)

type templateKind string

const (
	pageTemplate   templateKind = "page"
	layoutTemplate templateKind = "layout"
)

const (
	defaultLiveBadRequestMessage = "invalid datastar signal payload"
	typesFileName                = "types.go"
	resolverFileName             = "resolver.go"
)

type generationPaths struct {
	AppRoot            string
	GenRoot            string
	GenImportRoot      string
	ResolverRoot       string
	ResolverImportRoot string
}

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
	return safeIdentifier(s.StaticName)
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

type componentDef struct {
	SourcePath string
	ModuleName string
	Package    string
	OutputDir  string
	OutputFile string
}

type resolverTypeDecl struct {
	PackageName  string
	HasLiveState bool
}

type routeParamDef struct {
	Name      string
	FieldName string
}

type routeMeta struct {
	RouteID            string
	Segments           []routeSegment
	RouteName          string
	ParamsTypeName     string
	Params             []routeParamDef
	Page               templateDef
	ResolverDir        string
	ResolverImportPath string
	ResolverAlias      string
	ResolverPackage    string
	ResolverField      string
	HasLive            bool
	LiveSelectorID     string
}

type routeFiles struct {
	Templates []templateDef
	Pages     []templateDef
	Layouts   map[string]templateDef
}

func Run() error {
	paths, err := resolvePaths()
	if err != nil {
		return err
	}

	routes, err := discoverRouteFiles(paths.AppRoot, paths.GenRoot)
	if err != nil {
		return err
	}
	if len(routes.Pages) == 0 {
		return errors.New("no page.templ files found in internal/web/app")
	}

	components, err := discoverSharedComponents(paths.AppRoot, paths.GenRoot)
	if err != nil {
		return err
	}

	metas, err := buildRouteMetas(routes.Pages, paths)
	if err != nil {
		return err
	}

	if err := os.RemoveAll(paths.GenRoot); err != nil {
		return fmt.Errorf("clear generated output: %w", err)
	}
	if err := os.MkdirAll(paths.GenRoot, 0o755); err != nil {
		return fmt.Errorf("create generated output root: %w", err)
	}

	for _, tpl := range routes.Templates {
		if err := writeTemplCopy(tpl); err != nil {
			return err
		}
	}
	for _, component := range components {
		tpl := templateDef{
			SourcePath: component.SourcePath,
			Package:    component.Package,
			OutputDir:  component.OutputDir,
			OutputFile: component.OutputFile,
		}
		if err := writeTemplCopy(tpl); err != nil {
			return err
		}
	}

	contractsSource, err := generateContractsSource(metas)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(paths.GenRoot, "contracts_gen.go"), contractsSource, 0o644); err != nil {
		return fmt.Errorf("write contracts_gen.go: %w", err)
	}

	registrySource, err := generateRegistrySource(paths, metas, routes.Layouts)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(paths.GenRoot, "registry_gen.go"), registrySource, 0o644); err != nil {
		return fmt.Errorf("write registry_gen.go: %w", err)
	}

	for _, meta := range metas {
		if err := os.MkdirAll(meta.ResolverDir, 0o755); err != nil {
			return fmt.Errorf("create resolver dir %q: %w", meta.ResolverDir, err)
		}

		if err := ensureRouteResolverStub(meta); err != nil {
			return err
		}
	}

	resolverAdapterSource, err := generateResolversSource(metas)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(paths.GenRoot, "resolvers_gen.go"), resolverAdapterSource, 0o644); err != nil {
		return fmt.Errorf("write resolvers_gen.go: %w", err)
	}

	return nil
}

func resolvePaths() (generationPaths, error) {
	moduleRoot, err := findModuleRoot()
	if err != nil {
		return generationPaths{}, err
	}

	return generationPaths{
		AppRoot:            filepath.ToSlash(filepath.Join(moduleRoot, "internal/web/app")),
		GenRoot:            filepath.ToSlash(filepath.Join(moduleRoot, "internal/web/gen")),
		GenImportRoot:      "internal/web/gen",
		ResolverRoot:       filepath.ToSlash(filepath.Join(moduleRoot, "internal/web/appcore/resolvers")),
		ResolverImportRoot: "internal/web/appcore/resolvers",
	}, nil
}

func findModuleRoot() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}

	for {
		if pathExists(filepath.Join(currentDir, "internal/web/app")) {
			return currentDir, nil
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			break
		}
		currentDir = parentDir
	}

	return "", errors.New("strict app root missing: expected internal/web/app")
}

func pathExists(target string) bool {
	_, err := os.Stat(target)
	return err == nil
}

func discoverRouteFiles(appRoot string, outputRoot string) (routeFiles, error) {
	templates := make([]templateDef, 0, 16)
	pages := make([]templateDef, 0, 8)
	layouts := make(map[string]templateDef)

	walkErr := filepath.WalkDir(appRoot, func(filePath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}

		relPath, relErr := filepath.Rel(appRoot, filePath)
		if relErr != nil {
			return fmt.Errorf("resolve relative path for %q: %w", filePath, relErr)
		}
		relPath = filepath.ToSlash(relPath)

		if !strings.HasSuffix(relPath, ".templ") {
			return nil
		}

		if strings.HasPrefix(relPath, "components/") {
			return nil
		}

		if strings.Contains(relPath, "/components/") {
			return fmt.Errorf("component templates must be under app/components only: %q", relPath)
		}

		base := path.Base(relPath)
		var kind templateKind
		switch base {
		case "page.templ":
			kind = pageTemplate
		case "layout.templ":
			kind = layoutTemplate
		default:
			return fmt.Errorf("unsupported route template %q; only page.templ/layout.templ are allowed", relPath)
		}

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
		tpl := templateDef{
			Kind:       kind,
			RouteID:    routeID,
			SourcePath: filepath.ToSlash(filePath),
			Segments:   segments,
			ModuleName: moduleName,
			Package:    moduleName,
			OutputDir:  filepath.ToSlash(filepath.Join(outputRoot, moduleName)),
			OutputFile: string(kind) + ".templ",
		}
		templates = append(templates, tpl)
		if kind == pageTemplate {
			pages = append(pages, tpl)
		}
		if kind == layoutTemplate {
			layouts[routeID] = tpl
		}

		return nil
	})
	if walkErr != nil {
		return routeFiles{}, fmt.Errorf("walk app templates: %w", walkErr)
	}

	sort.Slice(templates, func(i int, j int) bool {
		left := templates[i]
		right := templates[j]
		if left.RouteID != right.RouteID {
			return left.RouteID < right.RouteID
		}
		return left.Kind < right.Kind
	})
	sort.Slice(pages, func(i int, j int) bool {
		return pages[i].RouteID < pages[j].RouteID
	})

	return routeFiles{Templates: templates, Pages: pages, Layouts: layouts}, nil
}

func discoverSharedComponents(appRoot string, outputRoot string) ([]componentDef, error) {
	componentsRoot := filepath.Join(appRoot, "components")
	if !pathExists(componentsRoot) {
		return []componentDef{}, nil
	}

	components := make([]componentDef, 0, 4)
	seenModules := make(map[string]string)

	walkErr := filepath.WalkDir(componentsRoot, func(filePath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if filePath == componentsRoot {
				return nil
			}
			return fmt.Errorf("nested component directories are not allowed: %q", filepath.ToSlash(filePath))
		}

		if filepath.Ext(filePath) != ".templ" {
			return nil
		}

		base := path.Base(filepath.ToSlash(filePath))
		componentName := strings.TrimSuffix(base, ".templ")
		if strings.TrimSpace(componentName) == "" {
			return fmt.Errorf("invalid component filename %q", base)
		}

		moduleName := "c_" + safeIdentifier(componentName)
		if existing, ok := seenModules[moduleName]; ok {
			return fmt.Errorf("component module name conflict %q between %q and %q", moduleName, existing, filePath)
		}
		seenModules[moduleName] = filePath

		components = append(components, componentDef{
			SourcePath: filepath.ToSlash(filePath),
			ModuleName: moduleName,
			Package:    moduleName,
			OutputDir:  filepath.ToSlash(filepath.Join(outputRoot, moduleName)),
			OutputFile: base,
		})

		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("walk shared components: %w", walkErr)
	}

	sort.Slice(components, func(i int, j int) bool {
		return components[i].SourcePath < components[j].SourcePath
	})

	return components, nil
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
		return routeSegment{}, fmt.Errorf(
			"legacy wildcard segment %q is not allowed; use [param] directories",
			part,
		)
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

func buildRouteMetas(pages []templateDef, paths generationPaths) ([]routeMeta, error) {
	metas := make([]routeMeta, 0, len(pages))

	for _, page := range pages {
		resolverRel := resolverRelativePath(page.Segments)
		resolverDir := filepath.Join(paths.ResolverRoot, filepath.FromSlash(resolverRel))
		typeDecl, err := readResolverTypes(filepath.Join(resolverDir, typesFileName))
		if err != nil {
			return nil, fmt.Errorf("route %q: %w", page.RouteID, err)
		}

		params, err := routeParamsFromSegments(page.RouteID, page.Segments)
		if err != nil {
			return nil, err
		}

		routeName := routeNameFromSegments(page.Segments)
		meta := routeMeta{
			RouteID:            page.RouteID,
			Segments:           page.Segments,
			RouteName:          routeName,
			ParamsTypeName:     routeName + "Params",
			Params:             params,
			Page:               page,
			ResolverDir:        filepath.ToSlash(resolverDir),
			ResolverImportPath: filepath.ToSlash(path.Join(paths.ResolverImportRoot, resolverRel)),
			ResolverAlias:      "rr_" + routeSafeKey(page.Segments),
			ResolverPackage:    typeDecl.PackageName,
			ResolverField:      "r" + routeName,
			HasLive:            typeDecl.HasLiveState,
		}
		if meta.HasLive {
			selectorID, selectorErr := extractLiveSelectorID(page.SourcePath)
			if selectorErr != nil {
				return nil, fmt.Errorf("route %q: %w", page.RouteID, selectorErr)
			}
			meta.LiveSelectorID = selectorID
		}

		metas = append(metas, meta)
	}

	sort.Slice(metas, func(i int, j int) bool {
		return metas[i].RouteID < metas[j].RouteID
	})

	return metas, nil
}

func resolverRelativePath(segments []routeSegment) string {
	if len(segments) == 0 {
		return "root"
	}

	parts := make([]string, 0, len(segments))
	for _, segment := range segments {
		if segment.IsParam() {
			parts = append(parts, "param_"+strings.ToLower(segment.ParamName))
			continue
		}
		parts = append(parts, segment.StaticName)
	}
	return path.Join(parts...)
}

func routeSafeKey(segments []routeSegment) string {
	if len(segments) == 0 {
		return "root"
	}
	parts := make([]string, 0, len(segments))
	for _, segment := range segments {
		parts = append(parts, segment.SafePart())
	}
	return strings.Join(parts, "_")
}

func routeNameFromSegments(segments []routeSegment) string {
	if len(segments) == 0 {
		return "Root"
	}

	builder := strings.Builder{}
	for _, segment := range segments {
		if segment.IsParam() {
			builder.WriteString("Param")
			builder.WriteString(pascalToken(segment.ParamName))
			continue
		}
		builder.WriteString(pascalToken(segment.StaticName))
	}

	name := builder.String()
	if name == "" {
		return "Root"
	}
	return name
}

func routeParamsFromSegments(routeID string, segments []routeSegment) ([]routeParamDef, error) {
	params := make([]routeParamDef, 0, len(segments))
	seen := make(map[string]struct{})

	for _, segment := range segments {
		if !segment.IsParam() {
			continue
		}

		fieldName := pascalToken(segment.ParamName)
		if fieldName == "" {
			return nil, fmt.Errorf("route %q has invalid param name %q", routeID, segment.ParamName)
		}
		if _, ok := seen[fieldName]; ok {
			return nil, fmt.Errorf("route %q has duplicate param field %q", routeID, fieldName)
		}
		seen[fieldName] = struct{}{}

		params = append(params, routeParamDef{
			Name:      segment.ParamName,
			FieldName: fieldName,
		})
	}

	return params, nil
}

func readResolverTypes(typesPath string) (resolverTypeDecl, error) {
	if !pathExists(typesPath) {
		return resolverTypeDecl{}, fmt.Errorf("required resolver type file missing: %q", filepath.ToSlash(typesPath))
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, typesPath, nil, parser.SkipObjectResolution)
	if err != nil {
		return resolverTypeDecl{}, fmt.Errorf("parse %q: %w", filepath.ToSlash(typesPath), err)
	}

	pkgName := strings.TrimSpace(file.Name.Name)
	if pkgName == "" {
		return resolverTypeDecl{}, fmt.Errorf("%q has invalid package name", filepath.ToSlash(typesPath))
	}

	foundPageView := false
	foundLiveState := false
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			switch strings.TrimSpace(typeSpec.Name.Name) {
			case "PageView":
				foundPageView = true
			case "LiveState":
				foundLiveState = true
			}
		}
	}

	if !foundPageView {
		return resolverTypeDecl{}, fmt.Errorf("%q must declare type PageView", filepath.ToSlash(typesPath))
	}

	return resolverTypeDecl{PackageName: pkgName, HasLiveState: foundLiveState}, nil
}

func extractLiveSelectorID(pageTemplatePath string) (string, error) {
	source, err := os.ReadFile(pageTemplatePath)
	if err != nil {
		return "", fmt.Errorf("read %q: %w", pageTemplatePath, err)
	}

	text := string(source)
	matches := liveContainerPattern.FindStringSubmatch(text)
	if len(matches) == 0 {
		return "", fmt.Errorf(
			"%q must contain an element with id and data-signals for live routes",
			filepath.ToSlash(pageTemplatePath),
		)
	}
	if matches[1] != "" {
		return matches[1], nil
	}
	if matches[2] != "" {
		return matches[2], nil
	}

	return "", fmt.Errorf("%q has data-signals but selector id could not be parsed", filepath.ToSlash(pageTemplatePath))
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

func generateContractsSource(metas []routeMeta) ([]byte, error) {
	importLines := []string{
		"\"context\"",
		"\"net/http\"",
		"\"blog/internal/web/appcore\"",
	}

	resolverImports := make([]string, 0, len(metas))
	seen := make(map[string]struct{}, len(metas))
	for _, meta := range metas {
		if _, ok := seen[meta.ResolverAlias]; ok {
			continue
		}
		seen[meta.ResolverAlias] = struct{}{}
		resolverImports = append(resolverImports, fmt.Sprintf("%s \"blog/%s\"", meta.ResolverAlias, meta.ResolverImportPath))
	}
	sort.Strings(resolverImports)
	importLines = append(importLines, resolverImports...)

	buffer := &bytes.Buffer{}
	buffer.WriteString("// Code generated by framework/cmd/approutegen. DO NOT EDIT.\n")
	buffer.WriteString("package gen\n\n")
	buffer.WriteString("import (\n")
	for _, line := range importLines {
		buffer.WriteString("\t" + line + "\n")
	}
	buffer.WriteString(")\n\n")

	for _, meta := range metas {
		writeParamsStruct(buffer, meta)
	}

	buffer.WriteString("type RouteResolvers interface {\n")
	for _, meta := range metas {
		writef(
			buffer,
			"\t%s(ctx context.Context, appCtx *appcore.Context, r *http.Request, params %s) (%s.PageView, error)\n",
			resolvePageMethod(meta),
			meta.ParamsTypeName,
			meta.ResolverAlias,
		)
		if meta.HasLive {
			writef(
				buffer,
				"\t%s(r *http.Request) (%s.LiveState, error)\n",
				parseLiveMethod(meta),
				meta.ResolverAlias,
			)
			writef(
				buffer,
				"\t%s(ctx context.Context, appCtx *appcore.Context, r *http.Request, params %s, "+
					"state %s.LiveState) (%s.PageView, error)\n",
				resolveLiveMethod(meta),
				meta.ParamsTypeName,
				meta.ResolverAlias,
				meta.ResolverAlias,
			)
		}
	}
	buffer.WriteString("}\n")

	formatted, err := format.Source(buffer.Bytes())
	if err != nil {
		return nil, fmt.Errorf("format contracts source: %w", err)
	}
	return formatted, nil
}

func writeParamsStruct(buffer *bytes.Buffer, meta routeMeta) {
	writef(buffer, "type %s struct {\n", meta.ParamsTypeName)
	if len(meta.Params) == 0 {
		buffer.WriteString("}\n\n")
		return
	}
	for _, param := range meta.Params {
		writef(buffer, "\t%s string\n", param.FieldName)
	}
	buffer.WriteString("}\n\n")
}

func generateRegistrySource(
	paths generationPaths,
	metas []routeMeta,
	layouts map[string]templateDef,
) ([]byte, error) {
	importLines := []string{
		"\"context\"",
		"\"net/http\"",
		"\"strings\"",
		"\"blog/framework\"",
		"\"blog/framework/router\"",
		"\"blog/internal/web/appcore\"",
		"\"github.com/a-h/templ\"",
	}

	moduleImports := make([]string, 0, len(metas)+len(layouts)+len(metas))
	for _, meta := range metas {
		moduleImports = append(moduleImports, fmt.Sprintf(
			"%s \"blog/%s/%s\"",
			meta.Page.ModuleName,
			paths.GenImportRoot,
			meta.Page.ModuleName,
		))
		moduleImports = append(moduleImports, fmt.Sprintf(
			"%s \"blog/%s\"",
			meta.ResolverAlias,
			meta.ResolverImportPath,
		))
	}

	layoutKeys := make([]string, 0, len(layouts))
	for routeID := range layouts {
		layoutKeys = append(layoutKeys, routeID)
	}
	sort.Strings(layoutKeys)
	for _, routeID := range layoutKeys {
		layout := layouts[routeID]
		moduleImports = append(moduleImports, fmt.Sprintf(
			"%s \"blog/%s/%s\"",
			layout.ModuleName,
			paths.GenImportRoot,
			layout.ModuleName,
		))
	}

	moduleImports = dedupeSorted(moduleImports)
	importLines = append(importLines, moduleImports...)

	buffer := &bytes.Buffer{}
	buffer.WriteString("// Code generated by framework/cmd/approutegen. DO NOT EDIT.\n")
	buffer.WriteString("package gen\n\n")
	buffer.WriteString("import (\n")
	for _, line := range importLines {
		buffer.WriteString("\t" + line + "\n")
	}
	buffer.WriteString(")\n\n")

	buffer.WriteString("func Handlers(resolvers RouteResolvers) []framework.RouteHandler[*appcore.Context] {\n")
	buffer.WriteString("\treturn []framework.RouteHandler[*appcore.Context]{\n")
	for _, meta := range metas {
		if meta.HasLive {
			writef(
				buffer,
				"\t\tframework.PageAndLiveRouteHandler[*appcore.Context, %s, %s.PageView, %s.LiveState]{\n",
				meta.ParamsTypeName,
				meta.ResolverAlias,
				meta.ResolverAlias,
			)
		} else {
			writef(
				buffer,
				"\t\tframework.PageOnlyRouteHandler[*appcore.Context, %s, %s.PageView]{\n",
				meta.ParamsTypeName,
				meta.ResolverAlias,
			)
		}

		writePageModule(buffer, meta, layouts)
		if meta.HasLive {
			writeLiveModule(buffer, meta)
		}
		buffer.WriteString("\t\t},\n")
	}
	buffer.WriteString("\t}\n")
	buffer.WriteString("}\n\n")

	for _, meta := range metas {
		writeParseParamsFunc(buffer, meta, false)
		if meta.HasLive {
			writeParseParamsFunc(buffer, meta, true)
		}
	}

	wrappers, err := collectLayoutWrappers(metas, layouts)
	if err != nil {
		return nil, err
	}
	wrapperNames := make([]string, 0, len(wrappers))
	for name := range wrappers {
		wrapperNames = append(wrapperNames, name)
	}
	sort.Strings(wrapperNames)
	for _, name := range wrapperNames {
		wrapper := wrappers[name]
		writef(
			buffer,
			"func %s(view %s.PageView, child templ.Component) templ.Component {\n",
			wrapper.Name,
			wrapper.ViewAlias,
		)
		writef(buffer, "\treturn %s.Layout(view, child)\n", wrapper.LayoutModule)
		buffer.WriteString("}\n\n")
	}

	formatted, err := format.Source(buffer.Bytes())
	if err != nil {
		return nil, fmt.Errorf("format registry source: %w", err)
	}
	return formatted, nil
}

type layoutWrapperDef struct {
	Name         string
	ViewAlias    string
	LayoutModule string
}

func writePageModule(buffer *bytes.Buffer, meta routeMeta, layouts map[string]templateDef) {
	writef(
		buffer,
		"\t\t\tPage: framework.PageModule[*appcore.Context, %s, %s.PageView]{\n",
		meta.ParamsTypeName,
		meta.ResolverAlias,
	)
	writef(buffer, "\t\t\t\tPattern:     %q,\n", routePattern(meta.RouteID))
	writef(buffer, "\t\t\t\tParseParams: %s,\n", parseParamsFuncName(meta, false))
	writef(
		buffer,
		"\t\t\t\tLoad: func(ctx context.Context, appCtx *appcore.Context, r *http.Request, "+
			"params %s) (%s.PageView, error) {\n",
		meta.ParamsTypeName,
		meta.ResolverAlias,
	)
	writef(buffer, "\t\t\t\t\treturn resolvers.%s(ctx, appCtx, r, params)\n", resolvePageMethod(meta))
	buffer.WriteString("\t\t\t\t},\n")
	writef(buffer, "\t\t\t\tRender: %s.Page,\n", meta.Page.ModuleName)

	chain := layoutChain(meta.RouteID, layouts)
	if len(chain) == 0 {
		writef(buffer, "\t\t\t\tLayouts: []framework.LayoutRenderer[%s.PageView]{},\n", meta.ResolverAlias)
	} else {
		writef(buffer, "\t\t\t\tLayouts: []framework.LayoutRenderer[%s.PageView]{\n", meta.ResolverAlias)
		for _, layout := range chain {
			layoutName := routeNameFromSegments(layout.Segments)
			writef(buffer, "\t\t\t\t\t%s,\n", wrapperFuncName(meta.RouteName, layoutName))
		}
		buffer.WriteString("\t\t\t\t},\n")
	}
	buffer.WriteString("\t\t\t},\n")
}

func writeLiveModule(buffer *bytes.Buffer, meta routeMeta) {
	writef(
		buffer,
		"\t\t\tLive: framework.LiveModule[*appcore.Context, %s, %s.PageView, %s.LiveState]{\n",
		meta.ParamsTypeName,
		meta.ResolverAlias,
		meta.ResolverAlias,
	)
	writef(buffer, "\t\t\t\tPattern:           %q,\n", routePattern(meta.RouteID)+"/live")
	writef(buffer, "\t\t\t\tParseParams:       %s,\n", parseParamsFuncName(meta, true))
	writef(buffer, "\t\t\t\tParseState:        resolvers.%s,\n", parseLiveMethod(meta))
	writef(
		buffer,
		"\t\t\t\tLoad: func(ctx context.Context, appCtx *appcore.Context, r *http.Request, "+
			"params %s, state %s.LiveState) (%s.PageView, error) {\n",
		meta.ParamsTypeName,
		meta.ResolverAlias,
		meta.ResolverAlias,
	)
	writef(buffer, "\t\t\t\t\treturn resolvers.%s(ctx, appCtx, r, params, state)\n", resolveLiveMethod(meta))
	buffer.WriteString("\t\t\t\t},\n")
	writef(buffer, "\t\t\t\tRender:            %s.Page,\n", meta.Page.ModuleName)
	writef(buffer, "\t\t\t\tSelectorID:        %q,\n", meta.LiveSelectorID)
	writef(buffer, "\t\t\t\tBadRequestMessage: %q,\n", defaultLiveBadRequestMessage)
	buffer.WriteString("\t\t\t},\n")
}

func collectLayoutWrappers(metas []routeMeta, layouts map[string]templateDef) (map[string]layoutWrapperDef, error) {
	wrappers := make(map[string]layoutWrapperDef)
	for _, meta := range metas {
		chain := layoutChain(meta.RouteID, layouts)
		for _, layout := range chain {
			layoutName := routeNameFromSegments(layout.Segments)
			name := wrapperFuncName(meta.RouteName, layoutName)
			def := layoutWrapperDef{
				Name:         name,
				ViewAlias:    meta.ResolverAlias,
				LayoutModule: layout.ModuleName,
			}

			existing, ok := wrappers[name]
			if !ok {
				wrappers[name] = def
				continue
			}
			if existing.ViewAlias != def.ViewAlias || existing.LayoutModule != def.LayoutModule {
				return nil, fmt.Errorf("layout wrapper conflict for %q", name)
			}
		}
	}

	return wrappers, nil
}

func parseParamsFuncName(meta routeMeta, live bool) string {
	if live {
		return "parse" + meta.RouteName + "LiveParams"
	}
	return "parse" + meta.RouteName + "Params"
}

func writeParseParamsFunc(buffer *bytes.Buffer, meta routeMeta, live bool) {
	funcName := parseParamsFuncName(meta, live)
	pattern := routePattern(meta.RouteID)
	if live {
		pattern += "/live"
	}

	writef(buffer, "func %s(requestPath string) (%s, bool) {\n", funcName, meta.ParamsTypeName)
	if len(meta.Params) == 0 {
		writef(buffer, "\t_, ok := router.MatchPathPattern(%q, requestPath)\n", pattern)
		buffer.WriteString("\tif !ok {\n")
		writef(buffer, "\t\treturn %s{}, false\n", meta.ParamsTypeName)
		buffer.WriteString("\t}\n")
		writef(buffer, "\treturn %s{}, true\n", meta.ParamsTypeName)
		buffer.WriteString("}\n\n")
		return
	}

	writef(buffer, "\tparams, ok := router.MatchPathPattern(%q, requestPath)\n", pattern)
	buffer.WriteString("\tif !ok {\n")
	writef(buffer, "\t\treturn %s{}, false\n", meta.ParamsTypeName)
	buffer.WriteString("\t}\n")
	writef(buffer, "\tout := %s{}\n", meta.ParamsTypeName)
	for _, param := range meta.Params {
		writef(buffer, "\t%sValue := strings.TrimSpace(params[%q])\n", param.FieldName, param.Name)
		if param.Name == "slug" {
			writef(buffer, "\tif !router.IsValidSlug(%sValue) {\n", param.FieldName)
			writef(buffer, "\t\treturn %s{}, false\n", meta.ParamsTypeName)
			buffer.WriteString("\t}\n")
		}
		writef(buffer, "\tout.%s = %sValue\n", param.FieldName, param.FieldName)
	}
	buffer.WriteString("\treturn out, true\n")
	buffer.WriteString("}\n\n")
}

func routePattern(routeID string) string {
	if routeID == "" {
		return "/"
	}
	return "/" + routeID
}

func resolvePageMethod(meta routeMeta) string {
	return "Resolve" + meta.RouteName + "Page"
}

func parseLiveMethod(meta routeMeta) string {
	return "Parse" + meta.RouteName + "LiveState"
}

func resolveLiveMethod(meta routeMeta) string {
	return "Resolve" + meta.RouteName + "Live"
}

func wrapperFuncName(routeName string, layoutName string) string {
	return "wrap" + routeName + "With" + layoutName + "Layout"
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

func generateResolversSource(metas []routeMeta) ([]byte, error) {
	importLines := []string{
		"\"context\"",
		"\"net/http\"",
		"\"blog/internal/web/appcore\"",
	}

	routeImports := make([]string, 0, len(metas))
	for _, meta := range metas {
		routeImports = append(routeImports, fmt.Sprintf("%s \"blog/%s\"", meta.ResolverAlias, meta.ResolverImportPath))
	}
	routeImports = dedupeSorted(routeImports)
	importLines = append(importLines, routeImports...)

	buffer := &bytes.Buffer{}
	buffer.WriteString("// Code generated by framework/cmd/approutegen. DO NOT EDIT.\n")
	buffer.WriteString("package gen\n\n")
	buffer.WriteString("import (\n")
	for _, line := range importLines {
		buffer.WriteString("\t" + line + "\n")
	}
	buffer.WriteString(")\n\n")

	buffer.WriteString("type generatedResolvers struct {\n")
	for _, meta := range metas {
		writef(buffer, "\t%s %s.Resolver\n", meta.ResolverField, meta.ResolverAlias)
	}
	buffer.WriteString("}\n\n")

	buffer.WriteString("func NewRouteResolvers() RouteResolvers {\n")
	buffer.WriteString("\treturn &generatedResolvers{}\n")
	buffer.WriteString("}\n\n")
	buffer.WriteString("var _ RouteResolvers = (*generatedResolvers)(nil)\n\n")

	for _, meta := range metas {
		writef(
			buffer,
			"func (r *generatedResolvers) %s(ctx context.Context, appCtx *appcore.Context, req *http.Request, "+
				"params %s) (%s.PageView, error) {\n",
			resolvePageMethod(meta),
			meta.ParamsTypeName,
			meta.ResolverAlias,
		)
		writef(
			buffer,
			"\treturn r.%s.ResolvePage(ctx, appCtx, req, to%sParams(params))\n",
			meta.ResolverField,
			meta.RouteName,
		)
		buffer.WriteString("}\n\n")

		if meta.HasLive {
			writef(
				buffer,
				"func (r *generatedResolvers) %s(req *http.Request) (%s.LiveState, error) {\n",
				parseLiveMethod(meta),
				meta.ResolverAlias,
			)
			writef(buffer, "\treturn r.%s.ParseLiveState(req)\n", meta.ResolverField)
			buffer.WriteString("}\n\n")

			writef(
				buffer,
				"func (r *generatedResolvers) %s(ctx context.Context, appCtx *appcore.Context, req *http.Request, "+
					"params %s, state %s.LiveState) (%s.PageView, error) {\n",
				resolveLiveMethod(meta),
				meta.ParamsTypeName,
				meta.ResolverAlias,
				meta.ResolverAlias,
			)
			writef(
				buffer,
				"\treturn r.%s.ResolveLive(ctx, appCtx, req, to%sParams(params), state)\n",
				meta.ResolverField,
				meta.RouteName,
			)
			buffer.WriteString("}\n\n")
		}

		writef(
			buffer,
			"func to%sParams(params %s) %s.Params {\n",
			meta.RouteName,
			meta.ParamsTypeName,
			meta.ResolverAlias,
		)
		if len(meta.Params) == 0 {
			buffer.WriteString("\t_ = params\n")
			writef(buffer, "\treturn %s.Params{}\n", meta.ResolverAlias)
			buffer.WriteString("}\n\n")
			continue
		}
		writef(buffer, "\treturn %s.Params{\n", meta.ResolverAlias)
		for _, param := range meta.Params {
			writef(buffer, "\t\t%s: params.%s,\n", param.FieldName, param.FieldName)
		}
		buffer.WriteString("\t}\n")
		buffer.WriteString("}\n\n")
	}

	formatted, err := format.Source(buffer.Bytes())
	if err != nil {
		return nil, fmt.Errorf("format resolver adapter source: %w", err)
	}
	return formatted, nil
}

func ensureRouteResolverStub(meta routeMeta) error {
	resolverPath := filepath.Join(meta.ResolverDir, resolverFileName)
	if pathExists(resolverPath) {
		return nil
	}

	source, err := generateRouteResolverStubSource(meta)
	if err != nil {
		return err
	}

	if err := os.WriteFile(resolverPath, source, 0o644); err != nil {
		return fmt.Errorf("write resolver stub %q: %w", filepath.ToSlash(resolverPath), err)
	}

	return nil
}

func generateRouteResolverStubSource(meta routeMeta) ([]byte, error) {
	buffer := &bytes.Buffer{}
	writef(buffer, "package %s\n\n", meta.ResolverPackage)
	buffer.WriteString("import (\n")
	buffer.WriteString("\t\"context\"\n")
	buffer.WriteString("\t\"errors\"\n")
	buffer.WriteString("\t\"net/http\"\n")
	buffer.WriteString("\t\"blog/internal/web/appcore\"\n")
	buffer.WriteString(")\n\n")
	buffer.WriteString("type Params struct {\n")
	for _, param := range meta.Params {
		writef(buffer, "\t%s string\n", param.FieldName)
	}
	buffer.WriteString("}\n\n")
	buffer.WriteString("type routeResolver interface {\n")
	buffer.WriteString(
		"\tResolvePage(ctx context.Context, appCtx *appcore.Context, r *http.Request, params Params) (PageView, error)\n",
	)
	if meta.HasLive {
		buffer.WriteString("\tParseLiveState(r *http.Request) (LiveState, error)\n")
		buffer.WriteString(
			"\tResolveLive(ctx context.Context, appCtx *appcore.Context, r *http.Request, " +
				"params Params, state LiveState) (PageView, error)\n",
		)
	}
	buffer.WriteString("}\n\n")
	buffer.WriteString("type Resolver struct{}\n\n")
	buffer.WriteString("var _ routeResolver = (*Resolver)(nil)\n\n")

	writef(
		buffer,
		"func (Resolver) ResolvePage(ctx context.Context, appCtx *appcore.Context, r *http.Request, "+
			"params Params) (PageView, error) {\n",
	)
	buffer.WriteString("\t_ = ctx\n")
	buffer.WriteString("\t_ = appCtx\n")
	buffer.WriteString("\t_ = r\n")
	buffer.WriteString("\t_ = params\n")
	buffer.WriteString("\tvar view PageView\n")
	writef(buffer, "\treturn view, errors.New(%q)\n", "TODO: implement ResolvePage for route "+routePattern(meta.RouteID))
	buffer.WriteString("}\n\n")

	if meta.HasLive {
		buffer.WriteString("func (Resolver) ParseLiveState(r *http.Request) (LiveState, error) {\n")
		buffer.WriteString("\t_ = r\n")
		buffer.WriteString("\tvar state LiveState\n")
		writef(
			buffer,
			"\treturn state, errors.New(%q)\n",
			"TODO: implement ParseLiveState for route "+routePattern(meta.RouteID),
		)
		buffer.WriteString("}\n\n")

		buffer.WriteString(
			"func (Resolver) ResolveLive(ctx context.Context, appCtx *appcore.Context, r *http.Request, " +
				"params Params, state LiveState) (PageView, error) {\n",
		)
		buffer.WriteString("\t_ = ctx\n")
		buffer.WriteString("\t_ = appCtx\n")
		buffer.WriteString("\t_ = r\n")
		buffer.WriteString("\t_ = params\n")
		buffer.WriteString("\t_ = state\n")
		buffer.WriteString("\tvar view PageView\n")
		writef(buffer, "\treturn view, errors.New(%q)\n", "TODO: implement ResolveLive for route "+routePattern(meta.RouteID))
		buffer.WriteString("}\n")
	}

	formatted, err := format.Source(buffer.Bytes())
	if err != nil {
		return nil, fmt.Errorf("format route resolver stub for %q: %w", meta.RouteID, err)
	}
	return formatted, nil
}

func safeIdentifier(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "value"
	}

	normalized := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, trimmed)
	normalized = strings.Trim(normalized, "_")
	if normalized == "" {
		return "value"
	}
	return strings.ToLower(normalized)
}

func pascalToken(value string) string {
	normalized := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return ' '
	}, strings.TrimSpace(value))

	parts := strings.Fields(normalized)
	if len(parts) == 0 {
		return ""
	}

	builder := strings.Builder{}
	for _, part := range parts {
		if part == "" {
			continue
		}
		builder.WriteString(strings.ToUpper(part[:1]))
		if len(part) > 1 {
			builder.WriteString(strings.ToLower(part[1:]))
		}
	}

	return builder.String()
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
