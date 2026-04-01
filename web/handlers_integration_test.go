package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"regexp"
	goruntime "runtime"
	"strings"
	"testing"
	"time"

	"blog/internal/config"
	"blog/internal/imageloader"
	"blog/internal/notes"
	"blog/internal/site"
	generated "blog/web/generated"
	"blog/web/view"
	"github.com/Khan/genqlient/graphql"
	"github.com/RevoTale/no-js/framework/httpserver"
	frameworksite "github.com/RevoTale/no-js/framework/site"
	frameworkstaticassets "github.com/RevoTale/no-js/framework/staticassets"
	"github.com/stretchr/testify/require"
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
	bundle  testStaticBundle
}

type testStaticBundle struct {
	hash      string
	urlPrefix string
}

func (bundle testStaticBundle) URL(assetPath string) string {
	trimmedPath := strings.TrimPrefix(strings.TrimSpace(assetPath), "/")
	return frameworkstaticassets.Manifest{Hash: bundle.hash}.VersionedURLPrefix(bundle.urlPrefix) + trimmedPath
}

type testServerOptions struct {
	enableImageLoader  bool
	lovelyEyeScriptURL string
	lovelyEyeSiteID    string
	mountExtraRoutes   func(*http.ServeMux) error
	siteResolver       frameworksite.Resolver
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

	handler, bundle := newTestHandler(t, options)

	return testServer{
		handler: handler,
		bundle:  bundle,
	}
}

func newTestHandler(t *testing.T, options testServerOptions) (http.Handler, testStaticBundle) {
	t.Helper()

	const staticURLPrefix = "/_assets/"
	_, currentFile, _, ok := goruntime.Caller(0)
	require.True(t, ok)
	t.Chdir(filepath.Dir(filepath.Dir(currentFile)))

	manifestPath := "web/assets-build/manifest.json"
	manifest, err := frameworkstaticassets.ReadManifest(manifestPath)
	require.NoError(t, err)

	siteResolver := options.siteResolver
	if siteResolver == nil {
		var err error
		siteResolver, err = site.NewResolver(config.Config{RootURL: testRootURL})
		require.NoError(t, err)
	}
	imageLoader := imageloader.New(options.enableImageLoader)
	noteService := notes.NewService(fakeGraphQLClient{}, 12, imageLoader)
	appContext, err := runtime.NewContext(runtime.Config{
		Notes:              noteService,
		SiteResolver:       siteResolver,
		ImageLoader:        imageLoader,
		LovelyEyeScriptURL: options.lovelyEyeScriptURL,
		LovelyEyeSiteID:    options.lovelyEyeSiteID,
	})
	require.NoError(t, err)

	cachePolicies := httpserver.DefaultCachePolicies()
	cachePolicies.Static = "public, max-age=31536000, immutable"

	handler, err := httpserver.NewApp(httpserver.Config[*runtime.Context]{
		App: generated.Bundle(appContext),
		Custom: httpserver.CustomConfig{
			ExtraRoutes: options.mountExtraRoutes,
			MainMiddlewares: []func(http.Handler) http.Handler{
				runtime.WithCanonicalNotesRedirects,
			},
			CachePolicies:  cachePolicies,
			LogServerError: func(error) {},
		},
	})
	require.NoError(t, err)

	return handler, testStaticBundle{
		hash:      manifest.Hash,
		urlPrefix: staticURLPrefix,
	}
}

func requireBody(t *testing.T, body io.Reader) string {
	t.Helper()

	content, err := io.ReadAll(body)
	require.NoError(t, err)
	return string(content)
}

