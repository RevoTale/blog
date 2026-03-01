package i18n

import "strings"

type RouteDecision struct {
	Locale          string
	OriginalPath    string
	StrippedPath    string
	CanonicalPath   string
	ShouldRedirect  bool
	NotFound        bool
	HadLocalePrefix bool
}

type Resolver struct {
	config Config
}

func NewResolver(cfg Config) (*Resolver, error) {
	normalized, err := NormalizeConfig(cfg)
	if err != nil {
		return nil, err
	}

	return &Resolver{config: normalized}, nil
}

func (resolver *Resolver) Config() Config {
	if resolver == nil {
		return Config{}
	}
	return resolver.config
}

func (resolver *Resolver) Resolve(requestPath string) RouteDecision {
	normalizedPath := NormalizePath(requestPath)
	decision := RouteDecision{
		OriginalPath: normalizedPath,
		StrippedPath: normalizedPath,
	}
	if resolver == nil {
		return decision
	}

	locale, strippedPath, hadPrefix, ok := StripLocale(resolver.config, normalizedPath)
	if !ok {
		return RouteDecision{
			OriginalPath:  normalizedPath,
			StrippedPath:  normalizedPath,
			CanonicalPath: normalizedPath,
			NotFound:      true,
		}
	}

	decision.Locale = locale
	decision.StrippedPath = strippedPath
	decision.HadLocalePrefix = hadPrefix
	decision.CanonicalPath = LocalizePath(resolver.config, locale, strippedPath)
	decision.ShouldRedirect = decision.CanonicalPath != normalizedPath

	switch resolver.config.PrefixMode {
	case PrefixNever:
		if hadPrefix {
			decision.ShouldRedirect = true
			decision.CanonicalPath = strippedPath
		}
	case PrefixAlways:
		if !hadPrefix {
			decision.ShouldRedirect = true
			decision.CanonicalPath = prefixedPath(locale, strippedPath)
		}
	case PrefixAsNeeded:
		if hadPrefix && strings.EqualFold(locale, resolver.config.DefaultLocale) {
			decision.ShouldRedirect = true
			decision.CanonicalPath = strippedPath
		}
	}

	if strings.TrimSpace(decision.CanonicalPath) == "" {
		decision.CanonicalPath = normalizedPath
	}

	return decision
}
