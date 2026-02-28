package approutegen

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverRouteFilesStaticAndDynamic(t *testing.T) {
	root := t.TempDir()
	appRoot := filepath.Join(root, "app")
	genRoot := filepath.Join(root, "gen")

	writeTestFile(t, filepath.Join(appRoot, "layout.templ"), "package appsrc\n")
	writeTestFile(t, filepath.Join(appRoot, "notes", "page.templ"), "package appsrc\n")
	writeTestFile(t, filepath.Join(appRoot, "author", "[slug]", "page.templ"), "package appsrc\n")

	routes, err := discoverRouteFiles(appRoot, genRoot)
	if err != nil {
		t.Fatalf("discover routes: %v", err)
	}

	if len(routes.Pages) != 2 {
		t.Fatalf("expected 2 pages, got %d", len(routes.Pages))
	}
	if routes.Pages[0].RouteID != "author/[slug]" {
		t.Fatalf("expected first route author/[slug], got %q", routes.Pages[0].RouteID)
	}
	if routes.Pages[1].RouteID != "notes" {
		t.Fatalf("expected second route notes, got %q", routes.Pages[1].RouteID)
	}
}

func TestDiscoverRouteFilesRejectsRouteLocalComponents(t *testing.T) {
	root := t.TempDir()
	appRoot := filepath.Join(root, "app")
	genRoot := filepath.Join(root, "gen")

	writeTestFile(t, filepath.Join(appRoot, "notes", "page.templ"), "package appsrc\n")
	writeTestFile(t, filepath.Join(appRoot, "notes", "components", "card.templ"), "package appsrc\n")

	_, err := discoverRouteFiles(appRoot, genRoot)
	if err == nil {
		t.Fatal("expected route-local components error")
	}
	if !strings.Contains(err.Error(), "internal/web/components") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDiscoverRouteFilesRejectsRootComponentsDir(t *testing.T) {
	root := t.TempDir()
	appRoot := filepath.Join(root, "app")
	genRoot := filepath.Join(root, "gen")

	writeTestFile(t, filepath.Join(appRoot, "components", "note_card.templ"), "package appsrc\n")

	_, err := discoverRouteFiles(appRoot, genRoot)
	if err == nil {
		t.Fatal("expected root components error")
	}
	if !strings.Contains(err.Error(), "internal/web/components") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDiscoverRouteFilesRejectsLegacyWildcardSyntax(t *testing.T) {
	root := t.TempDir()
	appRoot := filepath.Join(root, "app")
	genRoot := filepath.Join(root, "gen")

	writeTestFile(t, filepath.Join(appRoot, "note", "_slug", "page.templ"), "package appsrc\n")

	_, err := discoverRouteFiles(appRoot, genRoot)
	if err == nil {
		t.Fatal("expected legacy wildcard syntax error")
	}
	if !strings.Contains(err.Error(), "use [param]") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDiscoverRouteFilesCollectsNotFoundTemplates(t *testing.T) {
	root := t.TempDir()
	appRoot := filepath.Join(root, "app")
	genRoot := filepath.Join(root, "gen")

	writeTestFile(t, filepath.Join(appRoot, "404.templ"), "package appsrc\n\ntempl Page(path string) { <div>{ path }</div> }\n")
	writeTestFile(t, filepath.Join(appRoot, "author", "[slug]", "404.templ"), "package appsrc\n\ntempl Page(path string) { <div>{ path }</div> }\n")
	writeTestFile(t, filepath.Join(appRoot, "author", "[slug]", "page.templ"), "package appsrc\n\nimport \"blog/internal/web/appcore\"\n\ntempl Page(view appcore.AuthorPageView) { <div id=\"notes-content\"></div> }\n")

	routes, err := discoverRouteFiles(appRoot, genRoot)
	if err != nil {
		t.Fatalf("discover routes: %v", err)
	}

	if _, ok := routes.NotFounds[""]; !ok {
		t.Fatalf("expected root 404 template")
	}
	if _, ok := routes.NotFounds["author/[slug]"]; !ok {
		t.Fatalf("expected nested author 404 template")
	}
}

func TestParsePageViewType(t *testing.T) {
	root := t.TempDir()
	pagePath := filepath.Join(root, "page.templ")
	writeTestFile(t, pagePath, "package appsrc\n\nimport \"blog/internal/web/appcore\"\n\ntempl Page(view appcore.NotePageView) { <div/> }\n")

	viewType, err := parsePageViewType(pagePath)
	if err != nil {
		t.Fatalf("parse page view type: %v", err)
	}
	if viewType != "appcore.NotePageView" {
		t.Fatalf("expected appcore.NotePageView, got %q", viewType)
	}
}

func TestParsePageViewTypeRejectsNonAppcoreType(t *testing.T) {
	root := t.TempDir()
	pagePath := filepath.Join(root, "page.templ")
	writeTestFile(t, pagePath, "package appsrc\n\ntempl Page(view note.NotePageView) { <div/> }\n")

	_, err := parsePageViewType(pagePath)
	if err == nil {
		t.Fatal("expected appcore-qualified type error")
	}
	if !strings.Contains(err.Error(), "appcore-qualified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateLayoutTemplateSignature(t *testing.T) {
	root := t.TempDir()
	validPath := filepath.Join(root, "layout_valid.templ")
	invalidPath := filepath.Join(root, "layout_invalid.templ")
	writeTestFile(
		t,
		validPath,
		"package appsrc\n\nimport \"blog/internal/web/appcore\"\n\ntempl Layout(view appcore.RootLayoutView, child templ.Component) { @child }\n",
	)
	writeTestFile(
		t,
		invalidPath,
		"package appsrc\n\nimport \"blog/internal/web/appcore\"\n\ntempl Layout(view appcore.NotesPageView, child templ.Component) { @child }\n",
	)

	if err := validateLayoutTemplateSignature(validPath); err != nil {
		t.Fatalf("expected valid signature, got %v", err)
	}
	if err := validateLayoutTemplateSignature(invalidPath); err == nil {
		t.Fatal("expected invalid layout signature error")
	}
}

func TestValidateNotFoundTemplateSignature(t *testing.T) {
	root := t.TempDir()
	validPath := filepath.Join(root, "404_valid.templ")
	invalidPath := filepath.Join(root, "404_invalid.templ")
	writeTestFile(t, validPath, "package appsrc\n\ntempl Page(path string) { <div>{ path }</div> }\n")
	writeTestFile(t, invalidPath, "package appsrc\n\ntempl Page(target string) { <div>{ target }</div> }\n")

	if err := validateNotFoundTemplateSignature(validPath); err != nil {
		t.Fatalf("expected valid 404 signature, got %v", err)
	}
	if err := validateNotFoundTemplateSignature(invalidPath); err == nil {
		t.Fatal("expected invalid 404 signature error")
	}
}

func TestBuildRouteMetasPageOnly(t *testing.T) {
	root := t.TempDir()
	appRoot := filepath.Join(root, "app")
	genRoot := filepath.Join(root, "gen")

	rootTemplate := "package appsrc\n\nimport \"blog/internal/web/appcore\"\n\ntempl Page(view appcore.NotesPageView) { <div id=\"notes-content\"></div> }\n"
	authorTemplate := "package appsrc\n\nimport \"blog/internal/web/appcore\"\n\ntempl Page(view appcore.AuthorPageView) { <div id=\"notes-content\"></div> }\n"
	writeTestFile(t, filepath.Join(appRoot, "page.templ"), rootTemplate)
	writeTestFile(t, filepath.Join(appRoot, "author", "[slug]", "page.templ"), authorTemplate)

	routes, err := discoverRouteFiles(appRoot, genRoot)
	if err != nil {
		t.Fatalf("discover routes: %v", err)
	}

	metas, err := buildRouteMetas(routes.Pages, generationPaths{})
	if err != nil {
		t.Fatalf("build route metas: %v", err)
	}

	byRoute := map[string]routeMeta{}
	for _, meta := range metas {
		byRoute[meta.RouteID] = meta
	}

	rootMeta, ok := byRoute[""]
	if !ok {
		t.Fatalf("missing root route meta: %#v", byRoute)
	}
	if rootMeta.PageViewType != "appcore.NotesPageView" {
		t.Fatalf("expected root page view type, got %q", rootMeta.PageViewType)
	}

	authorMeta, ok := byRoute["author/[slug]"]
	if !ok {
		t.Fatalf("missing author route meta: %#v", byRoute)
	}
	if authorMeta.PageViewType != "appcore.AuthorPageView" {
		t.Fatalf("expected author page view type, got %q", authorMeta.PageViewType)
	}
}

func TestBuildRouteMetasAllowsNonPageViewSuffix(t *testing.T) {
	root := t.TempDir()
	appRoot := filepath.Join(root, "app")
	genRoot := filepath.Join(root, "gen")

	pageTemplate := "package appsrc\n\nimport \"blog/internal/web/appcore\"\n\ntempl Page(view appcore.NoteView) { <div id=\"note-content\"></div> }\n"
	writeTestFile(t, filepath.Join(appRoot, "note", "[slug]", "page.templ"), pageTemplate)

	routes, err := discoverRouteFiles(appRoot, genRoot)
	if err != nil {
		t.Fatalf("discover routes: %v", err)
	}

	metas, err := buildRouteMetas(routes.Pages, generationPaths{})
	if err != nil {
		t.Fatalf("build route metas: %v", err)
	}
	if len(metas) != 1 {
		t.Fatalf("expected 1 route meta, got %d", len(metas))
	}
	if metas[0].PageViewType != "appcore.NoteView" {
		t.Fatalf("expected appcore.NoteView, got %q", metas[0].PageViewType)
	}
}

func TestResolverNamespaceGenerationDeterministic(t *testing.T) {
	metas := []routeMeta{
		{
			RouteID:        "",
			RouteName:      "Root",
			ParamsTypeName: "RootParams",
			PageViewType:   "appcore.NotesPageView",
		},
		{
			RouteID:        "author/[slug]",
			RouteName:      "AuthorParamSlug",
			ParamsTypeName: "AuthorParamSlugParams",
			Params:         []routeParamDef{{Name: "slug", FieldName: "Slug"}},
			PageViewType:   "appcore.AuthorPageView",
		},
	}

	first, err := generateResolverNamespaceSource(metas)
	if err != nil {
		t.Fatalf("first generation failed: %v", err)
	}
	second, err := generateResolverNamespaceSource(metas)
	if err != nil {
		t.Fatalf("second generation failed: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("resolver namespace generation is not deterministic")
	}
	if !bytes.Contains(first, []byte("var _ RouteResolver = (*Resolver)(nil)")) {
		t.Fatalf("expected compile-time assertion in generated resolver namespace:\n%s", string(first))
	}
}

func TestRegistryGenerationUsesSingleResolverNamespace(t *testing.T) {
	metas := []routeMeta{
		{
			RouteID:        "",
			RouteName:      "Root",
			ParamsTypeName: "RootParams",
			PageViewType:   "appcore.NotesPageView",
			Page:           templateDef{ModuleName: "r_page_root"},
		},
		{
			RouteID:        "author/[slug]",
			RouteName:      "AuthorParamSlug",
			ParamsTypeName: "AuthorParamSlugParams",
			Params:         []routeParamDef{{Name: "slug", FieldName: "Slug"}},
			PageViewType:   "appcore.AuthorPageView",
			Page:           templateDef{ModuleName: "r_page_author_param_slug"},
		},
	}

	registry, err := generateRegistrySource(
		generationPaths{GenImportRoot: "internal/web/gen"},
		metas,
		map[string]templateDef{},
		map[string]templateDef{
			"": {
				Kind:       notFoundTemplate,
				RouteID:    "",
				ModuleName: "r_not_found_root",
			},
		},
	)
	if err != nil {
		t.Fatalf("generate registry: %v", err)
	}

	text := string(registry)
	if !strings.Contains(text, "route_resolvers \"blog/internal/web/resolvers\"") {
		t.Fatalf("expected unified resolver namespace import in registry:\n%s", text)
	}
	if strings.Contains(text, "rr_") {
		t.Fatalf("did not expect per-route resolver aliases in registry:\n%s", text)
	}
	if !strings.Contains(text, "func NewRouteResolvers() RouteResolvers") {
		t.Fatalf("expected NewRouteResolvers constructor in registry:\n%s", text)
	}
	if !strings.Contains(text, "return &route_resolvers.Resolver{}") {
		t.Fatalf("expected route resolver constructor to return unified resolver:\n%s", text)
	}
	if !strings.Contains(text, "framework.PageOnlyRouteHandler") {
		t.Fatalf("expected page-only route handlers:\n%s", text)
	}
	if strings.Contains(text, "PageAndLiveRouteHandler") {
		t.Fatalf("did not expect live route handlers:\n%s", text)
	}
	if strings.Contains(text, "/.live/") {
		t.Fatalf("did not expect live route patterns:\n%s", text)
	}
	if strings.Contains(text, "ParseRootLiveState") {
		t.Fatalf("did not expect live resolver contract references:\n%s", text)
	}
	if !strings.Contains(text, "func NotFoundPage(notFound framework.NotFoundContext) templ.Component") {
		t.Fatalf("expected generated NotFoundPage helper in registry:\n%s", text)
	}
}

func TestRegistryGenerationRequiresRootNotFoundTemplate(t *testing.T) {
	metas := []routeMeta{
		{
			RouteID:        "",
			RouteName:      "Root",
			ParamsTypeName: "RootParams",
			PageViewType:   "appcore.NotesPageView",
			Page:           templateDef{ModuleName: "r_page_root"},
		},
	}

	_, err := generateRegistrySource(
		generationPaths{GenImportRoot: "internal/web/gen"},
		metas,
		map[string]templateDef{},
		map[string]templateDef{},
	)
	if err == nil {
		t.Fatal("expected missing root 404 metadata error")
	}
	if !strings.Contains(err.Error(), "missing root 404") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRewritePackageDeclarationAddsGeneratedMarker(t *testing.T) {
	source := "package appsrc\n\nimport (\n\t\"fmt\"\n)\n"

	rewritten, err := rewritePackageDeclaration([]byte(source), "r_page_root")
	if err != nil {
		t.Fatalf("rewrite package declaration: %v", err)
	}

	text := string(rewritten)
	if !strings.HasPrefix(text, "package r_page_root\n"+generatedTemplHeader+"\n") {
		t.Fatalf("expected generated marker after package declaration, got:\n%s", text)
	}
	if strings.Count(text, generatedTemplHeader) != 1 {
		t.Fatalf("expected exactly one generated marker, got:\n%s", text)
	}
}

func TestRewritePackageDeclarationKeepsSingleGeneratedMarker(t *testing.T) {
	source := "package appsrc\n\n" + generatedTemplHeader + "\n\ntempl Page() { <div></div> }\n"

	rewritten, err := rewritePackageDeclaration([]byte(source), "r_page_root")
	if err != nil {
		t.Fatalf("rewrite package declaration: %v", err)
	}

	text := string(rewritten)
	if strings.Count(text, generatedTemplHeader) != 1 {
		t.Fatalf("expected exactly one generated marker, got:\n%s", text)
	}
	if !strings.HasPrefix(text, "package r_page_root\n") {
		t.Fatalf("expected package rename to be applied, got:\n%s", text)
	}
}

func writeTestFile(t *testing.T, filePath string, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatalf("mkdir %q: %v", filepath.Dir(filePath), err)
	}
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("write %q: %v", filePath, err)
	}
}