func performRequest(mux http.Handler, method string, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

type requestHostSiteResolver struct {
	canonicalURL string
}

func (resolver requestHostSiteResolver) CanonicalURL() string {
	return strings.TrimSpace(resolver.canonicalURL)
}

func (resolver requestHostSiteResolver) Resolve(r *http.Request) string {
	parsed, err := url.Parse(strings.TrimSpace(resolver.canonicalURL))
	if err != nil {
		return strings.TrimSpace(resolver.canonicalURL)
	}

	if r == nil {
		return parsed.String()
	}

	if scheme := strings.TrimSpace(r.URL.Scheme); scheme != "" {
		parsed.Scheme = scheme
	}
	if host := strings.TrimSpace(r.Host); host != "" {
		parsed.Host = host
	}

	return parsed.String()
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
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		var doc map[string]any
		require.NoError(t, json.Unmarshal([]byte(match[1]), &doc))
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
	require.FailNow(t, "expected JSON-LD document with @type=%q", typeName)
	return nil
}

func stringField(t *testing.T, object map[string]any, key string) string {
	t.Helper()

	value, ok := object[key]
	require.True(t, ok)
	text, ok := value.(string)
	require.True(t, ok)
	return text
}

func objectField(t *testing.T, object map[string]any, key string) map[string]any {
	t.Helper()

	value, ok := object[key]
	require.True(t, ok)
	out, ok := value.(map[string]any)
	require.True(t, ok)
	return out
}

func arrayField(t *testing.T, object map[string]any, key string) []any {
	t.Helper()

	value, ok := object[key]
	require.True(t, ok)
	out, ok := value.([]any)
	require.True(t, ok)
	return out
}

func objectFromAny(t *testing.T, value any, field string) map[string]any {
	t.Helper()

	out, ok := value.(map[string]any)
	require.True(t, ok)
	return out
}

func TestHandlerPageRoutesRenderHTML(t *testing.T) {
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

		require.Equal(t, http.StatusOK, rec.Code)
		require.Contains(t, rec.Header().Get("Content-Type"), "text/html")

		body := requireBody(t, rec.Body)
		require.Contains(t, body, tc.mustContain)
		require.NotContains(t, body, "event: datastar-patch-elements")
	}
}

func TestCanonicalListingQueryRedirects(t *testing.T) {
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
		require.Equal(t, http.StatusPermanentRedirect, rec.Code)
		require.Equal(t, tc.location, rec.Header().Get("Location"))
	}
}

func TestRobotsRulesWithAndWithoutQuery(t *testing.T) {
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
		require.Equal(t, http.StatusOK, rec.Code)
		body := requireBody(t, rec.Body)
		expectedTag := `name="robots" content="` + tc.expectedRobots + `"`
		require.Contains(t, body, expectedTag)
	}
}

func TestUnknownListingQueryParamsStayNoIndexWithoutCanonicalRedirect(t *testing.T) {
	testSrv := newTestServer(t)
	mux := testSrv.handler

	cases := []string{
		"/?utm_source=test",
		"/?author=l-you&utm_source=test",
		"/tag/go?utm_source=test",
	}

	for _, path := range cases {
		rec := performRequest(mux, http.MethodGet, path)
		require.Equal(t, http.StatusOK, rec.Code)

		body := requireBody(t, rec.Body)
		require.Contains(t, body, `name="robots" content="noindex, follow"`)
	}
}

