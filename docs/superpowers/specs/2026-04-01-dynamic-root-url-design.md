# Dynamic Root URL Design

## Summary

The app currently uses two incompatible root URL patterns:

- discovery endpoints already resolve the root URL per request
- page loaders, metadata builders, and note markdown rendering still depend on a static app-global root URL

This design standardizes root URL handling around the framework runtime. The framework will expose a request-scoped root URL resolver on `RuntimeContext`, with a default request-derived implementation and an app override hook for static or custom policies.

## Goals

- Remove the app-global resolved root URL as a primary runtime dependency.
- Make root URL resolution available anywhere that already has `RuntimeContext` and `*http.Request`.
- Preserve a simple static configuration path for apps that want a fixed root URL.
- Keep canonical URL, alternates, discovery output, structured data, and note markdown link normalization consistent.

## Non-Goals

- Canonical host redirect enforcement.
- Trust-policy hardening for reverse proxy deployments beyond the existing request-origin behavior.
- A broad refactor of unrelated app wiring or i18n routing.

## Current Problems

### Split behavior

- [`web/routes/discovery_helpers.go`](/workspaces/blog/web/routes/discovery_helpers.go) already resolves the root URL from the request through the site resolver.
- [`web/view/context.go`](/workspaces/blog/web/view/context.go) exposes `RootURL()` from a static resolver canonical URL.
- [`web/seo/metadata.go`](/workspaces/blog/web/seo/metadata.go) and [`web/view/loaders.go`](/workspaces/blog/web/view/loaders.go) depend on `appCtx.RootURL()`.
- [`internal/notes/service.go`](/workspaces/blog/internal/notes/service.go) stores `rootURL` on the service and uses it later for markdown link normalization.

### Architectural mismatch

The request is the source of truth for dynamic root URL resolution, but the app context is long-lived and shared across requests. A resolved absolute root URL therefore does not belong in app context state.

## Decision

Use the framework runtime as the root URL boundary.

The framework will accept an optional generic resolver hook:

```go
type RootURLResolver[C any] func(appCtx C, r *http.Request) (string, error)
```

The runtime contract will expose request-scoped resolution:

```go
type RuntimeContext[C any] interface {
	AppContext() C
	ResolveRootURL(r *http.Request) (string, error)
	// existing methods...
}
```

Apps may omit the hook and use the framework default request-derived behavior, or provide a custom resolver that returns a static value or any app-specific policy.

## Framework Design

### API placement

The resolver hook belongs on the generic `App Bundle` boundary, not `Custom Config`.

Reasoning:

- `App Bundle` is already generic over app context.
- `Custom Config` is currently non-generic in [`no-js/framework/httpserver/newapp.go`](/workspaces/blog/no-js/framework/httpserver/newapp.go).
- Moving this hook to `Custom Config` would force a larger breaking config redesign than this feature needs.

### Runtime flow

1. `httpserver.NewApp(...)` receives `AppBundle.RootURLResolver`.
2. `httpserver.New(...)` passes the resolver into `engine.New(...)`.
3. `engine.Engine` stores the resolver and implements `ResolveRootURL(r)`.
4. Route handlers, page loaders, metadata builders, and discovery handlers call `runtime.ResolveRootURL(r)`.

### Default behavior

If the app does not provide a resolver, the framework uses a default request-based resolver.

That default:

- derives scheme and host from the request
- reuses the same request-origin logic already used for discovery absolute URLs
- returns an absolute root URL string without query or fragment

This keeps the default behavior aligned with existing framework behavior instead of introducing a second origin parser.

### Static override

Apps that want a fixed root URL can provide:

```go
func StaticRootURL[C any](root string) RootURLResolver[C] {
	return func(C, *http.Request) (string, error) {
		return root, nil
	}
}
```

Invalid static values remain startup-time validation errors if the app validates them before installing the resolver.

## Blog App Design

### App wiring

[`cmd/server/main.go`](/workspaces/blog/cmd/server/main.go) will stop constructing a blog-local site resolver as a required startup dependency.

Instead:

- when `BLOG_ROOT_URL` is set, install a static framework resolver
- when `BLOG_ROOT_URL` is empty, rely on the framework default request-derived resolver

This keeps local development simple while removing the requirement that all deployments define a static global root.

### App context changes

[`web/view/context.go`](/workspaces/blog/web/view/context.go) will stop exposing a resolved `RootURL()` helper and stop owning a root URL resolver dependency.

The app context will keep long-lived app services only.

### Request-time call sites

The following paths will resolve the root URL from runtime at request time:

- discovery helpers in [`web/routes/discovery_helpers.go`](/workspaces/blog/web/routes/discovery_helpers.go)
- metadata generation in [`web/seo/metadata.go`](/workspaces/blog/web/seo/metadata.go)
- view-model loading in [`web/view/loaders.go`](/workspaces/blog/web/view/loaders.go)

The result will be passed explicitly into helper functions and view models instead of being read from app-global state.

### Notes service

[`internal/notes/service.go`](/workspaces/blog/internal/notes/service.go) currently stores `rootURL` on the service and applies it during markdown rendering in `GetNoteBySlug`.

That field will be removed. Request-specific root URL will be passed when rendering note content through a small request-scoped options struct accepted by note detail loading.

The notes service will remain reusable across requests and stop carrying request-derived state.

## Error Handling

### Startup-time errors

- malformed configured static root URL
- invalid app-provided static resolver setup

### Request-time errors

- resolver failure while building canonical metadata, alternates, or structured data
- missing host information for request-derived resolution when an absolute URL is required

Request-time root URL failures should surface as normal handler errors. They should not silently fall back to empty strings when absolute URLs are required for the response being built.

## Testing Strategy

### Framework tests

Add tests for:

- default request-derived resolution with plain `Host`
- `https` resolution from TLS
- `X-Forwarded-Proto` precedence
- missing host behavior
- static resolver override behavior

### Blog tests

Update or add tests covering:

- metadata canonical and alternates under request-derived resolution
- feed, robots, and sitemap documents under request-derived resolution
- static configured root override behavior
- note page markdown link normalization using the request-resolved root URL

## Migration Notes

- `BLOG_ROOT_URL` becomes optional rather than required.
- The blog-local `Site Resolver` can be removed after all app call sites have migrated to `runtime.ResolveRootURL(r)`.
- Existing apps using the framework do not need to provide a resolver immediately because the framework default preserves usable behavior.

## Risks

### Proxy trust assumptions

The default request-derived resolver inherits the current framework request-origin assumptions. Deployments behind proxies must still ensure forwarded headers are set correctly.

### Partial migration risk

If any call site continues reading a static app-global root URL while others use request-time resolution, canonical URLs and structured data may disagree. The migration must remove mixed behavior completely.

## Acceptance Criteria

- No app-global resolved root URL is required for normal operation.
- `RuntimeContext` exposes request-scoped root URL resolution.
- The blog app can run with dynamic request-derived root URLs and with an explicit static override.
- Metadata, discovery documents, structured data, and note markdown behavior all use the same resolved root URL source for a given request.
