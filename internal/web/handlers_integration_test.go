package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"blog/internal/imageloader"
	"blog/internal/notes"
	"blog/internal/robots"
	"blog/internal/seo"
	webgen "blog/internal/web/gen"
	webi18n "blog/internal/web/i18n"
	"blog/internal/web/runtime"
	"github.com/Khan/genqlient/graphql"
	"github.com/RevoTale/no-js/bundler/staticassets"
	"github.com/RevoTale/no-js/framework/httpserver"
	frameworki18n "github.com/RevoTale/no-js/framework/i18n"
)

const testRootURL = "https://revotale.com/blog/notes"
const testLovelyEyeTrackerURL = "https://analytics.example/tracker.js"
const testLovelyEyeSiteID = "site-key-123"

type fakeGraphQLClient struct{}

func (fakeGraphQLClient) MakeRequest(
	_ context.Context,
	req *graphql.Request,
	resp *graphql.Response,
) error {
	if err := requireLocaleVariables(req); err != nil {
		return err
	}

	slug := requestVarString(req, "slug")
	name := requestVarString(req, "name")
	queryValue := requestVarString(req, "query")

	switch req.OpName {
	case "AvailableTagsByPostType":
		return decodeGraphQLData(resp, `{
			"availableTagsByMicroPostType": [
				{"id":"tag-1","name":"go","title":"Go"},
				{"id":"tag-2","name":"rust","title":"Rust"}
			]
		}`)
	case "AvailableAuthors":
		return decodeGraphQLData(resp, `{
			"Authors": {
				"docs": [
					{"id":"author-1","name":"L You","slug":"l-you","bio":"writer"},
					{"id":"author-2","name":"Zed","slug":"zed","bio":"guest"}
				]
			}
		}`)
	case "TagIDsByNames":
		return decodeGraphQLData(resp, `{
			"Tags": {
				"docs": [
					{"id":"tag-1","name":"go","title":"Go"},
					{"id":"tag-2","name":"rust","title":"Rust"}
				]
			}
		}`)
	case "TagByName":
		if name == "missing" {
			return decodeGraphQLData(resp, `{"Tags": {"docs": []}}`)
		}
		if name == "rust" {
			return decodeGraphQLData(resp, `{
				"Tags": {"docs": [{"id":"tag-2","name":"rust","title":"Rust"}]}
			}`)
		}
		return decodeGraphQLData(resp, `{
			"Tags": {"docs": [{"id":"tag-1","name":"go","title":"Go"}]}
		}`)
	case "ListNotes":
		fallthrough
	case "ListNotesByType":
		fallthrough
	case "ListNotesByTagIDs":
		fallthrough
	case "ListNotesByTagIDsAndType":
		fallthrough
	case "ListNotesByAuthorAndTagIDs":
		fallthrough
	case "ListNotesByAuthorTagIDsAndType":
		fallthrough
	case "SearchNotes":
		fallthrough
	case "SearchNotesByType":
		fallthrough
	case "SearchNotesByTagIDs":
		fallthrough
	case "SearchNotesByTagIDsAndType":
		fallthrough
	case "SearchNotesByAuthorAndTagIDs":
		fallthrough
	case "SearchNotesByAuthorTagIDsAndType":
		if queryValue == "nomatch" {
			return decodeGraphQLData(resp, `{
				"Micro_posts": {
					"totalPages": 1,
					"docs": []
				}
			}`)
		}
		return decodeGraphQLData(resp, `{
			"Micro_posts": {
				"totalPages": 2,
				"docs": [
					{
						"id": "note-1",
						"slug": "hello-world",
						"title": "Hello World",
						"content": "# Hello",
						"publishedAt": "2024-01-02T00:00:00.000Z",
						"authors": [{"name":"L You","slug":"l-you","bio":"writer"}],
						"tags": [{"id":"tag-1","name":"go","title":"Go"}],
						"externalLinks": [{"id":"ext-1","target_url":"https://example.com/docs"}],
						"linkedMicroPosts": [{"id":"linked-1","slug":"hello-linked"}],
						"meta": {
							"title":"Hello World Meta",
							"description":"hello note",
							"image":{"url":"/images/meta-hello.webp","description":"hello image","width":1200,"height":630}
						}
					}
				]
			}
		}`)
	case "NotesByAuthorSlug":
		fallthrough
	case "SearchNotesByAuthorSlug":
		if queryValue == "nomatch" {
			return decodeGraphQLData(resp, `{"Micro_posts": {"totalPages": 1, "docs": []}}`)
		}
		if slug == "missing" {
			return decodeGraphQLData(resp, `{"Micro_posts": {"totalPages": 1, "docs": []}}`)
		}
		return decodeGraphQLData(resp, `{
			"Micro_posts": {
				"totalPages": 1,
				"docs": [
					{
						"id": "note-1",
						"slug": "hello-world",
						"title": "Hello World",
						"content": "# Hello",
						"publishedAt": "2024-01-02T00:00:00.000Z",
						"authors": [{"name":"L You","slug":"l-you","bio":"writer"}],
						"tags": [{"id":"tag-1","name":"go","title":"Go"}],
						"externalLinks": [{"id":"ext-1","target_url":"https://example.com/docs"}],
						"linkedMicroPosts": [{"id":"linked-1","slug":"hello-linked"}],
						"meta": {
							"title":"Hello World Meta",
							"description":"hello note",
							"image":{"url":"/images/meta-hello.webp","description":"hello image","width":1200,"height":630}
						}
					}
				]
			}
		}`)
	case "NotesByAuthorSlugAndType":
		fallthrough
	case "SearchNotesByAuthorSlugAndType":
		if queryValue == "nomatch" {
			return decodeGraphQLData(resp, `{"Micro_posts": {"totalPages": 1, "docs": []}}`)
		}
		if slug == "missing" {
			return decodeGraphQLData(resp, `{"Micro_posts": {"totalPages": 1, "docs": []}}`)
		}
		return decodeGraphQLData(resp, `{
			"Micro_posts": {
				"totalPages": 1,
				"docs": [
					{
						"id": "note-1",
						"slug": "hello-world",
						"title": "Hello World",
						"content": "# Hello",
						"publishedAt": "2024-01-02T00:00:00.000Z",
						"authors": [{"name":"L You","slug":"l-you","bio":"writer"}],
						"tags": [{"id":"tag-1","name":"go","title":"Go"}],
						"externalLinks": [{"id":"ext-1","target_url":"https://example.com/docs"}],
						"linkedMicroPosts": [{"id":"linked-1","slug":"hello-linked"}],
						"meta": {
							"title":"Hello World Meta",
							"description":"hello note",
							"image":{"url":"/images/meta-hello.webp","description":"hello image","width":1200,"height":630}
						}
					}
				]
			}
		}`)
	case "NoteBySlug":
		if slug == "missing" {
			return decodeGraphQLData(resp, `{"Micro_posts": {"docs": []}}`)
		}
		return decodeGraphQLData(resp, `{
			"Micro_posts": {
				"docs": [
					{
						"id": "note-1",
						"slug": "hello-world",
						"title": "Hello World",
						"content": "# Hello",
						"publishedAt": "2024-01-02T00:00:00.000Z",
						"authors": [{"name":"L You","slug":"l-you","bio":"writer"}],
						"tags": [{"id":"tag-1","name":"go","title":"Go"}],
						"externalLinks": [{"id":"ext-1","target_url":"https://example.com/docs"}],
						"linkedMicroPosts": [{"id":"linked-1","slug":"hello-linked"}],
						"meta": {
							"title":"Hello World",
							"description":"hello note",
							"image":{"url":"/images/meta-hello.webp","description":"hello image","width":1200,"height":630}
						}
					}
				]
			}
		}`)
	case "AuthorBySlug":
		if slug == "missing" {
			return decodeGraphQLData(resp, `{"Authors": {"docs": []}}`)
		}
		if slug == "zed" {
			return decodeGraphQLData(resp, `{
				"Authors": {
					"docs": [
						{"id":"author-2","name":"Zed","slug":"zed","bio":"guest"}
					]
				}
			}`)
		}
		return decodeGraphQLData(resp, `{
			"Authors": {
				"docs": [
					{"id":"author-1","name":"L You","slug":"l-you","bio":"writer"}
				]
			}
		}`)
	default:
		return decodeGraphQLData(resp, `{}`)
	}
}

