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
- MUST require `task test` to pass when defined and include `go test ./...` (or explicit scoped equivalent).
- MUST require `task fix` to run `gofmt` before merge.

### Working Agreements
- MUST follow root interaction protocol from [../AGENTS.md](../AGENTS.md) before finalizing policy changes.
- MUST ask user to choose test/lint scope when scope is ambiguous (`all packages` vs `subset`).

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
- Keep styling minimal and terminal-like in `static/tui.css`.
- Default run command from `blog/`: `go run .`
- Optional env: `BLOG_ROOT_URL` (used by markdown link formatter to normalize same-domain absolute URLs).
