# RevoTale Blog

Fast, server-rendered blog for RevoTale, built with Go.

Live:
- https://blog.revotale.com
- https://revotale.com/blog (legacy entrypoint; redirected; was built with NextJs + ShadCN)

Source:
- https://github.com/RevoTale/blog

## What This Is

This project is a public app that renders internal CMS in the form of blog (notes, authors, tags, RSS, sitemap, robots). Internally in RevoTale infrastructure it is a submodule which is separated from the `cms` runtime.
Content is fetched by this app through GraphQL.

## Why This Exists

This project was migrated from the older Next.js + RSC + ShadCN blog path to reduce server/runtime overhead and improve perceived performance (especially RAM usage) for a read-heavy blog.

The `cms` app now redirects legacy `/blog/*` routes to this dedicated blog runtime.

## Stack

- Go HTTP server + custom lightweight framework layer ( Im trying framework APIs similar to NextJs experience)
- [`templ`](https://templ.guide/) for SSR components/layouts
- [`htmx`](https://htmx.org/) for partial/live navigation updates
- [`go-i18n`](https://github.com/nicksnyder/go-i18n) for localization
- [`esbuild`](https://esbuild.github.io/) for static asset bundling/fingerprints



## What This Project Optimizes For

- Low-overhead SSR with predictable caching behavior
- Early document streaming (head-first) and faster first bytes
- SEO completeness (metadata, canonical, alternates, feeds, sitemaps, robots)
- Strong i18n support across routes
- Simple, inspectable architecture suitable for low resource self-hosted infrastructure

## Local Development (Minimal)

```bash
task gen
BLOG_ROOT_URL=http://localhost:8080 \
BLOG_GRAPHQL_ENDPOINT=http://localhost:3000/api/graphql \
go run .
```

Validation:

```bash
task validate
task test
```