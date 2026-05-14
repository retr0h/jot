# CLAUDE.md

Architecture intent + standards for Claude Code working in this repo.

## Project

**jot** — terminal-first markdown notes + todos with linked tasks. Notes
are plain `.md` files edited in `$EDITOR` (nvim). `@task` markers create
linked todos with due dates and labels. SQLite indexes everything; FTS5
powers search. Sensitive notes encrypt at rest via kvlt (age + SSH keys).
An MCP server exposes the full surface to LLM agents.

## Architecture in one screen

```
jot (binary)
├── cmd/            cobra CLI tree
├── internal/
│   ├── cli/        theme (maxheadroom palette), banner
│   ├── jot/        store (SQLite), parser (@task), editor, kvlt secure notes
│   ├── tui/        full-screen bubbletea app (notes list, tasks, preview)
│   ├── mcp/        MCP server (stdio, meshx pattern)
│   └── version/    build-time identity
```

Key invariants:

- **Markdown is source of truth.** SQLite indexes; `.md` files are canonical.
- **@task round-trips.** Parser reads both `@task desc` and
  `@task(desc | due:X | label:Y)` forms — edits in nvim are always valid.
- **kvlt for secrets.** Secure note bodies live in `.kvlt/secrets/`; SQLite
  indexes title/labels/tasks only. `github.com/retr0h/kvlt/pkg/kvlt` imported
  as a library, not shelled out.

## Code standards

- **Conventional Commits** for messages.
- **Multi-line function signatures** for any function with 2+ params.
- **golangci-lint** chain: `errcheck`, `errname`, `govet`, `prealloc`,
  `predeclared`, `revive`, `staticcheck`. `just ready` runs the full
  format + lint suite locally.
- **Tests, not test plans** — every PR ships with tests.
  Table-driven, `foo.go ↔ foo_test.go` file pairing.
- **No inline hex colors** — palette constants live in
  `internal/cli/theme.go`.

## Color palette (Max Headroom)

```
#ffb86c  orange    accent, banner
#00d4ff  cyan      info hints
#c678dd  magenta   (reserved)
#50fa7b  green     success, done tasks
#e5c07b  yellow    (reserved)
#ff6ec7  pink      errors, overdue tasks
#6272a4  lavender  muted states
#c0caf5  fg        default text
#3b4261  drained   separators, dim text
```

## Building

```bash
go build -o jot .
go run . --help
```

## Data directory

`~/.config/jot/` by default. Override via `--config` flag or `JOT_CONFIG_DIR` env.