func decodeGraphQLData(resp *graphql.Response, payload string) error {
	return json.Unmarshal([]byte(payload), resp.Data)
}

func requestVarString(req *graphql.Request, key string) string {
	if req == nil || req.Variables == nil {
		return ""
	}

	raw, err := json.Marshal(req.Variables)
	if err != nil {
		return ""
	}

	values := make(map[string]json.RawMessage)
	if err := json.Unmarshal(raw, &values); err != nil {
		return ""
	}

	entry, ok := values[key]
	if !ok {
		return ""
	}

	var value string
	if err := json.Unmarshal(entry, &value); err != nil {
		return ""
	}
	return strings.TrimSpace(value)
}

var operationsWithLocaleAndFallback = map[string]struct{}{
	"AuthorBySlug":                     {},
	"AvailableAuthors":                 {},
	"ListNotes":                        {},
	"ListNotesByType":                  {},
	"ListNotesByTagIDs":                {},
	"ListNotesByTagIDsAndType":         {},
	"ListNotesByAuthorAndTagIDs":       {},
	"ListNotesByAuthorTagIDsAndType":   {},
	"NoteBySlug":                       {},
	"NotesByAuthorSlug":                {},
	"NotesByAuthorSlugAndType":         {},
	"SearchNotes":                      {},
	"SearchNotesByType":                {},
	"SearchNotesByTagIDs":              {},
	"SearchNotesByTagIDsAndType":       {},
	"SearchNotesByAuthorSlug":          {},
	"SearchNotesByAuthorSlugAndType":   {},
	"SearchNotesByAuthorAndTagIDs":     {},
	"SearchNotesByAuthorTagIDsAndType": {},
	"TagByName":                        {},
	"TagIDsByNames":                    {},
}

var allowedGraphQLLocales = map[string]struct{}{
	"en_US": {},
	"de_DE": {},
	"uk_UA": {},
	"hi_IN": {},
	"ru_RU": {},
	"ja_JP": {},
	"fr_FR": {},
	"es_ES": {},
}

func requireLocaleVariables(req *graphql.Request) error {
	if req == nil {
		return nil
	}

	if req.OpName == "AvailableTagsByPostType" {
		locale := requestVarString(req, "locale")
		if locale == "" {
			return fmt.Errorf("missing locale variable for %s", req.OpName)
		}
		if _, ok := allowedGraphQLLocales[locale]; !ok {
			return fmt.Errorf("unexpected locale variable %q for %s", locale, req.OpName)
		}
		return nil
	}

	if _, ok := operationsWithLocaleAndFallback[req.OpName]; !ok {
		return nil
	}

	locale := requestVarString(req, "locale")
	if locale == "" {
		return fmt.Errorf("missing locale variable for %s", req.OpName)
	}
	if _, ok := allowedGraphQLLocales[locale]; !ok {
		return fmt.Errorf("unexpected locale variable %q for %s", locale, req.OpName)
	}

	fallbackLocale := requestVarString(req, "fallbackLocale")
	if fallbackLocale == "" {
		return fmt.Errorf("missing fallbackLocale variable for %s", req.OpName)
	}
	if fallbackLocale != "en_US" {
		return fmt.Errorf("unexpected fallbackLocale variable %q for %s", fallbackLocale, req.OpName)
	}
	return nil
}

type testServer struct {
	handler http.Handler
	bundle  *staticassets.Bundle
}

type testServerBootstrap struct {
	cfg           webgen.ServerConfig
	bundle        *staticassets.Bundle
	notes         *notes.Service
	i18nConfig    frameworki18n.Config
	cachePolicies httpserver.CachePolicies
}

type testServerOptions struct {
	enableImageLoader  bool
	lovelyEyeScriptURL string
	lovelyEyeSiteID    string
}

func newTestServer(t *testing.T) testServer {
	return newTestServerWithOptions(t, testServerOptions{})
}

func newTestServerWithImageLoader(t *testing.T, enableImageLoader bool) testServer {
	return newTestServerWithOptions(t, testServerOptions{
		enableImageLoader: enableImageLoader,
	})
}

func newTestServerWithLovelyEye(t *testing.T, scriptURL string, siteID string) testServer {
	t.Helper()

	return newTestServerWithOptions(t, testServerOptions{
		lovelyEyeScriptURL: scriptURL,
		lovelyEyeSiteID:    siteID,
	})
}

func newTestServerWithOptions(t *testing.T, options testServerOptions) testServer {
	t.Helper()

	bootstrap := newTestServerBootstrap(t, options)

	handler, err := webgen.NewHandler(bootstrap.cfg)
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}
	handler = runtime.WithCanonicalNotesRedirects(handler)
	handler = seo.WithFeedAndSitemapEndpoints(handler, seo.FeedAndSitemapConfig{
		RootURL:    testRootURL,
		I18nConfig: bootstrap.i18nConfig,
		Notes:      bootstrap.notes,
	})
	handler = robots.WithRobotsEndpoint(handler, testRootURL, bootstrap.cachePolicies.HTML)

	return testServer{
		handler: handler,
		bundle:  bootstrap.bundle,
	}
}

func newTestServerBootstrap(t *testing.T, options testServerOptions) testServerBootstrap {
	t.Helper()

	bundle, err := staticassets.Build(staticassets.BuildConfig{
		SourceDir: "../../internal/web/static",
		URLPrefix: "/.revotale/",
	})
	if err != nil {
		t.Fatalf("build static assets: %v", err)
	}
	t.Cleanup(func() {
		if cleanupErr := bundle.Cleanup(); cleanupErr != nil {
			t.Fatalf("cleanup static assets: %v", cleanupErr)
		}
	})

	manifestPath := filepath.Join(bundle.Dir(), "manifest.json")
	if err := staticassets.WriteManifest(manifestPath, bundle.Manifest()); err != nil {
		t.Fatalf("write static manifest: %v", err)
	}

	i18nConfig, err := frameworki18n.NormalizeConfig(webi18n.Config())
	if err != nil {
		t.Fatalf("normalize i18n config: %v", err)
	}
	i18nCatalog, err := webi18n.LoadCatalog()
	if err != nil {
		t.Fatalf("load i18n catalog: %v", err)
	}
	imageLoader := imageloader.New(options.enableImageLoader)
	noteService := notes.NewService(fakeGraphQLClient{}, 12, testRootURL, imageLoader)
	cachePolicies := httpserver.DefaultCachePolicies()

	return testServerBootstrap{
		cfg: webgen.ServerConfig{
			AppContext: runtime.NewContext(
				noteService,
				i18nConfig,
				i18nCatalog,
				testRootURL,
				options.lovelyEyeScriptURL,
				options.lovelyEyeSiteID,
			),
			Runtime: runtime.BootstrapConfig{
				LocalizationConfig:  i18nConfig,
				StaticAssetBasePath: bundle.URLPrefix(),
				ImageLoader:         imageLoader,
				LovelyEyeScriptURL:  options.lovelyEyeScriptURL,
				LovelyEyeSiteID:     options.lovelyEyeSiteID,
			},
			StaticManifestPath: manifestPath,
			PublicDir:          "../../internal/web/public",
			CachePolicies:      cachePolicies,
			LogServerError: func(error) {
			},
		},
		bundle:        bundle,
		notes:         noteService,
		i18nConfig:    i18nConfig,
		cachePolicies: cachePolicies,
	}
}

