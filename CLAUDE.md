# CLAUDE.md

Architecture intent + standards for Claude Code working in this repo.

## Project

**jot** — terminal-first markdown notes + todos with linked tasks. Notes
are plain `.md` files with YAML frontmatter, edited in nvim (hardcoded —
obsidian.nvim provides wiki-link navigation). `@task` markers with `#tags`
create linked todos with due dates. `[[slug]]` wiki links connect notes.
Git backs everything — every save is a commit. Sensitive notes encrypt at
rest via kvlt (age + SSH keys). An MCP server exposes the full surface to
LLM agents. No database: files are the source of truth.

## Architecture

```
jot (binary)
├── cmd/            cobra CLI tree
├── internal/
│   ├── cli/        theme, banner, output helpers, TTY prompts
│   ├── config/     Config struct, viper mapstructure bindings
│   ├── gitops/     git integration — commit, log, diff, show
│   ├── jot/        core domain — parser (@task), editor, slug, date, kvlt
│   ├── mcp/        MCP server (stdio, go-sdk pattern)
│   └── version/    build-time identity
```

## Key invariants

- **Files are source of truth.** No database — `.md` files are canonical.
- **Git backs everything.** Every write goes through `internal/gitops`.
- **@task round-trips.** Parser reads both `@task desc #tag` and
  `@task(desc | due:X) #tag` forms.
- **[[slug]] links.** Wiki-style links connect notes by slug.
- **Interfaces where consumed.** Define interfaces in the consuming package,
  not the providing package.

## Code standards

- **Conventional Commits** for messages.
- **Multi-line function signatures** for any function with 2+ params.
- **golangci-lint** chain: `errcheck`, `errname`, `govet`, `prealloc`,
  `predeclared`, `revive`, `staticcheck`. `just ready` runs the full suite.
- **Tests, not test plans** — table-driven, `foo.go ↔ foo_test.go`.
- **No inline hex colors** — palette constants in `internal/cli/theme.go`.

## Color palette (Max Headroom)

```
#c678dd  magenta   accent, banner, titles
#00d4ff  cyan      info hints, dates
#ffb86c  orange    tags
#50fa7b  green     success, done tasks
#e5c07b  yellow    (reserved)
#ff6ec7  pink      errors, overdue tasks
#6272a4  lavender  muted states
#c0caf5  fg        default text
#3b4261  drained   separators, dim text
```

No inline hex colors — palette constants live in `internal/cli/theme.go`.

## Building

```bash
go build -o jot .
go run . --help
```

## Reference docs (read on demand)

- [docs/configuration.md](docs/configuration.md) — config file, env vars, flags, SSH keys, data layout
- [docs/development.md](docs/development.md) — testing conventions, adding commands, error handling, note format
- [docs/contributing.md](docs/contributing.md) — commit style, lint chain, PR checklist
