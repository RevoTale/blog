package i18n

import "testing"

func TestResolverResolveAsNeeded(t *testing.T) {
	t.Parallel()

	resolver, err := NewResolver(Config{
		Locales:       []string{"en", "uk", "de"},
		DefaultLocale: "en",
		PrefixMode:    PrefixAsNeeded,
	})
	if err != nil {
		t.Fatalf("new resolver: %v", err)
	}

	root := resolver.Resolve("/")
	if root.Locale != "en" || root.StrippedPath != "/" || root.ShouldRedirect {
		t.Fatalf("unexpected root decision: %#v", root)
	}

	localized := resolver.Resolve("/uk/note/hello")
	if localized.Locale != "uk" || localized.StrippedPath != "/note/hello" || localized.ShouldRedirect {
		t.Fatalf("unexpected localized decision: %#v", localized)
	}

	defaultPrefixed := resolver.Resolve("/en/note/hello")
	if !defaultPrefixed.ShouldRedirect {
		t.Fatalf("expected default prefixed path redirect")
	}
	if defaultPrefixed.CanonicalPath != "/note/hello" {
		t.Fatalf("canonical path: expected %q, got %q", "/note/hello", defaultPrefixed.CanonicalPath)
	}

	unknown := resolver.Resolve("/it/note/hello")
	if !unknown.NotFound {
		t.Fatalf("expected unknown locale-like prefix to be marked not-found")
	}
}