func requireBody(t *testing.T, body io.Reader) string {
	t.Helper()

	content, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(content)
}

func performRequest(mux http.Handler, method string, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func performRequestWithHeaders(
	mux http.Handler,
	method string,
	path string,
	headers map[string]string,
) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

var jsonLDScriptRe = regexp.MustCompile(`(?s)<script type="application/ld\+json">(.*?)</script>`)

func parseJSONLDScripts(t *testing.T, html string) []map[string]any {
	t.Helper()

	matches := jsonLDScriptRe.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		return nil
	}

	out := make([]map[string]any, 0, len(matches))
	for idx, match := range matches {
		if len(match) < 2 {
			continue
		}

		var doc map[string]any
		if err := json.Unmarshal([]byte(match[1]), &doc); err != nil {
			t.Fatalf("parse json-ld script[%d]: %v", idx, err)
		}
		out = append(out, doc)
	}
	return out
}

func requireJSONLDDocByType(t *testing.T, docs []map[string]any, typeName string) map[string]any {
	t.Helper()

	for _, doc := range docs {
		if strings.TrimSpace(stringField(t, doc, "@type")) == strings.TrimSpace(typeName) {
			return doc
		}
	}
	t.Fatalf("expected JSON-LD document with @type=%q", typeName)
	return nil
}

func stringField(t *testing.T, object map[string]any, key string) string {
	t.Helper()

	value, ok := object[key]
	if !ok {
		t.Fatalf("missing field %q", key)
	}
	text, ok := value.(string)
	if !ok {
		t.Fatalf("field %q should be string, got %T", key, value)
	}
	return text
}

func objectField(t *testing.T, object map[string]any, key string) map[string]any {
	t.Helper()

	value, ok := object[key]
	if !ok {
		t.Fatalf("missing field %q", key)
	}
	out, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("field %q should be object, got %T", key, value)
	}
	return out
}

func arrayField(t *testing.T, object map[string]any, key string) []any {
	t.Helper()

	value, ok := object[key]
	if !ok {
		t.Fatalf("missing field %q", key)
	}
	out, ok := value.([]any)
	if !ok {
		t.Fatalf("field %q should be array, got %T", key, value)
	}
	return out
}

func objectFromAny(t *testing.T, value any, field string) map[string]any {
	t.Helper()

	out, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("field %s should be object, got %T", field, value)
	}
	return out
}

func TestHandlerPageRoutesRenderHTML(t *testing.T) {
	t.Parallel()
	testSrv := newTestServer(t)
	mux := testSrv.handler
	rootTitleToken := "Notes - Quick Coding, Experience, Open Source, SEO &amp; Science Insights | RevoTale</title>"

	cases := []struct {
		path        string
		mustContain string
	}{
		{path: "/channels", mustContain: "Channels | RevoTale</title>"},
		{path: "/", mustContain: rootTitleToken},
		{path: "/?q=hello", mustContain: rootTitleToken},
		{path: "/?author=l-you&tag=go&type=short", mustContain: "<h1>L You</h1>"},
		{path: "/note/hello-world", mustContain: "Hello World | RevoTale</title>"},
		{path: "/author/l-you", mustContain: "L You | Author | RevoTale</title>"},
		{path: "/tag/go", mustContain: "#Go | RevoTale</title>"},
		{path: "/tales", mustContain: "Tales | RevoTale</title>"},
		{path: "/micro-tales", mustContain: "Micro-tales | RevoTale</title>"},
	}

	for _, tc := range cases {
		rec := performRequest(mux, http.MethodGet, tc.path)

		if rec.Code != http.StatusOK {
			t.Fatalf("%s status: expected %d, got %d", tc.path, http.StatusOK, rec.Code)
		}

		if contentType := rec.Header().Get("Content-Type"); !strings.Contains(contentType, "text/html") {
			t.Fatalf("%s content-type: expected html, got %q", tc.path, contentType)
		}

		body := requireBody(t, rec.Body)
		if !strings.Contains(body, tc.mustContain) {
			t.Fatalf("%s body missing %q", tc.path, tc.mustContain)
		}
		if strings.Contains(body, "event: datastar-patch-elements") {
			t.Fatalf("%s should not include live SSE patch payload", tc.path)
		}
	}
}

func TestCanonicalListingQueryRedirects(t *testing.T) {
	t.Parallel()
	testSrv := newTestServer(t)
	mux := testSrv.handler

	cases := []struct {
		path     string
		location string
	}{
		{path: "/?author=l-you", location: "/author/l-you"},
		{path: "/?tag=go", location: "/tag/go"},
		{path: "/?type=long", location: "/tales"},
		{path: "/?type=short", location: "/micro-tales"},
		{path: "/?author=l-you&page=2", location: "/author/l-you?page=2"},
		{path: "/author/l-you?author=zed", location: "/author/l-you"},
		{path: "/tag/go?tag=rust", location: "/tag/go"},
		{path: "/tales?type=short", location: "/tales"},
		{path: "/uk?author=l-you", location: "/uk/author/l-you"},
	}

	for _, tc := range cases {
		rec := performRequest(mux, http.MethodGet, tc.path)
		if rec.Code != http.StatusPermanentRedirect {
			t.Fatalf("%s status: expected %d, got %d", tc.path, http.StatusPermanentRedirect, rec.Code)
		}
		if location := rec.Header().Get("Location"); location != tc.location {
			t.Fatalf("%s redirect: expected %q, got %q", tc.path, tc.location, location)
		}
	}
}

func TestRobotsRulesWithAndWithoutQuery(t *testing.T) {
	t.Parallel()
	testSrv := newTestServer(t)
	mux := testSrv.handler

	cases := []struct {
		path           string
		expectedRobots string
	}{
		{path: "/", expectedRobots: "index, follow"},
		{path: "/tales", expectedRobots: "index, follow"},
		{path: "/micro-tales", expectedRobots: "index, follow"},
		{path: "/tag/go", expectedRobots: "index, follow"},
		{path: "/author/l-you", expectedRobots: "index, follow"},
		{path: "/author/l-you?page=2", expectedRobots: "index, follow"},
		{path: "/?q=hello", expectedRobots: "noindex, follow"},
		{path: "/?author=l-you&tag=go", expectedRobots: "noindex, follow"},
		{path: "/note/hello-world?utm_source=test", expectedRobots: "noindex, follow"},
	}

	for _, tc := range cases {
		rec := performRequest(mux, http.MethodGet, tc.path)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status: expected %d, got %d", tc.path, http.StatusOK, rec.Code)
		}
		body := requireBody(t, rec.Body)
		expectedTag := `name="robots" content="` + tc.expectedRobots + `"`
		if !strings.Contains(body, expectedTag) {
			t.Fatalf("%s should include robots tag %q", tc.path, expectedTag)
		}
	}
}

func TestUnknownListingQueryParamsStayNoIndexWithoutCanonicalRedirect(t *testing.T) {
	t.Parallel()
	testSrv := newTestServer(t)
	mux := testSrv.handler

	cases := []string{
		"/?utm_source=test",
		"/?author=l-you&utm_source=test",
		"/tag/go?utm_source=test",
	}

	for _, path := range cases {
		rec := performRequest(mux, http.MethodGet, path)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status: expected %d, got %d", path, http.StatusOK, rec.Code)
		}

		body := requireBody(t, rec.Body)
		if !strings.Contains(body, `name="robots" content="noindex, follow"`) {
			t.Fatalf("%s should include noindex robots metadata", path)
		}
	}
}

