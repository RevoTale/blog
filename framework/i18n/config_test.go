package i18n

import "testing"

func TestNormalizeConfig(t *testing.T) {
	t.Parallel()

	cfg, err := NormalizeConfig(Config{
		Locales:       []string{"EN", "uk", "uk", "de"},
		DefaultLocale: "EN",
		PrefixMode:    PrefixAsNeeded,
	})
	if err != nil {
		t.Fatalf("normalize config: %v", err)
	}

	if got := cfg.DefaultLocale; got != "en" {
		t.Fatalf("default locale: expected %q, got %q", "en", got)
	}
	if len(cfg.Locales) != 3 {
		t.Fatalf("locales length: expected %d, got %d", 3, len(cfg.Locales))
	}
	if got := cfg.PrefixMode; got != PrefixAsNeeded {
		t.Fatalf("prefix mode: expected %q, got %q", PrefixAsNeeded, got)
	}
}

func TestNormalizeConfigInvalid(t *testing.T) {
	t.Parallel()

	if _, err := NormalizeConfig(Config{
		Locales:       []string{"en"},
		DefaultLocale: "en",
		PrefixMode:    PrefixMode("invalid"),
	}); err == nil {
		t.Fatalf("expected invalid prefix mode error")
	}

	if _, err := NormalizeConfig(Config{
		Locales:       []string{"en", "broken-locale"},
		DefaultLocale: "en",
		PrefixMode:    PrefixAsNeeded,
	}); err == nil {
		t.Fatalf("expected invalid locale error")
	}
}
