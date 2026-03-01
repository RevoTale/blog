package i18n

import "context"

type RequestInfo struct {
	Locale       string
	OriginalPath string
	StrippedPath string
}

type requestInfoKey struct{}

func WithRequestInfo(ctx context.Context, info RequestInfo) context.Context {
	return context.WithValue(ctx, requestInfoKey{}, info)
}

func RequestInfoFromContext(ctx context.Context) (RequestInfo, bool) {
	if ctx == nil {
		return RequestInfo{}, false
	}

	info, ok := ctx.Value(requestInfoKey{}).(RequestInfo)
	return info, ok
}

func LocaleFromContext(ctx context.Context) string {
	info, ok := RequestInfoFromContext(ctx)
	if !ok {
		return ""
	}

	return info.Locale
}
