# Color System Architecture

## Goal

Use one source of truth for visual colors so dark/light theme changes are predictable and consistent.

## Single Source Of Truth

All themeable colors are defined as CSS custom properties in:

- `internal/web/static/tui.css`

Theme values are set in two places only:

1. `:root` for default (dark) values.
2. `@media (prefers-color-scheme: light) { :root { ... } }` for light overrides.

Component selectors must consume variables (`var(--token)`) instead of hardcoded hex/rgba values.

## Token Layers

Use semantic tokens, not route/component-local color literals.

- Foundation tokens:
  - `--bg-*`, `--text-*`, `--border-*`, `--divider`, `--focus-ring`, `--accent-*`
- Surface/state tokens:
  - `--topbar-*`, `--topbar-search-*`, `--content-header-*`, `--feed-toolbar-*`, `--note-detail-*`, `--footer-*`, `--server-*`, `--channel-*`, `--presence-*`, `--note-open-badge-*`
- Optional brand/status tokens:
  - reserved for stable accents (for example online state green).

## Naming Rules

- Tokens describe role, not raw color:
  - Good: `--topbar-search-focus-border`
  - Avoid: `--blue-400`
- Keep token names stable even if values change.
- Group related tokens by prefix.

## Implementation Rules

1. Add/modify token values in `:root` and light `:root` override.
2. Reference tokens in selectors.
3. Avoid selector-level light-theme overrides unless layout behavior differs (not color).
4. Avoid introducing new hardcoded colors in component rules for themeable surfaces.

## Current Coverage

Top/search/mid bars are token-driven:

- `.topbar` uses `--topbar-bg`
- `.topbar-search` and its states use `--topbar-search-*`
- buttons/clear states use `--topbar-search-submit-*` and `--topbar-search-clear-*`
- `.context-panel` uses `--content-header-bg`
- `.feed-toolbar` uses `--feed-toolbar-bg`
- `.note-detail` uses `--note-detail-bg`
- `.footer` uses `--footer-bg` and `--footer-link`
- code/media/empty surfaces use `--code-surface-bg`, `--media-surface-bg`, `--empty-state-bg`
- unread/read badge dot uses `--note-open-badge-*`

This ensures all bar surfaces switch with the system theme from a centralized token set.

## Update Workflow

1. Change token values in `tui.css`.
2. Verify dark and light in browser.
3. Run project checks:
   - `go test ./internal/web/...`
