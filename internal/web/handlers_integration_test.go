package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"blog/framework/httpserver"
	frameworki18n "blog/framework/i18n"
	"blog/framework/staticassets"
	"blog/internal/notes"
	"blog/internal/web/appcore"
	webgen "blog/internal/web/gen"
	webi18n "blog/internal/web/i18n"
	"github.com/Khan/genqlient/graphql"
)

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
						"meta": {"description": "hello note"}
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
						"meta": {"description": "hello note"}
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
						"meta": {"description": "hello note"}
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
						"externalLinks": [],
						"linkedMicroPosts": [],
						"meta": {"title":"Hello World","description":"hello note"}
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

func newTestServer(t *testing.T) testServer {
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

	appcore.SetStaticAssetBasePath(bundle.URLPrefix())
	i18nConfig, err := frameworki18n.NormalizeConfig(webi18n.Config())
	if err != nil {
		t.Fatalf("normalize i18n config: %v", err)
	}
	i18nCatalog, err := webi18n.LoadCatalog()
	if err != nil {
		t.Fatalf("load i18n catalog: %v", err)
	}
	i18nResolver, err := frameworki18n.NewResolver(i18nConfig)
	if err != nil {
		t.Fatalf("new i18n resolver: %v", err)
	}
	appcore.SetLocalizationConfig(i18nConfig)

	svc := notes.NewService(fakeGraphQLClient{}, 12, "")
	handler, err := httpserver.New(httpserver.Config[*appcore.Context]{
		AppContext:      appcore.NewContext(svc, i18nConfig, i18nCatalog),
		Handlers:        webgen.Handlers(webgen.NewRouteResolvers()),
		IsNotFoundError: appcore.IsNotFoundError,
		NotFoundPage:    webgen.NotFoundPage,
		Static: httpserver.StaticMount{
			URLPrefix: bundle.URLPrefix(),
			Dir:       bundle.Dir(),
		},
		CachePolicies: httpserver.DefaultCachePolicies(),
		LogServerError: func(error) {
		},
	})
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}
	handler = frameworki18n.Middleware(frameworki18n.MiddlewareConfig{
		Resolver: i18nResolver,
		BypassPrefixes: []string{
			bundle.URLPrefix(),
		},
		BypassExact: []string{
			"/healthz",
		},
	})(handler)

	return testServer{
		handler: handler,
		bundle:  bundle,
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

func TestHandlerPageRoutesRenderHTML(t *testing.T) {
	t.Parallel()
	testSrv := newTestServer(t)
	mux := testSrv.handler

	cases := []struct {
		path        string
		mustContain string
	}{
		{path: "/channels", mustContain: "<title>Channels :: blog</title>"},
		{path: "/", mustContain: "<title>Notes :: blog</title>"},
		{path: "/?q=hello", mustContain: "<title>Notes :: blog</title>"},
		{path: "/?author=l-you&tag=go&type=short", mustContain: "<h1>L You</h1>"},
		{path: "/note/hello-world", mustContain: "<title>Hello World :: blog</title>"},
		{path: "/author/l-you", mustContain: "<title>L You :: blog</title>"},
		{path: "/author/l-you?author=zed", mustContain: "<title>L You :: blog</title>"},
		{path: "/tag/go", mustContain: "<title>#Go :: blog</title>"},
		{path: "/tag/go?tag=rust", mustContain: "<title>#Go :: blog</title>"},
		{path: "/tales", mustContain: "<title>Tales :: blog</title>"},
		{path: "/tales?type=short", mustContain: "<title>Tales :: blog</title>"},
		{path: "/micro-tales", mustContain: "<title>Micro-tales :: blog</title>"},
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
	if !strings.Contains(ukBody, `<html lang="uk">`) {
		t.Fatalf("/uk should render localized html lang")
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
	if !strings.Contains(missingNoteBody, "<title>404 Not Found :: blog</title>") {
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
