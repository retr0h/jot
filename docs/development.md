# Development Guide

## Prerequisites

Install tools using [mise](https://mise.jdx.dev/):

```bash
mise install
```

- **[Go](https://go.dev)** >= 1.25
- **[just](https://just.systems)** — task runner

## Quick Start

```bash
git clone https://github.com/retr0h/jot.git
cd jot
go test ./...
```

## Layout

```
jot (binary)
├── cmd/                cobra CLI tree — one file per subcommand
├── internal/
│   ├── cli/            theme (maxheadroom palette), banner, output helpers
│   ├── config/         Config struct, viper mapstructure bindings
│   ├── gitops/         git integration — commit, log, diff, show
│   ├── jot/            core domain — parser (@task), editor, slug, date, kvlt
│   ├── mcp/            MCP server (stdio, JSON-RPC via go-sdk)
│   └── version/        build-time identity (ldflags injected)
├── lua/jot/            nvim plugin (obsidian.nvim integration)
├── plugin/             nvim auto-load entry point
```

## Common Tasks

```bash
go test ./...              # run tests
go vet ./...               # vet
gofmt -l .                 # find unformatted files
```

Or via just:

```bash
just deps
just test
just ready          # fmt + vet + lint
```

## Key Invariants

**Files are the source of truth.** Every note is a plain `.md` file.
There is no database — the filesystem is the store. Slugs are derived
from the title and a timestamp to avoid collisions.

**Git backs everything.** Every write goes through `internal/gitops`,
which stages the file and creates a commit. `jot git log` / `jot git
diff` / `jot git show` surface the history to the user. Nothing is lost.

**`@task` round-trips.** The parser handles both forms:

```
@task fix the broken deploy script
@task(fix the broken deploy script | due:friday) #ops
```

After the interactive prompt (or MCP call) jot rewrites bare markers to
the resolved `@task(...)` form in place. Editing the file in nvim leaves
both forms valid; the parser accepts either on the next read.

**`[[slug]]` links.** Wiki-style links reference other notes by slug.
obsidian.nvim resolves them and autocompletes on `[[`. jot itself does
not validate links at write time — broken links surface in the nvim UI.

**Tags in frontmatter.** Tags are YAML `tags:` arrays in frontmatter.
`@task` markers carry `#tag` annotations outside the parentheses. Both
surfaces are queryable via `jot tag list` and the MCP `list_notes` tool.

**Interfaces where consumed.** The MCP server defines its own interfaces
in `internal/mcp/` rather than depending on a concrete type from
`internal/jot/`. The cobra commands follow the same pattern — each
command file declares a narrow interface for the operations it needs.

## Testing Conventions

**Every public function and method MUST have a table-driven test.** One
table per function, with rows covering both the happy path and every
failure mode the function can produce.

> **Anti-pattern (do not do this):** writing a separate one-off test
> function for a failure scenario. If you find yourself drafting a
> `TestFoo_ReturnsErrorOnEmpty`, stop — instead add a row to the
> existing `TestFoo` table. Each public function gets exactly **one**
> `Test*` function in the codebase.

### File naming (non-negotiable)

**One test file per production file.** `notes.go` is tested by
`notes_test.go` — never `helpers_notes_test.go` or similar. If tests
outgrow one file, split the production file first.

Two exceptions: shared fixtures in `helpers_test.go` / `fakes_test.go`,
and `main_test.go` for `TestMain`.

### Table shape

For pure functions:

```go
func TestParseDate(t *testing.T) {
    cases := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"explicit date", "2026-05-20", "2026-05-20", false},
        {"today", "today", "2026-05-14", false},
        {"invalid", "nope", "", true},
    }
    for _, c := range cases {
        t.Run(c.name, func(t *testing.T) { ... })
    }
}
```

### Required failure rows

Every function's table MUST include rows covering:

| Scenario        | What to test                                        |
| --------------- | --------------------------------------------------- |
| Happy path      | Expected input produces expected output              |
| Empty input     | Empty string / nil / zero value handled gracefully   |
| Invalid input   | Malformed data returns an error, doesn't panic       |
| Edge cases      | Boundary values, duplicates, unicode, path traversal |

### Test naming

- Pure functions: `TestFunctionName`.
- Methods on a type: `TestType_Method`.
- Helper / unexported: `Test_helperName`.

## Adding a Command

1. Create `cmd/<parent>_<name>.go` (e.g. `note_edit.go`).
2. Define a `*cobra.Command` with `Args: cobra.NoArgs` — use flags for
   all inputs, never positional args. Mark required flags with
   `cobra.MarkFlagRequired`.
3. Register in `init()` via the parent subcommand.
4. Declare a narrow interface for the operations the command needs.
5. Pair with `cmd/<parent>_<name>_test.go` with table-driven tests.
6. Run `just ready`.

## Error Handling

- Every exported function returns `(T, error)` or `error`.
- Wrap errors with context: `fmt.Errorf("open store: %w", err)`.
- Exported functions never panic; they always return errors.
- Use `cli.Print`/`cli.Printf` for stdout output (never unchecked
  `fmt.Fprint*`).

## Data Directory Layout

```
~/.config/jot/
├── jot.yaml           # user config (created by jot init)
└── notes/
    ├── *.md           # plain notes
    ├── subdir/*.md    # nested notes
    └── .git/          # git repo tracking the notes directory
```

Override via `--config` flag or `JOT_CONFIG_DIR` env var. Notes
directory overridden independently with `JOT_NOTES_DIR`.

## Note Format

```markdown
---
title: Deployment runbook
tags: [ops, infra]
created: 2026-02-01
---

# Deployment runbook

Steps before promoting to production. See also [[pre-deploy-checklist]].

@task(run smoke tests | due:2026-05-16) #ops #ci
@task update rollback docs #ops

- [x] Review diff with team
- [ ] Tag release
```

## @task Syntax

| Form      | Example                                                        |
| --------- | -------------------------------------------------------------- |
| Bare      | `@task description #tag`                                       |
| Resolved  | `@task(description \| due:YYYY-MM-DD) #tag`                    |
| With done | `@task(description \| due:2026-05-14 \| done:2026-05-15) #tag` |

Natural language dates accepted by `due:`: `today`, `tomorrow`,
`monday`...`sunday`, `next week`, `YYYY-MM-DD`.

## Color Palette (Max Headroom)

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

All palette values are defined as named constants in
`internal/cli/theme.go`. Never pass raw hex strings to lipgloss in
command or handler code — reference the theme roles (`Accent`, `OK`,
`Err`, `Info`, `Mute`) instead.

## MCP Server

The server is implemented with `github.com/modelcontextprotocol/go-sdk`.
Tools are registered in `internal/mcp/tools.go` via `mcpsdk.AddTool`.
Each tool has a dedicated argument struct with `json` and `jsonschema`
tags — the SDK derives the JSON Schema from those tags automatically.

The server is spawned per agent session over stdio. When the agent
disconnects the process exits and any open resources are closed via
deferred cleanup.

## Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

- **Subject line**: max 50 characters, imperative mood, capitalized, no
  period
- **Body**: wrap at 72 characters, separated from subject by a blank
  line
- **Format**: `type(scope): description`
- **Types**: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`,
  `chore`
- **Scopes**: `cli`, `jot`, `gitops`, `mcp`, `docs`, `nvim`