func TestSidebarLinkBehavior(t *testing.T) {
	t.Parallel()
	testSrv := newTestServer(t)
	mux := testSrv.handler

	root := performRequest(mux, http.MethodGet, "/")
	rootBody := requireBody(t, root.Body)
	if !strings.Contains(rootBody, `href="/channels"`) {
		t.Fatalf("root page missing channels button link")
	}
	if !strings.Contains(rootBody, `href="/author/l-you"`) {
		t.Fatalf("root notes missing canonical author link")
	}
	if !strings.Contains(rootBody, `href="/tag/go"`) {
		t.Fatalf("root notes missing canonical tag link")
	}
	if !strings.Contains(rootBody, `class="topbar-rss-link" href="/feed.xml?locale=en"`) {
		t.Fatalf("root page should include rss link to local feed endpoint")
	}
	if !strings.Contains(rootBody, `href="/tales"`) {
		t.Fatalf("root notes missing tales route link")
	}
	if !strings.Contains(rootBody, `href="/micro-tales"`) {
		t.Fatalf("root notes missing micro-tales route link")
	}
	if strings.Contains(rootBody, `href="/?author=`) {
		t.Fatalf("root notes should not render author # All clear link when no author filter")
	}
	if strings.Contains(rootBody, `href="/?tag=`) {
		t.Fatalf("root notes should not render tag # All clear link when no tag filter")
	}
	if strings.Contains(rootBody, `topbar-search-clear`) {
		t.Fatalf("root page should not render search clear action when q is empty")
	}

	search := performRequest(mux, http.MethodGet, "/?q=hello")
	searchBody := requireBody(t, search.Body)
	if !strings.Contains(searchBody, `<form class="topbar-search" role="search" method="get" action="/">`) {
		t.Fatalf("search page should render topbar search form")
	}
	if !strings.Contains(searchBody, `name="q"`) || !strings.Contains(searchBody, `value="hello"`) {
		t.Fatalf("search page should preserve q value in search input")
	}
	if !strings.Contains(searchBody, `class="topbar-rss-link" href="/feed.xml?locale=en&amp;q=hello"`) {
		t.Fatalf("search page should preserve q in rss feed link")
	}
	if !strings.Contains(searchBody, `href="/channels?q=hello"`) {
		t.Fatalf("search page should preserve q in channels link")
	}
	if !strings.Contains(searchBody, `href="/?author=l-you&amp;q=hello"`) {
		t.Fatalf("search page should preserve q in author links")
	}
	if !strings.Contains(searchBody, `href="/?q=hello&amp;tag=go"`) {
		t.Fatalf("search page should preserve q in tag links")
	}
	if !strings.Contains(searchBody, `class="topbar-search-clear"`) {
		t.Fatalf("search page should render search clear action when q is present")
	}
	if !strings.Contains(searchBody, `class="topbar-search-clear" href="/"`) {
		t.Fatalf("search clear action should reset to root when only q is active")
	}
	if !strings.Contains(
		searchBody,
		`rel="alternate" type="application/rss+xml" href="https://revotale.com/blog/notes/feed.xml?locale=en&amp;q=hello"`,
	) {
		t.Fatalf("search page should include filtered rss alternate metadata")
	}

	filtered := performRequest(mux, http.MethodGet, "/author/l-you?tag=go&type=short")
	filteredBody := requireBody(t, filtered.Body)
	if !strings.Contains(filteredBody, `href="/channels?author=l-you&amp;tag=go&amp;type=short"`) {
		t.Fatalf("filtered page missing carried channels button link")
	}
	if !strings.Contains(filteredBody, `href="/"`) {
		t.Fatalf("filtered page missing All link to /")
	}
	if !strings.Contains(filteredBody, `href="/?tag=go&amp;type=short"`) {
		t.Fatalf("filtered page missing ANY author clear link")
	}
	if !strings.Contains(filteredBody, `href="/?author=l-you&amp;type=short"`) {
		t.Fatalf("filtered page missing ANY tag clear link")
	}
	if !strings.Contains(filteredBody, `href="/?author=l-you&amp;tag=go"`) {
		t.Fatalf("filtered page missing ANY type clear link")
	}
	if !strings.Contains(
		filteredBody,
		`class="topbar-rss-link" href="/feed.xml?author=l-you&amp;locale=en&amp;tag=go&amp;type=short"`,
	) {
		t.Fatalf("filtered page should preserve author/tag/type in rss feed link")
	}
	if !strings.Contains(filteredBody, `href="/?author=zed&amp;tag=go&amp;type=short"`) {
		t.Fatalf("filtered page missing merged author link")
	}
	if !strings.Contains(filteredBody, `href="/?author=l-you&amp;tag=rust&amp;type=short"`) {
		t.Fatalf("filtered page missing merged tag link")
	}
	if !strings.Contains(filteredBody, `href="/?author=l-you&amp;tag=go&amp;type=long"`) {
		t.Fatalf("filtered page missing merged tales type link")
	}
	if !strings.Contains(filteredBody, `href="/?tag=go&amp;type=short"`) {
		t.Fatalf("filtered page should render author # All clear link")
	}
	if !strings.Contains(filteredBody, `href="/?author=l-you&amp;type=short"`) {
		t.Fatalf("filtered page should render tag # All clear link")
	}

	channelsFiltered := performRequest(mux, http.MethodGet, "/channels?author=l-you&tag=go&type=short")
	channelsFilteredBody := requireBody(t, channelsFiltered.Body)
	if !strings.Contains(channelsFilteredBody, `href="/?tag=go&amp;type=short"`) {
		t.Fatalf("channels page missing author clear link")
	}
	if !strings.Contains(channelsFilteredBody, `href="/?author=zed&amp;tag=go&amp;type=short"`) {
		t.Fatalf("channels page missing merged author link")
	}
	if !strings.Contains(channelsFilteredBody, `channels-desktop-hint`) {
		t.Fatalf("channels page missing desktop hint block")
	}
	if !strings.Contains(channelsFilteredBody, `channels-mobile-panel`) {
		t.Fatalf("channels page missing mobile panel block")
	}

	channelsSingle := performRequest(mux, http.MethodGet, "/channels?author=l-you")
	channelsSingleBody := requireBody(t, channelsSingle.Body)
	if !strings.Contains(channelsSingleBody, `class="back-link channels-back-button" href="/author/l-you"`) {
		t.Fatalf("channels page back button should use canonical author landing for a single filter")
	}
}

