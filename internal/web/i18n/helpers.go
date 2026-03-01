package i18n

import frameworki18n "blog/framework/i18n"

func MustResolver() *frameworki18n.Resolver {
	resolver, err := frameworki18n.NewResolver(Config())
	if err != nil {
		panic(err)
	}
	return resolver
}
