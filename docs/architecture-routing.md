# Routing And Generation Architecture

## Summary

This project uses a split generation model:

- Route modules from `internal/web/app` are generated into `internal/web/gen`.
- Components from `internal/web/components` are generated in place (`*_templ.go` next to the source file).
- Resolver contracts are generated into `internal/web/resolvers/generated.go`.
- Resolver implementations are handwritten in `internal/web/resolvers/*.go`.

## Why `internal/web/gen` Exists

Route directories can contain dynamic segments like `[slug]`.
Those names are valid for route discovery but not safe as Go package import paths.

Generating route modules into `internal/web/gen` solves this by:

- using deterministic, Go-safe package names such as `r_page_author_param_slug`
- keeping generated route wrappers separate from authored route templates
- giving registry wiring a stable namespace to import from

## Source Of Truth

- `internal/web/app/**/page.templ`: route page modules
- `internal/web/app/**/layout.templ`: route layout modules
- `internal/web/app/**/404.templ`: route-level not-found modules
- `internal/web/components/*.templ`: reusable UI components
- `internal/web/resolvers/*.go`: business data loading for routes

## Generated Outputs

- `internal/web/gen/r_*/...`: copied route templates with generated package names
- `internal/web/gen/registry_gen.go`: route registry, parameter parsing, not-found composition, `NewRouteResolvers`
- `internal/web/resolvers/generated.go`: `RouteResolver` interface + param structs + compile assertion
- `internal/web/components/*_templ.go`: templ output for components

All generated files include `DO NOT EDIT` headers and must not be edited manually.

## Resolver Contract Boundary

The generated resolver namespace (`internal/web/resolvers/generated.go`) is authoritative.

- Generated file defines route param types and the `RouteResolver` interface.
- Handwritten files in the same package implement methods such as `Resolve<...>Page`.
- Missing methods fail at compile time via:
  `var _ RouteResolver = (*Resolver)(nil)`.

This keeps route signatures generator-owned while data logic stays handwritten.

## Runtime Behavior

- Canonical routes are served by framework runtime (`framework/engine` + `framework/httpserver`).
- HTMX partial updates use canonical URLs with `HX-Request: true`.
- Full-page rendering and partial rendering share the same page resolver method.

## Update Workflow

1. Edit templates in `internal/web/app` or `internal/web/components`.
2. Edit resolver implementation files in `internal/web/resolvers/*.go`.
3. Run generation: `task gen`.
4. Validate: `go test ./...` and `task gen:check` (in clean CI state).