func TestI18nRoutingAndLocalizedURLs(t *testing.T) {
	t.Parallel()
	testSrv := newTestServer(t)
	mux := testSrv.handler

	recUK := performRequest(mux, http.MethodGet, "/uk")
	if recUK.Code != http.StatusOK {
		t.Fatalf("/uk status: expected %d, got %d", http.StatusOK, recUK.Code)
	}
	ukBody := requireBody(t, recUK.Body)
	if !strings.Contains(strings.ToLower(ukBody), "<!doctype html>") {
		t.Fatalf("/uk should render root document doctype")
	}
	if !strings.Contains(ukBody, `<html lang="uk">`) {
		t.Fatalf("/uk should render localized html lang")
	}
	headIndex := strings.Index(ukBody, "<head>")
	mainIndex := strings.Index(ukBody, "<main")
	if headIndex < 0 || mainIndex < 0 || headIndex > mainIndex {
		t.Fatalf("/uk should render head before body content")
	}
	if !strings.Contains(ukBody, `href="/uk/channels"`) {
		t.Fatalf("/uk should render localized channels URL")
	}
	if !strings.Contains(ukBody, `href="/uk/author/l-you"`) {
		t.Fatalf("/uk should render localized author URL")
	}
	if !strings.Contains(ukBody, `href="/uk/tag/go"`) {
		t.Fatalf("/uk should render localized tag URL")
	}
	if !strings.Contains(ukBody, `href="/uk/note/hello-world"`) {
		t.Fatalf("/uk should render localized note URL")
	}

	recUKNote := performRequest(mux, http.MethodGet, "/uk/note/hello-world")
	if recUKNote.Code != http.StatusOK {
		t.Fatalf("/uk/note status: expected %d, got %d", http.StatusOK, recUKNote.Code)
	}
	ukNoteBody := requireBody(t, recUKNote.Body)
	if !strings.Contains(ukNoteBody, `href="/uk"`) {
		t.Fatalf("/uk/note should keep back-link localized")
	}

	recDefaultPrefixed := performRequest(mux, http.MethodGet, "/en/note/hello-world")
	if recDefaultPrefixed.Code != http.StatusPermanentRedirect {
		t.Fatalf("/en/note status: expected %d, got %d", http.StatusPermanentRedirect, recDefaultPrefixed.Code)
	}
	if location := recDefaultPrefixed.Header().Get("Location"); location != "/note/hello-world" {
		t.Fatalf("/en/note redirect: expected %q, got %q", "/note/hello-world", location)
	}

	recUnknownLocale := performRequest(mux, http.MethodGet, "/it/note/hello-world")
	if recUnknownLocale.Code != http.StatusNotFound {
		t.Fatalf("/it/note status: expected %d, got %d", http.StatusNotFound, recUnknownLocale.Code)
	}
}

