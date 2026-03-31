# Routing And Generation Architecture

## Summary

This project uses a split generation model:

- Route modules from `web/routes` are generated into `web/generated`.
- Components from `web/components` are generated in place (`*_templ.go` next to the source file).
- Resolver contracts are generated into `web/resolvers/generated.go`.
- Resolver implementations are handwritten in `web/resolvers/*.go`.
- The generated namespace is the `App Bundle` boundary used by the preferred `httpserver.NewApp(...)` integration.

## Why `web/generated` Exists

Route directories can contain dynamic segments like `[slug]`.
Those names are valid for route discovery but not safe as Go package import paths.

Generating route modules into `web/generated` solves this by:

- using deterministic, Go-safe package names such as `r_page_author_param_slug`
- keeping generated route wrappers separate from authored route templates
- giving the generated app contract a stable namespace to import from

## Source Of Truth

- `web/routes/**/page.templ`: route page modules
- `web/routes/**/layout.templ`: route layout modules
- `web/routes/**/404.templ`: route-level not-found modules
- `web/components/*.templ`: reusable UI components
- `web/resolvers/*.go`: business data loading for routes

## Generated Outputs

- `web/generated/r_*/...`: copied route templates with generated package names
- `web/generated/registry_gen.go`: route registry, parameter parsing, not-found composition, `NewRouteResolvers`
- `web/resolvers/generated.go`: `RouteResolver` interface + param structs + compile assertion
- `web/components/*_templ.go`: templ output for components
- The target integration treats `web/generated` as the `App Bundle` boundary consumed by `httpserver.NewApp(...)`.
- The current `cmd/server/handler.go` remains transitional and is not the intended long-term integration model.

All generated files include `DO NOT EDIT` headers and must not be edited manually.

## App Bundle Boundary

The preferred runtime model is:

```go
handler, err := httpserver.NewApp(httpserver.Config[*runtime.Context]{
	App:    generated.Bundle(appContext),
	Custom: customConfig,
})
```

In that model:

- `generated.Bundle(appContext)` is the generated `App Bundle`
- `httpserver.NewApp(...)` is the happy-path framework entrypoint
- app-owned hooks live in `Custom Config`
- advanced composition may live in any app-owned package when needed

## Resolver Contract Boundary

The generated resolver namespace (`web/resolvers/generated.go`) is authoritative.

- Generated file defines route param types and the `RouteResolver` interface.
- Handwritten files in the same package implement methods such as `Resolve<...>Page`.
- Missing methods fail at compile time via:
  `var _ RouteResolver = (*Resolver)(nil)`.

This keeps route signatures generator-owned while data logic stays handwritten.

## Runtime Behavior

- Canonical routes are served by the `no-js` framework runtime
  (`github.com/RevoTale/no-js/framework/engine` + `github.com/RevoTale/no-js/framework/httpserver`).
- `main.go` should stay focused on environment loading and business service construction.
- Site and canonical-domain policy should be centralized through a `Site Resolver`.
- HTMX partial updates use canonical URLs with `HX-Request: true`.
- Full-page rendering and partial rendering share the same page resolver method.

## Update Workflow

1. Edit templates in `web/routes` or `web/components`.
2. Edit resolver implementation files in `web/resolvers/*.go`.
3. Run generation: `task gen`.
4. Validate: `go test ./...` and `task gen:check` (in clean CI state).
