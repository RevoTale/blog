package web

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"net/http/httptest"

	"blog/framework/httpserver"
	"blog/internal/notes"
	"blog/internal/web/appcore"
	webgen "blog/internal/web/gen"
	"github.com/Khan/genqlient/graphql"
)

type fakeGraphQLClient struct{}

func (fakeGraphQLClient) MakeRequest(
	_ context.Context,
	req *graphql.Request,
	resp *graphql.Response,
) error {
	slug := requestVarString(req, "slug")
	name := requestVarString(req, "name")

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

func newTestMux(t *testing.T) http.Handler {
	t.Helper()

	svc := notes.NewService(fakeGraphQLClient{}, 12, "")
	handler, err := httpserver.New(httpserver.Config[*appcore.Context]{
		AppContext:      appcore.NewContext(svc),
		Handlers:        webgen.Handlers(webgen.NewRouteResolvers()),
		IsNotFoundError: appcore.IsNotFoundError,
		NotFoundPage:    webgen.NotFoundPage,
		Static: httpserver.StaticMount{
			URLPrefix: "/.revotale/",
			Dir:       "../../internal/web/static",
		},
		CachePolicies: httpserver.DefaultCachePolicies(),
		LogServerError: func(error) {
		},
	})
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}
	return handler
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

func TestHandlerPageRoutesRenderHTML(t *testing.T) {
	t.Parallel()
	mux := newTestMux(t)

	cases := []struct {
		path        string
		mustContain string
	}{
		{path: "/channels", mustContain: "<title>Channels :: blog</title>"},
		{path: "/", mustContain: "<title>Notes :: blog</title>"},
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
	mux := newTestMux(t)

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

func TestHandlerLiveRoutesReturnPatch(t *testing.T) {
	t.Parallel()
	mux := newTestMux(t)

	cases := []struct {
		path     string
		selector string
	}{
		{path: "/live", selector: "#notes-content"},
		{path: "/author/l-you/live", selector: "#author-content"},
	}

	for _, tc := range cases {
		rec := performRequest(mux, http.MethodGet, tc.path)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status: expected %d, got %d", tc.path, http.StatusOK, rec.Code)
		}

		body := requireBody(t, rec.Body)
		if !strings.Contains(body, "event: datastar-patch-elements") {
			t.Fatalf("%s missing datastar patch event", tc.path)
		}
		if !strings.Contains(body, "data: selector "+tc.selector) {
			t.Fatalf("%s missing selector %q", tc.path, tc.selector)
		}
	}
}

func TestHandlerNotFoundAndHealth(t *testing.T) {
	t.Parallel()
	mux := newTestMux(t)

	recHealth := performRequest(mux, http.MethodGet, "/healthz")
	if recHealth.Code != http.StatusOK {
		t.Fatalf("healthz status: expected %d, got %d", http.StatusOK, recHealth.Code)
	}
	if body := strings.TrimSpace(requireBody(t, recHealth.Body)); body != "ok" {
		t.Fatalf("healthz body: expected %q, got %q", "ok", body)
	}

	recStatic := performRequest(mux, http.MethodGet, "/.revotale/tui.css")
	if recStatic.Code != http.StatusOK {
		t.Fatalf("static status: expected %d, got %d", http.StatusOK, recStatic.Code)
	}
	if !strings.Contains(recStatic.Header().Get("Content-Type"), "text/css") {
		t.Fatalf("static content-type: expected css, got %q", recStatic.Header().Get("Content-Type"))
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

	recNoLive := performRequest(mux, http.MethodGet, "/note/hello-world/live")
	if recNoLive.Code != http.StatusNotFound {
		t.Fatalf("note live status: expected %d, got %d", http.StatusNotFound, recNoLive.Code)
	}
	noLiveBody := requireBody(t, recNoLive.Body)
	if !strings.Contains(noLiveBody, "/note/hello-world/live") {
		t.Fatalf("note live fallback should render requested path")
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