func TestSidebarLinkBehavior(t *testing.T) {
	testSrv := newTestServer(t)
	mux := testSrv.handler

	root := performRequest(mux, http.MethodGet, "/")
	rootBody := requireBody(t, root.Body)
	require.Contains(t, rootBody, `href="/channels"`)
	require.Contains(t, rootBody, `href="/author/l-you"`)
	require.Contains(t, rootBody, `href="/tag/go"`)
	require.Contains(t, rootBody, `class="topbar-rss-link" href="/feed.xml?locale=en"`)
	require.Contains(t, rootBody, `href="/tales"`)
	require.Contains(t, rootBody, `href="/micro-tales"`)
	require.NotContains(t, rootBody, `href="/?author=`)
	require.NotContains(t, rootBody, `href="/?tag=`)
	require.NotContains(t, rootBody, `topbar-search-clear`)

	search := performRequest(mux, http.MethodGet, "/?q=hello")
	searchBody := requireBody(t, search.Body)
	require.Contains(t, searchBody, `<form class="topbar-search" role="search" method="get" action="/">`)
	require.Contains(t, searchBody, `name="q"`)
	require.Contains(t, searchBody, `value="hello"`)
	require.Contains(t, searchBody, `class="topbar-rss-link" href="/feed.xml?locale=en&amp;q=hello"`)
	require.Contains(t, searchBody, `href="/channels?q=hello"`)
	require.Contains(t, searchBody, `href="/?author=l-you&amp;q=hello"`)
	require.Contains(t, searchBody, `href="/?q=hello&amp;tag=go"`)
	require.Contains(t, searchBody, `class="topbar-search-clear"`)
	require.Contains(t, searchBody, `class="topbar-search-clear" href="/"`)
	require.Contains(
		t,
		searchBody,
		`rel="alternate" type="application/rss+xml" href="https://revotale.com/blog/notes/feed.xml?locale=en&amp;q=hello"`,
	)

	filtered := performRequest(mux, http.MethodGet, "/author/l-you?tag=go&type=short")
	filteredBody := requireBody(t, filtered.Body)
	require.Contains(t, filteredBody, `href="/channels?author=l-you&amp;tag=go&amp;type=short"`)
	require.Contains(t, filteredBody, `href="/"`)
	require.Contains(t, filteredBody, `href="/?tag=go&amp;type=short"`)
	require.Contains(t, filteredBody, `href="/?author=l-you&amp;type=short"`)
	require.Contains(t, filteredBody, `href="/?author=l-you&amp;tag=go"`)
	require.Contains(
		t,
		filteredBody,
		`class="topbar-rss-link" href="/feed.xml?author=l-you&amp;locale=en&amp;tag=go&amp;type=short"`,
	)
	require.Contains(t, filteredBody, `href="/?author=zed&amp;tag=go&amp;type=short"`)
	require.Contains(t, filteredBody, `href="/?author=l-you&amp;tag=rust&amp;type=short"`)
	require.Contains(t, filteredBody, `href="/?author=l-you&amp;tag=go&amp;type=long"`)
	require.Contains(t, filteredBody, `href="/?tag=go&amp;type=short"`)
	require.Contains(t, filteredBody, `href="/?author=l-you&amp;type=short"`)

	channelsFiltered := performRequest(mux, http.MethodGet, "/channels?author=l-you&tag=go&type=short")
	channelsFilteredBody := requireBody(t, channelsFiltered.Body)
	require.Contains(t, channelsFilteredBody, `href="/?tag=go&amp;type=short"`)
	require.Contains(t, channelsFilteredBody, `href="/?author=zed&amp;tag=go&amp;type=short"`)
	require.Contains(t, channelsFilteredBody, `channels-desktop-hint`)
	require.Contains(t, channelsFilteredBody, `channels-mobile-panel`)

	channelsSingle := performRequest(mux, http.MethodGet, "/channels?author=l-you")
	channelsSingleBody := requireBody(t, channelsSingle.Body)
	require.Contains(t, channelsSingleBody, `class="back-link channels-back-button" href="/author/l-you"`)
}

func TestI18nRoutingAndLocalizedURLs(t *testing.T) {
	testSrv := newTestServer(t)
	mux := testSrv.handler

	recUK := performRequest(mux, http.MethodGet, "/uk")
	require.Equal(t, http.StatusOK, recUK.Code)
	ukBody := requireBody(t, recUK.Body)
	require.Contains(t, strings.ToLower(ukBody), "<!doctype html>")
	require.Contains(t, ukBody, `<html lang="uk">`)
	headIndex := strings.Index(ukBody, "<head>")
	mainIndex := strings.Index(ukBody, "<main")
	require.True(t, headIndex >= 0 && mainIndex >= 0 && headIndex <= mainIndex)
	require.Contains(t, ukBody, `href="/uk/channels"`)
	require.Contains(t, ukBody, `href="/uk/author/l-you"`)
	require.Contains(t, ukBody, `href="/uk/tag/go"`)
	require.Contains(t, ukBody, `href="/uk/note/hello-world"`)

	recUKNote := performRequest(mux, http.MethodGet, "/uk/note/hello-world")
	require.Equal(t, http.StatusOK, recUKNote.Code)
	ukNoteBody := requireBody(t, recUKNote.Body)
	require.Contains(t, ukNoteBody, `href="/uk"`)

	recDefaultPrefixed := performRequest(mux, http.MethodGet, "/en/note/hello-world")
	require.Equal(t, http.StatusPermanentRedirect, recDefaultPrefixed.Code)
	require.Equal(t, "/note/hello-world", recDefaultPrefixed.Header().Get("Location"))

	recUnknownLocale := performRequest(mux, http.MethodGet, "/it/note/hello-world")
	require.Equal(t, http.StatusNotFound, recUnknownLocale.Code)
}