func TestHandlerHTMXRoutesReturnPartial(t *testing.T) {
	t.Parallel()
	testSrv := newTestServer(t)
	mux := testSrv.handler

	cases := []struct {
		path        string
		mustContain string
	}{
		{path: "/", mustContain: "<section class=\"context-panel\">"},
		{path: "/author/l-you", mustContain: "<section class=\"context-panel\">"},
		{path: "/tag/go", mustContain: "<section class=\"context-panel\">"},
		{path: "/tales", mustContain: "<section class=\"context-panel\">"},
		{path: "/micro-tales", mustContain: "<section class=\"context-panel\">"},
	}

	for _, tc := range cases {
		rec := performRequestWithHeaders(mux, http.MethodGet, tc.path, map[string]string{
			"HX-Request": "true",
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status: expected %d, got %d", tc.path, http.StatusOK, rec.Code)
		}
		if got := rec.Header().Get("Cache-Control"); got != httpserver.DefaultCachePolicies().Live {
			t.Fatalf("%s cache policy: expected %q, got %q", tc.path, httpserver.DefaultCachePolicies().Live, got)
		}

		body := requireBody(t, rec.Body)
		if !strings.Contains(body, tc.mustContain) {
			t.Fatalf("%s body missing %q", tc.path, tc.mustContain)
		}
		if strings.Contains(body, "<title>") {
			t.Fatalf("%s should return partial HTMX payload without layout title", tc.path)
		}
		if strings.Contains(body, `application/ld+json`) {
			t.Fatalf("%s should not include structured data scripts in HTMX partial payload", tc.path)
		}
	}
}

func TestHandlerSEOMetadataAndHTMXPatchHeaders(t *testing.T) {
	t.Parallel()
	testSrv := newTestServer(t)
	mux := testSrv.handler

	recNote := performRequest(mux, http.MethodGet, "/uk/note/hello-world")
	if recNote.Code != http.StatusOK {
		t.Fatalf("note status: expected %d, got %d", http.StatusOK, recNote.Code)
	}
	noteBody := requireBody(t, recNote.Body)
	if !strings.Contains(noteBody, `rel="canonical" href="https://revotale.com/blog/notes/uk/note/hello-world"`) {
		t.Fatalf("note page missing canonical link")
	}
	if strings.Contains(noteBody, "__live=navigation") {
		t.Fatalf("note canonical/hreflang should not include __live marker")
	}
	if !strings.Contains(noteBody, `rel="alternate" hreflang="en"`) {
		t.Fatalf("note page missing hreflang=en")
	}
	if !strings.Contains(noteBody, `property="og:title"`) {
		t.Fatalf("note page missing Open Graph title metadata")
	}
	if !strings.Contains(noteBody, `property="og:url" content="https://revotale.com/blog/notes/uk/note/hello-world"`) {
		t.Fatalf("note page should publish canonical Open Graph url")
	}
	if !strings.Contains(noteBody, `name="twitter:card"`) {
		t.Fatalf("note page missing twitter metadata")
	}
	if !strings.Contains(noteBody, `property="article:published_time"`) {
		t.Fatalf("note page should include article:published_time Open Graph metadata")
	}
	if !strings.Contains(noteBody, `property="article:author" content="https://revotale.com/blog/notes/uk/author/l-you"`) {
		t.Fatalf("note page should include article:author Open Graph metadata")
	}
	if !strings.Contains(noteBody, `property="article:tag" content="Go"`) {
		t.Fatalf("note page should include article:tag Open Graph metadata")
	}
	if !strings.Contains(noteBody, `class="topbar-rss-link" href="/feed.xml?locale=uk"`) {
		t.Fatalf("note page should include locale-aware rss link")
	}
	noteDocs := parseJSONLDScripts(t, noteBody)
	noteDoc := requireJSONLDDocByType(t, noteDocs, "BlogPosting")
	if got := stringField(t, noteDoc, "url"); got != "https://revotale.com/blog/notes/uk/note/hello-world" {
		t.Fatalf("note JSON-LD url: expected canonical note URL, got %q", got)
	}
	mainEntity := objectField(t, noteDoc, "mainEntityOfPage")
	if got := stringField(t, mainEntity, "@id"); got != "https://revotale.com/blog/notes/uk/note/hello-world" {
		t.Fatalf("note JSON-LD mainEntityOfPage.@id: expected canonical note URL, got %q", got)
	}
	publisher := objectField(t, noteDoc, "publisher")
	if got := stringField(t, publisher, "url"); got != "https://revotale.com/blog/notes" {
		t.Fatalf("note JSON-LD publisher.url: expected root URL, got %q", got)
	}
	authors := arrayField(t, noteDoc, "author")
	if len(authors) == 0 {
		t.Fatalf("note JSON-LD should include at least one author")
	}
	firstAuthor := objectFromAny(t, authors[0], "author[0]")
	if got := stringField(t, firstAuthor, "url"); got != "https://revotale.com/blog/notes/uk/author/l-you" {
		t.Fatalf("note JSON-LD author.url: expected localized author URL, got %q", got)
	}
	datePublished := stringField(t, noteDoc, "datePublished")
	if _, err := time.Parse(time.RFC3339, datePublished); err != nil {
		t.Fatalf("note JSON-LD datePublished should be RFC3339/ISO timestamp, got %q", datePublished)
	}
	mentions := arrayField(t, noteDoc, "mentions")
	if len(mentions) < 2 {
		t.Fatalf("note JSON-LD should include internal and external mentions, got %d", len(mentions))
	}
	mentionURLs := make(map[string]struct{}, len(mentions))
	for idx, mention := range mentions {
		obj := objectFromAny(t, mention, fmt.Sprintf("mentions[%d]", idx))
		mentionURLs[stringField(t, obj, "@id")] = struct{}{}
	}
	if _, ok := mentionURLs["https://example.com/docs"]; !ok {
		t.Fatalf("note JSON-LD should include external mention URL")
	}
	if _, ok := mentionURLs["https://revotale.com/blog/notes/uk/note/hello-linked"]; !ok {
		t.Fatalf("note JSON-LD should include localized internal mention URL")
	}

	recRoot := performRequest(mux, http.MethodGet, "/")
	if recRoot.Code != http.StatusOK {
		t.Fatalf("root status: expected %d, got %d", http.StatusOK, recRoot.Code)
	}

	recFavicon := performRequest(mux, http.MethodGet, "/favicon.svg")
	if recFavicon.Code != http.StatusOK {
		t.Fatalf("favicon status: expected %d, got %d", http.StatusOK, recFavicon.Code)
	}
	if got := recFavicon.Header().Get("Content-Type"); !strings.Contains(got, "image/svg+xml") {
		t.Fatalf("favicon content-type: expected svg, got %q", got)
	}

	recManifest := performRequest(mux, http.MethodGet, "/site.webmanifest")
	if recManifest.Code != http.StatusOK {
		t.Fatalf("manifest status: expected %d, got %d", http.StatusOK, recManifest.Code)
	}
	if got := recManifest.Header().Get("Content-Type"); !strings.Contains(got, "application/manifest+json") {
		t.Fatalf("manifest content-type: expected manifest+json, got %q", got)
	}

	rootBody := requireBody(t, recRoot.Body)
	expectedLogoURL := testSrv.bundle.URL("revtale-logo.svg")
	if !strings.Contains(rootBody, `class="server-logo" src="`+expectedLogoURL+`"`) {
		t.Fatalf("root page should keep original server logo src when image loader is disabled")
	}
	transformedLogoURL := imageloader.New(true).URL(expectedLogoURL, 28)
	if strings.Contains(rootBody, transformedLogoURL) {
		t.Fatalf("root page should not rewrite server logo src when image loader is disabled")
	}
	if !strings.Contains(rootBody, `rel="manifest" href="/site.webmanifest"`) {
		t.Fatalf("root page should include manifest link tag")
	}
	if !strings.Contains(rootBody, `rel="icon" href="/favicon.ico"`) {
		t.Fatalf("root page should include favicon link tag")
	}
	if !strings.Contains(rootBody, `rel="apple-touch-icon"`) {
		t.Fatalf("root page should include apple-touch-icon link tag")
	}
	if !strings.Contains(rootBody, `rel="alternate" type="application/rss+xml"`) {
		t.Fatalf("root page should include rss alternate metadata")
	}
	rootDocs := parseJSONLDScripts(t, rootBody)
	rootBlog := requireJSONLDDocByType(t, rootDocs, "Blog")
	if got := stringField(t, rootBlog, "url"); got != "https://revotale.com/blog/notes" {
		t.Fatalf("root Blog JSON-LD url: expected blog root URL, got %q", got)
	}
	blogPosts := arrayField(t, rootBlog, "blogPost")
	if len(blogPosts) == 0 {
		t.Fatalf("root Blog JSON-LD should include linked blog posts")
	}
	firstPost := objectFromAny(t, blogPosts[0], "blogPost[0]")
	if got := stringField(t, firstPost, "@type"); got != "BlogPosting" {
		t.Fatalf("root Blog JSON-LD blogPost[0].@type: expected BlogPosting, got %q", got)
	}
	if got := stringField(t, firstPost, "url"); got != "https://revotale.com/blog/notes/note/hello-world" {
		t.Fatalf("root Blog JSON-LD blogPost[0].url: expected note URL, got %q", got)
	}
	firstPostMainEntity := objectField(t, firstPost, "mainEntityOfPage")
	if got := stringField(t, firstPostMainEntity, "@id"); got != "https://revotale.com/blog/notes/note/hello-world" {
		t.Fatalf("root Blog JSON-LD blogPost[0].mainEntityOfPage.@id: expected note URL, got %q", got)
	}
	firstPostAuthors := arrayField(t, firstPost, "author")
	if len(firstPostAuthors) == 0 {
		t.Fatalf("root Blog JSON-LD blogPost[0] should include authors")
	}
	firstPostAuthor := objectFromAny(t, firstPostAuthors[0], "blogPost[0].author[0]")
	if got := stringField(t, firstPostAuthor, "url"); got != "https://revotale.com/blog/notes/author/l-you" {
		t.Fatalf("root Blog JSON-LD blogPost[0].author[0].url: expected author URL, got %q", got)
	}
	firstPostMentions := arrayField(t, firstPost, "mentions")
	if len(firstPostMentions) < 2 {
		t.Fatalf("root Blog JSON-LD blogPost[0] should include mentions, got %d", len(firstPostMentions))
	}

	recChannels := performRequest(mux, http.MethodGet, "/channels")
	if recChannels.Code != http.StatusOK {
		t.Fatalf("channels status: expected %d, got %d", http.StatusOK, recChannels.Code)
	}
	channelsBody := requireBody(t, recChannels.Body)
	if !strings.Contains(channelsBody, `name="robots" content="noindex, follow"`) {
		t.Fatalf("channels page should include noindex robots metadata")
	}
	channelsDocs := parseJSONLDScripts(t, channelsBody)
	if len(channelsDocs) != 0 {
		t.Fatalf("channels page should not include structured data scripts, got %d", len(channelsDocs))
	}

	recHTMX := performRequestWithHeaders(mux, http.MethodGet, "/?__live=navigation", map[string]string{
		"HX-Request": "true",
	})
	if recHTMX.Code != http.StatusOK {
		t.Fatalf("htmx status: expected %d, got %d", http.StatusOK, recHTMX.Code)
	}
	patchHeader := strings.TrimSpace(recHTMX.Header().Get("HX-Trigger-After-Settle"))
	if patchHeader == "" {
		t.Fatalf("htmx response should include metadata patch header")
	}
	if !strings.Contains(patchHeader, "metagen:patch") {
		t.Fatalf("htmx metadata patch header should include metagen patch event")
	}
	if strings.Contains(patchHeader, "__live=navigation") {
		t.Fatalf("htmx metadata patch should strip __live marker from canonical/hreflang")
	}

	payload := make(map[string]json.RawMessage)
	if err := json.Unmarshal([]byte(patchHeader), &payload); err != nil {
		t.Fatalf("htmx metadata patch header should be valid json payload: %v", err)
	}
	patchPayloadRaw, ok := payload["metagen:patch"]
	if !ok {
		t.Fatalf("htmx metadata patch header should include metagen:patch payload")
	}
	var patchPayload struct {
		Head string `json:"head"`
	}
	if err := json.Unmarshal(patchPayloadRaw, &patchPayload); err != nil {
		t.Fatalf("htmx metadata patch payload should be valid json: %v", err)
	}
	if strings.Contains(patchPayload.Head, `application/ld+json`) {
		t.Fatalf("htmx metadata patch head should not include structured data scripts")
	}
}

func TestHandlerLovelyEyeAnalyticsRendersWhenConfigured(t *testing.T) {
	testSrv := newTestServerWithLovelyEye(t, testLovelyEyeTrackerURL, testLovelyEyeSiteID)
	mux := testSrv.handler

	recRoot := performRequest(mux, http.MethodGet, "/")
	if recRoot.Code != http.StatusOK {
		t.Fatalf("root status: expected %d, got %d", http.StatusOK, recRoot.Code)
	}

	rootBody := requireBody(t, recRoot.Body)
	if !strings.Contains(rootBody, `src="`+testLovelyEyeTrackerURL+`"`) {
		t.Fatalf("root page should include lovely eye tracker script")
	}
	if !strings.Contains(rootBody, `data-site-key="`+testLovelyEyeSiteID+`"`) {
		t.Fatalf("root page should include lovely eye site key")
	}
	if !strings.Contains(rootBody, `href="https://github.com/RevoTale/lovely-eye"`) {
		t.Fatalf("root page should include lovely eye footer note")
	}

	recHTMX := performRequestWithHeaders(mux, http.MethodGet, "/?__live=navigation", map[string]string{
		"HX-Request": "true",
	})
	if recHTMX.Code != http.StatusOK {
		t.Fatalf("htmx status: expected %d, got %d", http.StatusOK, recHTMX.Code)
	}

	patchHeader := strings.TrimSpace(recHTMX.Header().Get("HX-Trigger-After-Settle"))
	if patchHeader == "" {
		t.Fatalf("htmx response should include metadata patch header")
	}
	if strings.Contains(patchHeader, testLovelyEyeTrackerURL) {
		t.Fatalf("htmx metadata patch should not include lovely eye tracker url")
	}
	if strings.Contains(patchHeader, testLovelyEyeSiteID) {
		t.Fatalf("htmx metadata patch should not include lovely eye site id")
	}
}

func TestHandlerLovelyEyeAnalyticsRequiresBothValues(t *testing.T) {
	cases := []struct {
		name      string
		scriptURL string
		siteID    string
	}{
		{name: "missing script url", siteID: testLovelyEyeSiteID},
		{name: "missing site id", scriptURL: testLovelyEyeTrackerURL},
		{name: "missing both"},
	}

	for _, tc := range cases {
		testSrv := newTestServerWithLovelyEye(t, tc.scriptURL, tc.siteID)
		recRoot := performRequest(testSrv.handler, http.MethodGet, "/")
		if recRoot.Code != http.StatusOK {
			t.Fatalf("%s status: expected %d, got %d", tc.name, http.StatusOK, recRoot.Code)
		}

		rootBody := requireBody(t, recRoot.Body)
		if strings.Contains(rootBody, `src="`+testLovelyEyeTrackerURL+`"`) {
			t.Fatalf("%s should not include lovely eye tracker script", tc.name)
		}
		if strings.Contains(rootBody, `href="https://github.com/RevoTale/lovely-eye"`) {
			t.Fatalf("%s should not include lovely eye footer note", tc.name)
		}
	}
}

func TestHandlerImageLoaderEnabledTransformsTemplateAndSEOImages(t *testing.T) {
	testSrv := newTestServerWithImageLoader(t, true)
	mux := testSrv.handler

	recRoot := performRequest(mux, http.MethodGet, "/")
	if recRoot.Code != http.StatusOK {
		t.Fatalf("root status: expected %d, got %d", http.StatusOK, recRoot.Code)
	}
	rootBody := requireBody(t, recRoot.Body)
	rawLogoURL := testSrv.bundle.URL("revtale-logo.svg")
	expectedLogoURL := imageloader.New(true).URL(rawLogoURL, 28)
	if !strings.Contains(rootBody, `class="server-logo" src="`+expectedLogoURL+`"`) {
		t.Fatalf("root page should rewrite server logo src when image loader is enabled")
	}

	recNote := performRequest(mux, http.MethodGet, "/uk/note/hello-world")
	if recNote.Code != http.StatusOK {
		t.Fatalf("note status: expected %d, got %d", http.StatusOK, recNote.Code)
	}
	noteBody := requireBody(t, recNote.Body)
	expectedSEOURL := "https://revotale.com/blog/notes/cdn/image/blog/1080/images/meta-hello.webp"
	if !strings.Contains(noteBody, `property="og:image" content="`+expectedSEOURL+`"`) {
		t.Fatalf("note page should include transformed og:image URL when loader is enabled")
	}
	if !strings.Contains(noteBody, `name="twitter:image" content="`+expectedSEOURL+`"`) {
		t.Fatalf("note page should include transformed twitter:image URL when loader is enabled")
	}

	noteDocs := parseJSONLDScripts(t, noteBody)
	noteDoc := requireJSONLDDocByType(t, noteDocs, "BlogPosting")
	noteImage := objectField(t, noteDoc, "image")
	if got := stringField(t, noteImage, "url"); got != expectedSEOURL {
		t.Fatalf("note JSON-LD image.url: expected %q, got %q", expectedSEOURL, got)
	}
}

func TestPagerLinksIncludeHTMXNavigationActions(t *testing.T) {
	t.Parallel()
	testSrv := newTestServer(t)
	mux := testSrv.handler

	recPrev := performRequest(mux, http.MethodGet, "/?page=2&author=l-you&tag=go&type=short")
	if recPrev.Code != http.StatusOK {
		t.Fatalf("pager prev page status: expected %d, got %d", http.StatusOK, recPrev.Code)
	}
	prevBody := requireBody(t, recPrev.Body)
	if !strings.Contains(prevBody, `hx-get="/?__live=navigation&amp;author=l-you&amp;tag=go&amp;type=short"`) {
		t.Fatalf("prev link should include htmx navigation url marker")
	}

	recNext := performRequest(mux, http.MethodGet, "/?author=l-you&tag=go&type=short")
	if recNext.Code != http.StatusOK {
		t.Fatalf("pager next page status: expected %d, got %d", http.StatusOK, recNext.Code)
	}
	nextBody := requireBody(t, recNext.Body)
	if !strings.Contains(nextBody, `hx-get="/?__live=navigation&amp;author=l-you&amp;page=2&amp;tag=go&amp;type=short"`) {
		t.Fatalf("next link should include htmx navigation url marker")
	}
	if !strings.Contains(nextBody, `hx-target="#notes-content"`) {
		t.Fatalf("pager links should target notes-content for partial swap")
	}
	if !strings.Contains(nextBody, `hx-select="#notes-content"`) {
		t.Fatalf("pager links should select notes-content fragment")
	}
	if !strings.Contains(nextBody, `hx-swap="outerHTML"`) {
		t.Fatalf("pager links should replace notes-content outer html")
	}

	recSearch := performRequest(mux, http.MethodGet, "/?q=hello&author=l-you&tag=go&type=short")
	if recSearch.Code != http.StatusOK {
		t.Fatalf("pager next search page status: expected %d, got %d", http.StatusOK, recSearch.Code)
	}
	searchBody := requireBody(t, recSearch.Body)
	if !strings.Contains(
		searchBody,
		`hx-get="/?__live=navigation&amp;author=l-you&amp;page=2&amp;q=hello&amp;tag=go&amp;type=short"`,
	) {
		t.Fatalf("next link should preserve q in htmx navigation url marker")
	}
	if !strings.Contains(searchBody, `class="topbar-search-clear" href="/?author=l-you&amp;tag=go&amp;type=short"`) {
		t.Fatalf("search clear action should preserve author/tag/type and drop q")
	}
	if !strings.Contains(nextBody, `hx-push-url="/?author=l-you&amp;page=2&amp;tag=go&amp;type=short"`) {
		t.Fatalf("pager links should push canonical url to history")
	}
	if !strings.Contains(nextBody, testSrv.bundle.URL("vendor/htmx.min.js")) {
		t.Fatalf("layout should include self-hosted htmx script")
	}
	if !strings.Contains(nextBody, testSrv.bundle.URL("app.js")) {
		t.Fatalf("layout should include self-hosted app script")
	}
}

func TestHandlerNotFoundAndHealth(t *testing.T) {
	t.Parallel()
	testSrv := newTestServer(t)
	mux := testSrv.handler

	recHealth := performRequest(mux, http.MethodGet, "/healthz")
	if recHealth.Code != http.StatusOK {
		t.Fatalf("healthz status: expected %d, got %d", http.StatusOK, recHealth.Code)
	}
	if body := strings.TrimSpace(requireBody(t, recHealth.Body)); body != "ok" {
		t.Fatalf("healthz body: expected %q, got %q", "ok", body)
	}

	recStatic := performRequest(mux, http.MethodGet, "/.revotale/tui.css")
	if recStatic.Code != http.StatusNotFound {
		t.Fatalf("unhashed static status: expected %d, got %d", http.StatusNotFound, recStatic.Code)
	}

	recHashedStatic := performRequest(mux, http.MethodGet, testSrv.bundle.URL("tui.css"))
	if recHashedStatic.Code != http.StatusOK {
		t.Fatalf("hashed static status: expected %d, got %d", http.StatusOK, recHashedStatic.Code)
	}
	if !strings.Contains(recHashedStatic.Header().Get("Content-Type"), "text/css") {
		t.Fatalf("hashed static content-type: expected css, got %q", recHashedStatic.Header().Get("Content-Type"))
	}
	staticBody := requireBody(t, recHashedStatic.Body)
	if !strings.Contains(staticBody, `:placeholder-shown)+.topbar-search-submit`) {
		t.Fatalf("static css should include active selector for search submit button")
	}

	recScript := performRequest(mux, http.MethodGet, testSrv.bundle.URL("app.js"))
	if recScript.Code != http.StatusOK {
		t.Fatalf("static script status: expected %d, got %d", http.StatusOK, recScript.Code)
	}
	if !strings.Contains(recScript.Header().Get("Content-Type"), "javascript") {
		t.Fatalf("static script content-type: expected javascript, got %q", recScript.Header().Get("Content-Type"))
	}
	scriptBody := requireBody(t, recScript.Body)
	if !strings.Contains(scriptBody, `scrollTo`) || !strings.Contains(scriptBody, `behavior:"smooth"`) {
		t.Fatalf("static script should include smooth scroll to top behavior")
	}
	if !strings.Contains(scriptBody, `.code-copy-button`) || !strings.Contains(scriptBody, `clipboard`) {
		t.Fatalf("static script should include copy button behavior")
	}

	recMissingNote := performRequest(mux, http.MethodGet, "/note/missing")
	if recMissingNote.Code != http.StatusNotFound {
		t.Fatalf("missing note status: expected %d, got %d", http.StatusNotFound, recMissingNote.Code)
	}
	missingNoteBody := requireBody(t, recMissingNote.Body)
	if !strings.Contains(missingNoteBody, "404 Not Found</title>") {
		t.Fatalf("missing note page should render custom 404 title")
	}
	if !strings.Contains(missingNoteBody, "/note/missing") {
		t.Fatalf("missing note page should include requested path")
	}

	recMissingAuthor := performRequest(mux, http.MethodGet, "/author/missing")
	if recMissingAuthor.Code != http.StatusNotFound {
		t.Fatalf("missing author status: expected %d, got %d", http.StatusNotFound, recMissingAuthor.Code)
	}
	missingAuthorBody := requireBody(t, recMissingAuthor.Body)
	if !strings.Contains(missingAuthorBody, "Signal lost") {
		t.Fatalf("missing author page should render custom 404 body")
	}

	recMissingTag := performRequest(mux, http.MethodGet, "/tag/missing")
	if recMissingTag.Code != http.StatusNotFound {
		t.Fatalf("missing tag status: expected %d, got %d", http.StatusNotFound, recMissingTag.Code)
	}
	missingTagBody := requireBody(t, recMissingTag.Body)
	if !strings.Contains(missingTagBody, "/tag/missing") {
		t.Fatalf("missing tag page should include requested path")
	}

	recNoLive := performRequest(mux, http.MethodGet, "/.live/note/hello-world")
	if recNoLive.Code != http.StatusNotFound {
		t.Fatalf("note live status: expected %d, got %d", http.StatusNotFound, recNoLive.Code)
	}
	noLiveBody := requireBody(t, recNoLive.Body)
	if !strings.Contains(noLiveBody, "/.live/note/hello-world") {
		t.Fatalf("note live fallback should render requested path")
	}

	recLegacyLive := performRequest(mux, http.MethodGet, "/live")
	if recLegacyLive.Code != http.StatusNotFound {
		t.Fatalf("legacy live status: expected %d, got %d", http.StatusNotFound, recLegacyLive.Code)
	}
	legacyLiveBody := requireBody(t, recLegacyLive.Body)
	if !strings.Contains(legacyLiveBody, "/live") {
		t.Fatalf("legacy live fallback should render requested path")
	}

	recMissingRoute := performRequest(mux, http.MethodGet, "/missing-route")
	if recMissingRoute.Code != http.StatusNotFound {
		t.Fatalf("missing route status: expected %d, got %d", http.StatusNotFound, recMissingRoute.Code)
	}
	missingRouteBody := requireBody(t, recMissingRoute.Body)
	if !strings.Contains(missingRouteBody, "/missing-route") {
		t.Fatalf("missing route page should include requested path")
	}
}

func TestGeneratedBootstrapSupportsAppOwnedEndpoints(t *testing.T) {
	t.Parallel()
	testSrv := newTestServer(t)
	mux := testSrv.handler

	recFeed := performRequest(mux, http.MethodGet, "/feed.xml?locale=en")
	if recFeed.Code != http.StatusOK {
		t.Fatalf("feed status: expected %d, got %d", http.StatusOK, recFeed.Code)
	}
	if got := recFeed.Header().Get("Content-Type"); !strings.Contains(got, "application/rss+xml") {
		t.Fatalf("feed content-type: expected rss xml, got %q", got)
	}
	feedBody := requireBody(t, recFeed.Body)
	if !strings.Contains(feedBody, "<rss") || !strings.Contains(feedBody, "Hello World") {
		t.Fatalf("feed endpoint should render rss payload")
	}

	recSitemap := performRequest(mux, http.MethodGet, "/sitemap-index")
	if recSitemap.Code != http.StatusOK {
		t.Fatalf("sitemap-index status: expected %d, got %d", http.StatusOK, recSitemap.Code)
	}
	if got := recSitemap.Header().Get("Content-Type"); !strings.Contains(got, "application/xml") {
		t.Fatalf("sitemap-index content-type: expected xml, got %q", got)
	}
	sitemapBody := requireBody(t, recSitemap.Body)
	if !strings.Contains(sitemapBody, "<sitemapindex") || !strings.Contains(sitemapBody, "/sitemap.xml") {
		t.Fatalf("sitemap-index endpoint should render sitemap index payload")
	}

	recRobots := performRequest(mux, http.MethodGet, "/robots.txt")
	if recRobots.Code != http.StatusOK {
		t.Fatalf("robots status: expected %d, got %d", http.StatusOK, recRobots.Code)
	}
	if got := recRobots.Header().Get("Content-Type"); !strings.Contains(got, "text/plain") {
		t.Fatalf("robots content-type: expected text/plain, got %q", got)
	}
	robotsBody := requireBody(t, recRobots.Body)
	if !strings.Contains(robotsBody, "User-agent: *") {
		t.Fatalf("robots endpoint should include user-agent rule")
	}
	if !strings.Contains(robotsBody, "Sitemap: https://revotale.com/blog/notes/sitemap-index") {
		t.Fatalf("robots endpoint should include sitemap index reference")
	}
}

func TestGeneratedMountRoutesAllowsManualRoutes(t *testing.T) {
	t.Parallel()

	bootstrap := newTestServerBootstrap(t, testServerOptions{})
	mux := http.NewServeMux()
	if err := webgen.MountRoutes(mux, bootstrap.cfg); err != nil {
		t.Fatalf("mount generated routes: %v", err)
	}
	mux.HandleFunc("/manual", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("manual"))
	})

	recManual := performRequest(mux, http.MethodGet, "/manual")
	if recManual.Code != http.StatusCreated {
		t.Fatalf("manual route status: expected %d, got %d", http.StatusCreated, recManual.Code)
	}
	if body := requireBody(t, recManual.Body); body != "manual" {
		t.Fatalf("manual route body: expected %q, got %q", "manual", body)
	}

	recGenerated := performRequest(mux, http.MethodGet, "/")
	if recGenerated.Code != http.StatusOK {
		t.Fatalf("generated route status: expected %d, got %d", http.StatusOK, recGenerated.Code)
	}
}
