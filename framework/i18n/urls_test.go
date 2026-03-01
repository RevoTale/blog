package i18n

import "testing"

func TestLocalizePathAndStripLocale(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Locales:       []string{"en", "uk", "de"},
		DefaultLocale: "en",
		PrefixMode:    PrefixAsNeeded,
	}

	if got := LocalizePath(cfg, "en", "/note/hello"); got != "/note/hello" {
		t.Fatalf("localized default path: expected %q, got %q", "/note/hello", got)
	}
	if got := LocalizePath(cfg, "uk", "/note/hello"); got != "/uk/note/hello" {
		t.Fatalf("localized non-default path: expected %q, got %q", "/uk/note/hello", got)
	}

	locale, stripped, hadPrefix, ok := StripLocale(cfg, "/uk/note/hello")
	if !ok {
		t.Fatalf("expected strip locale success")
	}
	if locale != "uk" || stripped != "/note/hello" || !hadPrefix {
		t.Fatalf(
			"unexpected strip result locale=%q stripped=%q hadPrefix=%t",
			locale,
			stripped,
			hadPrefix,
		)
	}

	_, _, _, ok = StripLocale(cfg, "/it/note/hello")
	if ok {
		t.Fatalf("expected unsupported locale-like prefix to fail")
	}
}