func TestHandlerHTMXRoutesReturnPartial(t *testing.T) {
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
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, httpserver.DefaultCachePolicies().Live, rec.Header().Get("Cache-Control"))

		body := requireBody(t, rec.Body)
		require.Contains(t, body, tc.mustContain)
		require.NotContains(t, body, "<title>")
		require.NotContains(t, body, `application/ld+json`)
	}
}

func TestHandlerSEOMetadataAndHTMXPatchHeaders(t *testing.T) {
	testSrv := newTestServer(t)
	mux := testSrv.handler

	recNote := performRequest(mux, http.MethodGet, "/uk/note/hello-world")
	require.Equal(t, http.StatusOK, recNote.Code)
	noteBody := requireBody(t, recNote.Body)
	require.Contains(t, noteBody, `rel="canonical" href="https://revotale.com/blog/notes/uk/note/hello-world"`)
	require.NotContains(t, noteBody, "__live=navigation")
	require.Contains(t, noteBody, `rel="alternate" hreflang="en"`)
	require.Contains(t, noteBody, `property="og:title"`)
	require.Contains(t, noteBody, `property="og:url" content="https://revotale.com/blog/notes/uk/note/hello-world"`)
	require.Contains(t, noteBody, `name="twitter:card"`)
	require.Contains(t, noteBody, `property="article:published_time"`)
	require.Contains(t, noteBody, `property="article:author" content="https://revotale.com/blog/notes/uk/author/l-you"`)
	require.Contains(t, noteBody, `property="article:tag" content="Go"`)
	require.Contains(t, noteBody, `class="topbar-rss-link" href="/feed.xml?locale=uk"`)
	noteDocs := parseJSONLDScripts(t, noteBody)
	noteDoc := requireJSONLDDocByType(t, noteDocs, "BlogPosting")
	require.Equal(t, "https://revotale.com/blog/notes/uk/note/hello-world", stringField(t, noteDoc, "url"))
	mainEntity := objectField(t, noteDoc, "mainEntityOfPage")
	require.Equal(t, "https://revotale.com/blog/notes/uk/note/hello-world", stringField(t, mainEntity, "@id"))
	publisher := objectField(t, noteDoc, "publisher")
	require.Equal(t, "https://revotale.com/blog/notes", stringField(t, publisher, "url"))
	authors := arrayField(t, noteDoc, "author")
	require.NotEmpty(t, authors)
	firstAuthor := objectFromAny(t, authors[0], "author[0]")
	require.Equal(t, "https://revotale.com/blog/notes/uk/author/l-you", stringField(t, firstAuthor, "url"))
	datePublished := stringField(t, noteDoc, "datePublished")
	_, err := time.Parse(time.RFC3339, datePublished)
	require.NoError(t, err)
	mentions := arrayField(t, noteDoc, "mentions")
	require.GreaterOrEqual(t, len(mentions), 2)
	mentionURLs := make(map[string]struct{}, len(mentions))
	for idx, mention := range mentions {
		obj := objectFromAny(t, mention, fmt.Sprintf("mentions[%d]", idx))
		mentionURLs[stringField(t, obj, "@id")] = struct{}{}
	}
	_, ok := mentionURLs["https://example.com/docs"]
	require.True(t, ok)
	_, ok = mentionURLs["https://revotale.com/blog/notes/uk/note/hello-linked"]
	require.True(t, ok)

	recRoot := performRequest(mux, http.MethodGet, "/")
	require.Equal(t, http.StatusOK, recRoot.Code)

	recFavicon := performRequest(mux, http.MethodGet, "/favicon.svg")
	require.Equal(t, http.StatusOK, recFavicon.Code)
	require.Contains(t, recFavicon.Header().Get("Content-Type"), "image/svg+xml")

	recManifest := performRequest(mux, http.MethodGet, "/site.webmanifest")
	require.Equal(t, http.StatusOK, recManifest.Code)
	require.Contains(t, recManifest.Header().Get("Content-Type"), "application/manifest+json")

	rootBody := requireBody(t, recRoot.Body)
	expectedLogoURL := testSrv.bundle.URL("revtale-logo.svg")
	require.Contains(t, rootBody, `class="server-logo" src="`+expectedLogoURL+`"`)
	transformedLogoURL := imageloader.New(true).URL(expectedLogoURL, 28)
	require.NotContains(t, rootBody, transformedLogoURL)
	require.Contains(t, rootBody, `rel="manifest" href="/site.webmanifest"`)
	require.Contains(t, rootBody, `rel="icon" href="/favicon.ico"`)
	require.Contains(t, rootBody, `rel="apple-touch-icon"`)
	require.Contains(t, rootBody, `rel="alternate" type="application/rss+xml"`)
	rootDocs := parseJSONLDScripts(t, rootBody)
	rootBlog := requireJSONLDDocByType(t, rootDocs, "Blog")
	require.Equal(t, "https://revotale.com/blog/notes", stringField(t, rootBlog, "url"))
	blogPosts := arrayField(t, rootBlog, "blogPost")
	require.NotEmpty(t, blogPosts)
	firstPost := objectFromAny(t, blogPosts[0], "blogPost[0]")
	require.Equal(t, "BlogPosting", stringField(t, firstPost, "@type"))
	require.Equal(t, "https://revotale.com/blog/notes/note/hello-world", stringField(t, firstPost, "url"))
	firstPostMainEntity := objectField(t, firstPost, "mainEntityOfPage")
	require.Equal(t, "https://revotale.com/blog/notes/note/hello-world", stringField(t, firstPostMainEntity, "@id"))
	firstPostAuthors := arrayField(t, firstPost, "author")
	require.NotEmpty(t, firstPostAuthors)
	firstPostAuthor := objectFromAny(t, firstPostAuthors[0], "blogPost[0].author[0]")
	require.Equal(t, "https://revotale.com/blog/notes/author/l-you", stringField(t, firstPostAuthor, "url"))
	firstPostMentions := arrayField(t, firstPost, "mentions")
	require.GreaterOrEqual(t, len(firstPostMentions), 2)

	recChannels := performRequest(mux, http.MethodGet, "/channels")
	require.Equal(t, http.StatusOK, recChannels.Code)
	channelsBody := requireBody(t, recChannels.Body)
	require.Contains(t, channelsBody, `name="robots" content="noindex, follow"`)
	channelsDocs := parseJSONLDScripts(t, channelsBody)
	require.Len(t, channelsDocs, 0)

	recHTMX := performRequestWithHeaders(mux, http.MethodGet, "/?__live=navigation", map[string]string{
		"HX-Request": "true",
	})
	require.Equal(t, http.StatusOK, recHTMX.Code)
	patchHeader := strings.TrimSpace(recHTMX.Header().Get("HX-Trigger-After-Settle"))
	require.NotEmpty(t, patchHeader)
	require.Contains(t, patchHeader, "metagen:patch")
	require.NotContains(t, patchHeader, "__live=navigation")

	payload := make(map[string]json.RawMessage)
	require.NoError(t, json.Unmarshal([]byte(patchHeader), &payload))
	patchPayloadRaw, ok := payload["metagen:patch"]
	require.True(t, ok)
	var patchPayload struct {
		Head string `json:"head"`
	}
	require.NoError(t, json.Unmarshal(patchPayloadRaw, &patchPayload))
	require.NotContains(t, patchPayload.Head, `application/ld+json`)
}

