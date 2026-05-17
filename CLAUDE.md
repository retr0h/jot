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

## Architecture in one screen

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

Command tree (long form / short alias):

```
jot
├── init
├── note (n)
│   ├── new (n)    [--title/-t <title>] [--secure/-x]  (omit title for scratch)
│   ├── edit (e)   --slug/-s <slug>
│   ├── cat (c)    --slug/-s <slug>
│   ├── mv (m)     --slug/-s <slug> --title/-t <new-title>
│   ├── list (l)   [--tag/-T X]
│   ├── search (s) --query/-q <query>
│   ├── encrypt (enc) --slug/-s <slug>
│   └── decrypt (dec) --slug/-s <slug>
├── task (t)
│   ├── list (l)   [--status/-S X] [--tag/-T X]
│   ├── done (d)   --slug/-s <slug> --desc/-D <desc>
│   └── due (du)   [--period/-p today|week|month]
├── tag (tg)
│   └── list (l)
├── git (g)
│   ├── log (l)    [--slug/-s X]
│   ├── diff (d)   [--slug/-s X]
│   └── show (s)   --commit/-c <hash> [--note/-n X]
└── mcp start
```

Key invariants:

- **Files are source of truth.** No database — `.md` files are canonical.
- **Git backs everything.** Every write goes through `internal/gitops`
  which stages and commits. `jot git log/diff/show` surface the history.
- **@task round-trips.** Parser reads both `@task desc #tag` and
  `@task(desc | due:X) #tag` forms — edits in nvim are always valid.
- **[[slug]] links.** Wiki-style links connect notes by slug; obsidian.nvim
  resolves them. jot does not validate links at write time.
- **Tags in frontmatter.** Tags are YAML `tags:` arrays. `@task` markers
  carry `#tag` annotations outside the parentheses. Both are queryable.
- **Interfaces where consumed.** Define interfaces in the consuming package
  (MCP `storer`, cobra command seams) not in the providing package.

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
#c678dd  magenta   accent, banner
#00d4ff  cyan      info hints
#ffb86c  orange    (reserved)
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
Notes directory defaults to `<config>/notes`; override via `--notes-dir` flag
or `JOT_NOTES_DIR` env.

## Configuration (jot.yaml)

```yaml
# notes_dir: ""       # override notes directory
# ssh_keys: []        # SSH key paths for kvlt (empty = auto-discover ~/.ssh/)
```

Global flags: `--config`, `--notes-dir`, `--ssh-key` (repeatable), `--debug`, `--json`.

## Scratch

`jot note new` with no `--title` opens `scratch.md` — a persistent scratch
pad created by `jot init`. Always one command away for quick dumps.
