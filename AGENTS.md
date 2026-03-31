# AGENTS.md

## Overview
`blog/` is a Go server-rendered blog module that serves dynamic `/notes`, `/note/:slug`, and `/author/:slug`
pages from CMS GraphQL data.

## Project Structure
```text
<go-repo-root>/
  AGENTS.md
  .golangci.yml|.golangci.yaml|.golangci.toml
  Taskfile.yml
```

## Go Engineering Module

### Strict Rules
- MUST use `golangci-lint` as the Go linter: https://github.com/golangci/golangci-lint.
- MUST enforce a maximum line length of 120 through golangci-lint configuration.
- SHOULD prefer patterns from `100 Go Mistakes and How to Avoid Them`: https://github.com/teivah/100-go-mistakes.
- MUST configure the `lll` linter in golangci-lint with line length set to 120.
- MUST run golangci-lint against all Go packages through Taskfile tasks.
- MUST require `task validate` to run `golangci-lint run` and pass for Go changes before merge.
- MUST run `task validate` and get a passing result before an update is considered finished.
- MUST require `task test` to pass when defined and include `go test ./...` (or explicit scoped equivalent).
- MUST require `task fix` to run `gofmt` before merge.

### Working Agreements
- MUST follow root interaction protocol from [../AGENTS.md](../AGENTS.md) before finalizing policy changes.
- MUST ask user to choose test/lint scope when scope is ambiguous (`all packages` vs `subset`).
- MUST keep all user-facing copy localized via `web/i18n` typed keys; avoid hardcoded English text in
  `.templ` files and app-facing view-model helpers.
- MUST update all locale files in `web/i18n/messages/` when adding or changing message IDs.
- MUST use the canonical architecture terms in docs and plans: `App Bundle`, `Custom Config`, `Site Resolver`, and
  `Advanced composition`.
- MUST keep generic framework config separate from app-specific hooks and dependencies.
- MUST not introduce a broad `blog/web` facade package for server wiring.
- MUST not treat `web/bootstrap` as a contract term; advanced composition may live in any app-owned package.
- MUST treat wiring/config contract failures as startup errors; do not hide them with request-time nil fallbacks in
  route conventions or handlers.

## Taskfile Workflow Module

### Strict Rules
- MUST use [Taskfile](https://github.com/go-task/task) as the primary workflow entrypoint for generation, fixes,
  validation, and testing.
- MUST treat Taskfile execution from repository root as project-wide orchestration; nested Taskfiles MUST be composable
  from root tasks.
- MUST define workflow commands in `Taskfile.yml` unless a runtime/tool requires technology-specific files.
- MUST execute tests via Taskfile tasks instead of direct stack-specific test commands.
- MUST keep task interface names consistent as `gen`, `gen:check`, `gen:code-diff` (when needed), `fix`, `validate`, and
  `test`.
- MUST keep `task gen:check` non-mutating and return non-zero when generation would change outputs.
- MUST provide `task gen:code-diff` as CI fallback when generators do not support dry-run checks.
- SHOULD run `task validate` before `task test` when both tasks exist.
- MAY skip a task category when no relevant tools exist.
- MUST provide a VCS/code-diff generation check for CI use (`git diff --exit-code`).
- MUST compose reusable tasks and invoke them via `task:`.

### Working Agreements
- MUST follow root interaction protocol from [../AGENTS.md](../AGENTS.md) before finalizing policy changes.
- MUST ask for explicit `Accept` before approving deviations from standard task interface names.

## Implementation Notes
- GraphQL typed client generation: `github.com/Khan/genqlient`
  Link: https://github.com/Khan/genqlient
- Markdown rendering: `github.com/gomarkdown/markdown`
  Link: https://github.com/gomarkdown/markdown
- Keep data fetching server-side; templates receive pre-mapped view models.
- Keep styling minimal and terminal-like in `web/assets/tui.css`.
- Default run command from `blog/`: `go run .`
- Preferred target integration: `generated.Bundle(appContext)` passed to `httpserver.NewApp(...)`.
- Optional env: `BLOG_ROOT_URL` (currently used as canonical site input while site-resolution refactoring is in flight).