func TestDynamicRootURLUsesRequestHostAcrossMetadataAndDiscovery(t *testing.T) {
	testSrv := newTestServerWithOptions(t, testServerOptions{
		siteResolver: requestHostSiteResolver{canonicalURL: testRootURL},
	})
	mux := testSrv.handler

	recNote := performRequest(mux, http.MethodGet, "https://mirror.example/uk/note/hello-world")
	require.Equal(t, http.StatusOK, recNote.Code)
	noteBody := requireBody(t, recNote.Body)
	require.Contains(t, noteBody, `rel="canonical" href="https://mirror.example/blog/notes/uk/note/hello-world"`)
	require.Contains(t, noteBody, `property="og:url" content="https://mirror.example/blog/notes/uk/note/hello-world"`)
	require.Contains(t, noteBody, `property="article:author" content="https://mirror.example/blog/notes/uk/author/l-you"`)

	noteDocs := parseJSONLDScripts(t, noteBody)
	noteDoc := requireJSONLDDocByType(t, noteDocs, "BlogPosting")
	require.Equal(t, "https://mirror.example/blog/notes/uk/note/hello-world", stringField(t, noteDoc, "url"))
	publisher := objectField(t, noteDoc, "publisher")
	require.Equal(t, "https://mirror.example/blog/notes", stringField(t, publisher, "url"))

	recFeed := performRequest(mux, http.MethodGet, "https://mirror.example/feed.xml?locale=uk")
	require.Equal(t, http.StatusOK, recFeed.Code)
	feedBody := requireBody(t, recFeed.Body)
	require.Contains(t, feedBody, "https://mirror.example/blog/notes/uk/note/hello-world")

	recRobots := performRequest(mux, http.MethodGet, "https://mirror.example/robots.txt")
	require.Equal(t, http.StatusOK, recRobots.Code)
	robotsBody := requireBody(t, recRobots.Body)
	require.Contains(t, robotsBody, "Sitemap: https://mirror.example/blog/notes/sitemap-index")
}

