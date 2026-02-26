package router

import (
	"testing"
	"testing/fstest"
)

func TestAppRouterMatch(t *testing.T) {
	router, err := NewAppRouter(fstest.MapFS{
		"app/layout.templ":               {Data: []byte("package web")},
		"app/notes/page.templ":           {Data: []byte("package web")},
		"app/note/[slug]/page.templ":     {Data: []byte("package web")},
		"app/author/[slug]/page.templ":   {Data: []byte("package web")},
		"app/author/[slug]/layout.templ": {Data: []byte("package web")},
		"app/author/settings/page.templ": {Data: []byte("package web")},
		"app/author/[slug]/live/page.templ": {
			Data: []byte("package web"),
		},
	}, "app")
	if err != nil {
		t.Fatalf("new app router: %v", err)
	}

	tests := []struct {
		name        string
		path        string
		expectedID  string
		expectedKey string
		expectedVal string
	}{
		{name: "notes", path: "/notes", expectedID: "notes"},
		{name: "notes trailing slash", path: "/notes/", expectedID: "notes"},
		{
			name:        "note wildcard",
			path:        "/note/hello-world",
			expectedID:  "note/[slug]",
			expectedKey: "slug",
			expectedVal: "hello-world",
		},
		{name: "author static precedence", path: "/author/settings", expectedID: "author/settings"},
		{
			name:        "author wildcard",
			path:        "/author/nina",
			expectedID:  "author/[slug]",
			expectedKey: "slug",
			expectedVal: "nina",
		},
		{
			name:        "author nested wildcard",
			path:        "/author/nina/live",
			expectedID:  "author/[slug]/live",
			expectedKey: "slug",
			expectedVal: "nina",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			match, ok := router.Match(tc.path)
			if !ok {
				t.Fatalf("expected a match for %q", tc.path)
			}
			if match.ID != tc.expectedID {
				t.Fatalf("expected route id %q, got %q", tc.expectedID, match.ID)
			}
			if tc.expectedKey == "" {
				return
			}

			value, ok := match.Param(tc.expectedKey)
			if !ok {
				t.Fatalf("expected param %q", tc.expectedKey)
			}
			if value != tc.expectedVal {
				t.Fatalf("expected param %q=%q, got %q", tc.expectedKey, tc.expectedVal, value)
			}
		})
	}
}

func TestAppRouterConflict(t *testing.T) {
	_, err := NewAppRouter(fstest.MapFS{
		"app/author/[slug]/page.templ": {Data: []byte("package web")},
		"app/author/[id]/page.templ":   {Data: []byte("package web")},
	}, "app")
	if err == nil {
		t.Fatal("expected conflict error, got nil")
	}
}

func TestMatchPathPattern(t *testing.T) {
	params, ok := MatchPathPattern("/author/[slug]/live", "/author/nina/live")
	if !ok {
		t.Fatal("expected wildcard pattern to match")
	}
	if params["slug"] != "nina" {
		t.Fatalf("expected slug to be %q, got %q", "nina", params["slug"])
	}

	if _, ok = MatchPathPattern("/author/[slug]/live", "/author/nina"); ok {
		t.Fatal("expected mismatch for shorter path")
	}
}

func TestIsValidSlug(t *testing.T) {
	if !IsValidSlug("l-you") {
		t.Fatal("expected l-you to be a valid slug")
	}
	if IsValidSlug("bad slug") {
		t.Fatal("expected slug with spaces to be invalid")
	}
}
