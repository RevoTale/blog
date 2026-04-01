package site

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"blog/internal/config"
	frameworksite "github.com/RevoTale/no-js/framework/site"
)

type staticResolver struct {
	canonicalRootURL *url.URL
}

func NewResolver(cfg config.Config) (frameworksite.Resolver, error) {
	canonicalRootURL, err := normalizeCanonicalURL(cfg.RootURL)
	if err != nil {
		return nil, err
	}

	return staticResolver{canonicalRootURL: canonicalRootURL}, nil
}

func (resolver staticResolver) CanonicalURL() string {
	root := resolver.CanonicalRoot()
	if root == nil {
		return ""
	}
	return root.String()
}

func (resolver staticResolver) Resolve(*http.Request) string {
	return resolver.CanonicalURL()
}

func (resolver staticResolver) CanonicalRoot() *url.URL {
	if resolver.canonicalRootURL == nil {
		return nil
	}

	root := *resolver.canonicalRootURL
	return &root
}

func (resolver staticResolver) ResolveRoot(*http.Request) *url.URL {
	return resolver.CanonicalRoot()
}

func normalizeCanonicalURL(value string) (*url.URL, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, fmt.Errorf("BLOG_ROOT_URL is required and must be an absolute URL")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return nil, fmt.Errorf("parse BLOG_ROOT_URL %q: %w", trimmed, err)
	}
	if !parsed.IsAbs() || strings.TrimSpace(parsed.Host) == "" {
		return nil, fmt.Errorf("BLOG_ROOT_URL %q must be absolute", trimmed)
	}

	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed, nil
}