func TestHandlerLovelyEyeAnalyticsRendersWhenConfigured(t *testing.T) {
	testSrv := newTestServerWithLovelyEye(t, testLovelyEyeTrackerURL, testLovelyEyeSiteID)
	mux := testSrv.handler

	recRoot := performRequest(mux, http.MethodGet, "/")
	require.Equal(t, http.StatusOK, recRoot.Code)

	rootBody := requireBody(t, recRoot.Body)
	require.Contains(t, rootBody, `src="`+testLovelyEyeTrackerURL+`"`)
	require.Contains(t, rootBody, `data-site-key="`+testLovelyEyeSiteID+`"`)
	require.Contains(t, rootBody, `href="https://github.com/RevoTale/lovely-eye"`)

	recHTMX := performRequestWithHeaders(mux, http.MethodGet, "/?__live=navigation", map[string]string{
		"HX-Request": "true",
	})
	require.Equal(t, http.StatusOK, recHTMX.Code)

	patchHeader := strings.TrimSpace(recHTMX.Header().Get("HX-Trigger-After-Settle"))
	require.NotEmpty(t, patchHeader)
	require.NotContains(t, patchHeader, testLovelyEyeTrackerURL)
	require.NotContains(t, patchHeader, testLovelyEyeSiteID)
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
		require.Equal(t, http.StatusOK, recRoot.Code)

		rootBody := requireBody(t, recRoot.Body)
		require.NotContains(t, rootBody, `src="`+testLovelyEyeTrackerURL+`"`)
		require.NotContains(t, rootBody, `href="https://github.com/RevoTale/lovely-eye"`)
	}
}

func TestHandlerImageLoaderEnabledTransformsTemplateAndSEOImages(t *testing.T) {
	testSrv := newTestServerWithImageLoader(t, true)
	mux := testSrv.handler

	recRoot := performRequest(mux, http.MethodGet, "/")
	require.Equal(t, http.StatusOK, recRoot.Code)
	rootBody := requireBody(t, recRoot.Body)
	rawLogoURL := testSrv.bundle.URL("revtale-logo.svg")
	expectedLogoURL := imageloader.New(true).URL(rawLogoURL, 28)
	require.Contains(t, rootBody, `class="server-logo" src="`+expectedLogoURL+`"`)

	recNote := performRequest(mux, http.MethodGet, "/uk/note/hello-world")
	require.Equal(t, http.StatusOK, recNote.Code)
	noteBody := requireBody(t, recNote.Body)
	expectedSEOURL := "https://revotale.com/blog/notes/cdn/image/blog/1080/images/meta-hello.webp"
	require.Contains(t, noteBody, `property="og:image" content="`+expectedSEOURL+`"`)
	require.Contains(t, noteBody, `name="twitter:image" content="`+expectedSEOURL+`"`)

	noteDocs := parseJSONLDScripts(t, noteBody)
	noteDoc := requireJSONLDDocByType(t, noteDocs, "BlogPosting")
	noteImage := objectField(t, noteDoc, "image")
	require.Equal(t, expectedSEOURL, stringField(t, noteImage, "url"))
}

