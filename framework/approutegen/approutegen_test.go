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

func TestBuildRouteMetasMissingTypesFile(t *testing.T) {
	root := t.TempDir()
	appRoot := filepath.Join(root, "app")
	genRoot := filepath.Join(root, "gen")

	writeTestFile(t, filepath.Join(appRoot, "note", "[slug]", "page.templ"), "package appsrc\n")
	routes, err := discoverRouteFiles(appRoot, genRoot)
	if err != nil {
		t.Fatalf("discover routes: %v", err)
	}

	_, err = buildRouteMetas(routes.Pages, generationPaths{
		ResolverRoot:       filepath.Join(root, "appcore", "resolvers"),
		ResolverImportRoot: "internal/web/appcore/resolvers",
	})
	if err == nil {
		t.Fatal("expected missing types.go error")
	}
	if !strings.Contains(err.Error(), "types.go") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildRouteMetasMissingPageView(t *testing.T) {
	root := t.TempDir()
	appRoot := filepath.Join(root, "app")
	genRoot := filepath.Join(root, "gen")
	resolverRoot := filepath.Join(root, "appcore", "resolvers")

	writeTestFile(t, filepath.Join(appRoot, "notes", "page.templ"), "package appsrc\n")
	writeTestFile(t, filepath.Join(resolverRoot, "notes", "types.go"), "package notes\n\ntype LiveState struct{}\n")

	routes, err := discoverRouteFiles(appRoot, genRoot)
	if err != nil {
		t.Fatalf("discover routes: %v", err)
	}

	_, err = buildRouteMetas(routes.Pages, generationPaths{
		ResolverRoot:       resolverRoot,
		ResolverImportRoot: "internal/web/appcore/resolvers",
	})
	if err == nil {
		t.Fatal("expected missing PageView error")
	}
	if !strings.Contains(err.Error(), "PageView") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLiveGenerationRequiresLiveStateDeclaration(t *testing.T) {
	root := t.TempDir()
	appRoot := filepath.Join(root, "app")
	genRoot := filepath.Join(root, "gen")
	resolverRoot := filepath.Join(root, "appcore", "resolvers")

	notesPageTemplate := "package appsrc\n\n" +
		"templ Page() { <div id=\"notes-content\" data-signals=\"{}\"></div> }\n"
	notePageTemplate := "package appsrc\n\n" +
		"templ Page() { <div id=\"note-content\"></div> }\n"
	writeTestFile(t, filepath.Join(appRoot, "notes", "page.templ"), notesPageTemplate)
	writeTestFile(t, filepath.Join(appRoot, "note", "[slug]", "page.templ"), notePageTemplate)
	writeTestFile(t, filepath.Join(appRoot, "layout.templ"), "package appsrc\n")

	notesTypes := "package notes\n\n" +
		"type PageView struct{}\n" +
		"type LiveState struct{}\n"
	noteTypes := "package param_slug\n\n" +
		"type PageView struct{}\n"
	writeTestFile(t, filepath.Join(resolverRoot, "notes", "types.go"), notesTypes)
	writeTestFile(t, filepath.Join(resolverRoot, "note", "param_slug", "types.go"), noteTypes)

	routes, err := discoverRouteFiles(appRoot, genRoot)
	if err != nil {
		t.Fatalf("discover routes: %v", err)
	}

	metas, err := buildRouteMetas(routes.Pages, generationPaths{
		GenImportRoot:      "internal/web/gen",
		ResolverRoot:       resolverRoot,
		ResolverImportRoot: "internal/web/appcore/resolvers",
	})
	if err != nil {
		t.Fatalf("build route metas: %v", err)
	}

	registry, err := generateRegistrySource(
		generationPaths{GenImportRoot: "internal/web/gen"},
		metas,
		routes.Layouts,
	)
	if err != nil {
		t.Fatalf("generate registry: %v", err)
	}

	if !bytes.Contains(registry, []byte("\"/notes/live\"")) {
		t.Fatalf("expected notes live route in registry:\n%s", string(registry))
	}
	if bytes.Contains(registry, []byte("\"/note/[slug]/live\"")) {
		t.Fatalf("did not expect note live route in registry:\n%s", string(registry))
	}
}

func TestContractsGenerationDeterministic(t *testing.T) {
	metas := []routeMeta{
		{
			RouteID:            "notes",
			RouteName:          "Notes",
			ParamsTypeName:     "NotesParams",
			ResolverAlias:      "rr_notes",
			ResolverImportPath: "internal/web/appcore/resolvers/notes",
			HasLive:            true,
		},
		{
			RouteID:            "note/[slug]",
			RouteName:          "NoteParamSlug",
			ParamsTypeName:     "NoteParamSlugParams",
			Params:             []routeParamDef{{Name: "slug", FieldName: "Slug"}},
			ResolverAlias:      "rr_note_param_slug",
			ResolverImportPath: "internal/web/appcore/resolvers/note/param_slug",
		},
	}

	first, err := generateContractsSource(metas)
	if err != nil {
		t.Fatalf("first generation failed: %v", err)
	}
	second, err := generateContractsSource(metas)
	if err != nil {
		t.Fatalf("second generation failed: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("contracts generation is not deterministic")
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
