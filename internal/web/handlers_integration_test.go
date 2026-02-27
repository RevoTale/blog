package web

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"net/http/httptest"

	"blog/internal/config"
	"blog/internal/notes"
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

func newTestMux(t *testing.T) *http.ServeMux {
	t.Helper()

	svc := notes.NewService(fakeGraphQLClient{}, 12, "")
	handler, err := NewHandler(config.Config{StaticDir: "../../static"}, svc)
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	mux := http.NewServeMux()
	handler.Register(mux)
	return mux
}

func requireBody(t *testing.T, body io.Reader) string {
	t.Helper()

	content, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(content)
}

func performRequest(mux *http.ServeMux, method string, path string) *httptest.ResponseRecorder {
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
		{path: "/notes", mustContain: "<title>Notes :: blog</title>"},
		{path: "/notes?author=l-you&tag=go&type=short", mustContain: "<h1>L You</h1>"},
		{path: "/note/hello-world", mustContain: "<title>Hello World :: blog</title>"},
		{path: "/author/l-you", mustContain: "<title>L You :: blog</title>"},
		{path: "/author/l-you?author=zed", mustContain: "<title>L You :: blog</title>"},
		{path: "/tag/go", mustContain: "<title>#Go :: blog</title>"},
		{path: "/tag/go?tag=rust", mustContain: "<title>#Go :: blog</title>"},
		{path: "/notes/tales", mustContain: "<title>Tales :: blog</title>"},
		{path: "/notes/tales?type=short", mustContain: "<title>Tales :: blog</title>"},
		{path: "/notes/micro-tales", mustContain: "<title>Micro-tales :: blog</title>"},
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

	root := performRequest(mux, http.MethodGet, "/notes")
	rootBody := requireBody(t, root.Body)
	if !strings.Contains(rootBody, `href="/author/l-you"`) {
		t.Fatalf("root notes missing canonical author link")
	}
	if !strings.Contains(rootBody, `href="/tag/go"`) {
		t.Fatalf("root notes missing canonical tag link")
	}
	if !strings.Contains(rootBody, `href="/notes/tales"`) {
		t.Fatalf("root notes missing tales route link")
	}
	if !strings.Contains(rootBody, `href="/notes/micro-tales"`) {
		t.Fatalf("root notes missing micro-tales route link")
	}

	filtered := performRequest(mux, http.MethodGet, "/author/l-you?tag=go&type=short")
	filteredBody := requireBody(t, filtered.Body)
	if !strings.Contains(filteredBody, `href="/notes"`) {
		t.Fatalf("filtered page missing All link to /notes")
	}
	if !strings.Contains(filteredBody, `href="/notes?tag=go&amp;type=short"`) {
		t.Fatalf("filtered page missing ANY author clear link")
	}
	if !strings.Contains(filteredBody, `href="/notes?author=l-you&amp;type=short"`) {
		t.Fatalf("filtered page missing ANY tag clear link")
	}
	if !strings.Contains(filteredBody, `href="/notes?author=l-you&amp;tag=go"`) {
		t.Fatalf("filtered page missing ANY type clear link")
	}
	if !strings.Contains(filteredBody, `href="/notes?author=zed&amp;tag=go&amp;type=short"`) {
		t.Fatalf("filtered page missing merged author link")
	}
	if !strings.Contains(filteredBody, `href="/notes?author=l-you&amp;tag=rust&amp;type=short"`) {
		t.Fatalf("filtered page missing merged tag link")
	}
	if !strings.Contains(filteredBody, `href="/notes?author=l-you&amp;tag=go&amp;type=long"`) {
		t.Fatalf("filtered page missing merged tales type link")
	}
}

func TestHandlerLiveRoutesReturnPatch(t *testing.T) {
	t.Parallel()
	mux := newTestMux(t)

	cases := []struct {
		path     string
		selector string
	}{
		{path: "/notes/live", selector: "#notes-content"},
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

	recMissingNote := performRequest(mux, http.MethodGet, "/note/missing")
	if recMissingNote.Code != http.StatusNotFound {
		t.Fatalf("missing note status: expected %d, got %d", http.StatusNotFound, recMissingNote.Code)
	}
	_ = requireBody(t, recMissingNote.Body)

	recMissingAuthor := performRequest(mux, http.MethodGet, "/author/missing")
	if recMissingAuthor.Code != http.StatusNotFound {
		t.Fatalf("missing author status: expected %d, got %d", http.StatusNotFound, recMissingAuthor.Code)
	}
	_ = requireBody(t, recMissingAuthor.Body)

	recMissingTag := performRequest(mux, http.MethodGet, "/tag/missing")
	if recMissingTag.Code != http.StatusNotFound {
		t.Fatalf("missing tag status: expected %d, got %d", http.StatusNotFound, recMissingTag.Code)
	}
	_ = requireBody(t, recMissingTag.Body)

	recNoLive := performRequest(mux, http.MethodGet, "/note/hello-world/live")
	if recNoLive.Code != http.StatusNotFound {
		t.Fatalf("note live status: expected %d, got %d", http.StatusNotFound, recNoLive.Code)
	}
	_ = requireBody(t, recNoLive.Body)
}