func TestPagerLinksIncludeHTMXNavigationActions(t *testing.T) {
	testSrv := newTestServer(t)
	mux := testSrv.handler

	recPrev := performRequest(mux, http.MethodGet, "/?page=2&author=l-you&tag=go&type=short")
	require.Equal(t, http.StatusOK, recPrev.Code)
	prevBody := requireBody(t, recPrev.Body)
	require.Contains(t, prevBody, `hx-get="/?__live=navigation&amp;author=l-you&amp;tag=go&amp;type=short"`)

	recNext := performRequest(mux, http.MethodGet, "/?author=l-you&tag=go&type=short")
	require.Equal(t, http.StatusOK, recNext.Code)
	nextBody := requireBody(t, recNext.Body)
	require.Contains(t, nextBody, `hx-get="/?__live=navigation&amp;author=l-you&amp;page=2&amp;tag=go&amp;type=short"`)
	require.Contains(t, nextBody, `hx-target="#notes-content"`)
	require.Contains(t, nextBody, `hx-select="#notes-content"`)
	require.Contains(t, nextBody, `hx-swap="outerHTML"`)

	recSearch := performRequest(mux, http.MethodGet, "/?q=hello&author=l-you&tag=go&type=short")
	require.Equal(t, http.StatusOK, recSearch.Code)
	searchBody := requireBody(t, recSearch.Body)
	require.Contains(
		t,
		searchBody,
		`hx-get="/?__live=navigation&amp;author=l-you&amp;page=2&amp;q=hello&amp;tag=go&amp;type=short"`,
	)
	require.Contains(t, searchBody, `class="topbar-search-clear" href="/?author=l-you&amp;tag=go&amp;type=short"`)
	require.Contains(t, nextBody, `hx-push-url="/?author=l-you&amp;page=2&amp;tag=go&amp;type=short"`)
	require.Contains(t, nextBody, testSrv.bundle.URL("vendor/htmx.min.js"))
	require.Contains(t, nextBody, testSrv.bundle.URL("app.js"))
}

