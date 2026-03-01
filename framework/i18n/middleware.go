package i18n

import (
	"net/http"
	"strings"
)

type MiddlewareConfig struct {
	Resolver       *Resolver
	BypassPrefixes []string
	BypassExact    []string
}

func Middleware(cfg MiddlewareConfig) func(http.Handler) http.Handler {
	byPassExact := normalizedExact(cfg.BypassExact)
	byPassPrefixes := normalizedPrefixes(cfg.BypassPrefixes)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if next == nil {
				return
			}
			if r == nil || r.URL == nil || cfg.Resolver == nil {
				next.ServeHTTP(w, r)
				return
			}

			originalPath := NormalizePath(r.URL.Path)
			if shouldBypass(originalPath, byPassExact, byPassPrefixes) {
				next.ServeHTTP(w, r)
				return
			}

			decision := cfg.Resolver.Resolve(originalPath)
			if decision.NotFound {
				http.NotFound(w, r)
				return
			}
			if decision.ShouldRedirect {
				target := decision.CanonicalPath
				if strings.TrimSpace(r.URL.RawQuery) != "" {
					target += "?" + r.URL.RawQuery
				}
				http.Redirect(w, r, target, http.StatusPermanentRedirect)
				return
			}

			rewrittenRequest := r.Clone(
				WithRequestInfo(
					r.Context(),
					RequestInfo{
						Locale:       decision.Locale,
						OriginalPath: decision.OriginalPath,
						StrippedPath: decision.StrippedPath,
					},
				),
			)
			rewrittenRequest.URL.Path = decision.StrippedPath
			if strings.TrimSpace(rewrittenRequest.URL.RawPath) != "" {
				rewrittenRequest.URL.RawPath = decision.StrippedPath
			}

			next.ServeHTTP(w, rewrittenRequest)
		})
	}
}

func normalizedExact(paths []string) map[string]struct{} {
	out := make(map[string]struct{}, len(paths))
	for _, candidate := range paths {
		pathValue := NormalizePath(candidate)
		out[pathValue] = struct{}{}
	}
	return out
}

func normalizedPrefixes(prefixes []string) []string {
	out := make([]string, 0, len(prefixes))
	for _, candidate := range prefixes {
		trimmed := strings.TrimSpace(candidate)
		if trimmed == "" {
			continue
		}
		prefix := NormalizePath(trimmed)
		if !strings.HasSuffix(prefix, "/") {
			prefix += "/"
		}
		out = append(out, prefix)
	}
	return out
}

func shouldBypass(pathValue string, exact map[string]struct{}, prefixes []string) bool {
	if _, ok := exact[pathValue]; ok {
		return true
	}

	for _, prefix := range prefixes {
		if strings.HasPrefix(pathValue+"/", prefix) || strings.HasPrefix(pathValue, prefix) {
			return true
		}
	}
	return false
}
