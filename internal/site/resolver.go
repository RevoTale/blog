package site

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"blog/internal/config"
)

type Resolver interface {
	CanonicalURL() string
	Resolve(*http.Request) string
}

type staticResolver struct {
	canonicalURL string
}

func NewResolver(cfg config.Config) (Resolver, error) {
	canonicalURL, err := normalizeCanonicalURL(cfg.RootURL)
	if err != nil {
		return nil, err
	}

	return staticResolver{canonicalURL: canonicalURL}, nil
}

func (resolver staticResolver) CanonicalURL() string {
	return resolver.canonicalURL
}

func (resolver staticResolver) Resolve(*http.Request) string {
	return resolver.CanonicalURL()
}

func normalizeCanonicalURL(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("BLOG_ROOT_URL is required and must be an absolute URL")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("parse BLOG_ROOT_URL %q: %w", trimmed, err)
	}
	if !parsed.IsAbs() || strings.TrimSpace(parsed.Host) == "" {
		return "", fmt.Errorf("BLOG_ROOT_URL %q must be absolute", trimmed)
	}

	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}