func TestHandlerNotFoundAndHealth(t *testing.T) {
	testSrv := newTestServer(t)
	mux := testSrv.handler

	recHealth := performRequest(mux, http.MethodGet, "/healthz")
	require.Equal(t, http.StatusOK, recHealth.Code)
	require.Equal(t, "ok", strings.TrimSpace(requireBody(t, recHealth.Body)))

	recStatic := performRequest(mux, http.MethodGet, "/_assets/tui.css")
	require.Equal(t, http.StatusNotFound, recStatic.Code)

	recHashedStatic := performRequest(mux, http.MethodGet, testSrv.bundle.URL("tui.css"))
	require.Equal(t, http.StatusOK, recHashedStatic.Code)
	require.Contains(t, recHashedStatic.Header().Get("Content-Type"), "text/css")
	staticBody := requireBody(t, recHashedStatic.Body)
	require.Contains(t, staticBody, `:placeholder-shown)+.topbar-search-submit`)

	recScript := performRequest(mux, http.MethodGet, testSrv.bundle.URL("app.js"))
	require.Equal(t, http.StatusOK, recScript.Code)
	require.Contains(t, recScript.Header().Get("Content-Type"), "javascript")
	scriptBody := requireBody(t, recScript.Body)
	require.Contains(t, scriptBody, `scrollTo`)
	require.Contains(t, scriptBody, `behavior:"smooth"`)
	require.Contains(t, scriptBody, `.code-copy-button`)
	require.Contains(t, scriptBody, `clipboard`)

	recMissingNote := performRequest(mux, http.MethodGet, "/note/missing")
	require.Equal(t, http.StatusNotFound, recMissingNote.Code)
	missingNoteBody := requireBody(t, recMissingNote.Body)
	require.Contains(t, missingNoteBody, "404 Not Found</title>")
	require.Contains(t, missingNoteBody, "/note/missing")

	recMissingAuthor := performRequest(mux, http.MethodGet, "/author/missing")
	require.Equal(t, http.StatusNotFound, recMissingAuthor.Code)
	missingAuthorBody := requireBody(t, recMissingAuthor.Body)
	require.Contains(t, missingAuthorBody, "Signal lost")

	recMissingTag := performRequest(mux, http.MethodGet, "/tag/missing")
	require.Equal(t, http.StatusNotFound, recMissingTag.Code)
	missingTagBody := requireBody(t, recMissingTag.Body)
	require.Contains(t, missingTagBody, "/tag/missing")

	recNoLive := performRequest(mux, http.MethodGet, "/.live/note/hello-world")
	require.Equal(t, http.StatusNotFound, recNoLive.Code)
	noLiveBody := requireBody(t, recNoLive.Body)
	require.Contains(t, noLiveBody, "/.live/note/hello-world")

	recLegacyLive := performRequest(mux, http.MethodGet, "/live")
	require.Equal(t, http.StatusNotFound, recLegacyLive.Code)
	legacyLiveBody := requireBody(t, recLegacyLive.Body)
	require.Contains(t, legacyLiveBody, "/live")

	recMissingRoute := performRequest(mux, http.MethodGet, "/missing-route")
	require.Equal(t, http.StatusNotFound, recMissingRoute.Code)
	missingRouteBody := requireBody(t, recMissingRoute.Body)
	require.Contains(t, missingRouteBody, "/missing-route")
}

func TestHTTPServerSupportsAppOwnedEndpoints(t *testing.T) {
	testSrv := newTestServer(t)
	mux := testSrv.handler

	recFeed := performRequest(mux, http.MethodGet, "/feed.xml?locale=en")
	require.Equal(t, http.StatusOK, recFeed.Code)
	require.Contains(t, recFeed.Header().Get("Content-Type"), "application/rss+xml")
	feedBody := requireBody(t, recFeed.Body)
	require.Contains(t, feedBody, "<rss")
	require.Contains(t, feedBody, "Hello World")

	recSitemap := performRequest(mux, http.MethodGet, "/sitemap-index")
	require.Equal(t, http.StatusOK, recSitemap.Code)
	require.Contains(t, recSitemap.Header().Get("Content-Type"), "application/xml")
	sitemapBody := requireBody(t, recSitemap.Body)
	require.Contains(t, sitemapBody, "<sitemapindex")
	require.Contains(t, sitemapBody, "/sitemap.xml")

	recRobots := performRequest(mux, http.MethodGet, "/robots.txt")
	require.Equal(t, http.StatusOK, recRobots.Code)
	require.Contains(t, recRobots.Header().Get("Content-Type"), "text/plain")
	robotsBody := requireBody(t, recRobots.Body)
	require.Contains(t, robotsBody, "User-agent: *")
	require.Contains(t, robotsBody, "Sitemap: https://revotale.com/blog/notes/sitemap-index")
}

func TestHTTPServerExtraRoutesHookAllowsManualRoutes(t *testing.T) {
	testSrv := newTestServerWithOptions(t, testServerOptions{
		mountExtraRoutes: func(mux *http.ServeMux) error {
			mux.HandleFunc("/manual", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte("manual"))
			})

			return nil
		},
	})

	recManual := performRequest(testSrv.handler, http.MethodGet, "/manual")
	require.Equal(t, http.StatusCreated, recManual.Code)
	require.Equal(t, "manual", requireBody(t, recManual.Body))

	recGenerated := performRequest(testSrv.handler, http.MethodGet, "/")
	require.Equal(t, http.StatusOK, recGenerated.Code)
}
