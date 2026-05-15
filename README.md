# jot

Terminal-first markdown notes + todos with linked tasks.

Plain `.md` files with YAML frontmatter, opened in `$EDITOR` (nvim by
default). Write `[[slug]]` wiki links to connect notes, embed `@task`
markers with `#tags` to create linked todos, and let jot track
everything in git — every save is a commit. Sensitive notes encrypt at
rest via [kvlt] (age + SSH keys). An MCP server exposes the full surface
to LLM agents. No database: files are the source of truth.

[![Go Report Card][badge-go-report]][go-report]
[![License: MIT][badge-license]][MIT]
[![Build][badge-build]][build]
[![Conventional Commits][badge-commits]][commits]
[![Built with just][badge-just]][just]
[![Commit Activity][badge-activity]][activity]
[![hovnokod][badge-hovnokod]][hovnokod]

## Install

```bash
curl -fsSL https://github.com/retr0h/jot/raw/main/install.sh | bash
```

### Build from source

```bash
git clone https://github.com/retr0h/jot.git
cd jot
go build -o jot .
```

## Quick Start

```bash
# Initialize config and notes directory
jot init

# Create a new note (opens $EDITOR)
jot note new --title "meeting notes"

# List all notes
jot note list

# Full-text search
jot note search --query "standup"

# List open tasks
jot task list

# Tasks due this week
jot task due --period week

# Browse the git history of your notes
jot git log
```

## Note Format

```markdown
---
title: Team standup 2026-05-14
tags: [work, meetings]
created: 2026-05-14
---

# Team standup 2026-05-14

Discussed the [[deployment-runbook]] and next steps.

@task(follow up on infra ticket | due:friday) #work #ops

@task review PR before EOD #work

- [x] Send meeting summary
- [ ] Update [[deployment-runbook]]
```

Frontmatter keys:

| Key       | Type       | Description                          |
| --------- | ---------- | ------------------------------------ |
| `title`   | string     | Note title                           |
| `tags`    | string[]   | Searchable tags                      |
| `created` | YYYY-MM-DD | Creation date (set automatically)    |

`[[slug]]` links connect notes by filename slug. `@task` markers create
linked todos; the resolved form `@task(desc | due:X)` is written back
by jot after the interactive prompt. Tags follow the marker outside the
parentheses: `#tag`.

## nvim Setup

```lua
vim.pack.add('epwalsh/obsidian.nvim')
vim.pack.add('OXY2DEV/markview.nvim')
vim.pack.add('retr0h/jot.git')
```

jot ships an nvim plugin at `lua/jot/init.lua` that auto-configures
[obsidian.nvim] and [markview.nvim] when `$JOT_NOTES_DIR` is set in
your environment. No manual workspace setup required — set the env var
and the plugin wires everything.

### nvim Keybindings

| Key           | Action                              |
| ------------- | ----------------------------------- |
| `<leader>jf`  | Follow `[[link]]` under cursor      |
| `<leader>jj`  | Jump to note (quick switch)         |
| `<leader>js`  | Search notes (full-text)            |
| `<leader>jt`  | Browse tags                         |
| `<leader>jb`  | Show backlinks for current note     |
| `<leader>jl`  | List links in current note          |
| `<leader>jn`  | Create new note                     |
| `<leader>jx`  | Toggle checkbox done/undone         |
| `[[`          | Autocomplete `[[slug]]` links       |
| `#`           | Autocomplete `#tag` from frontmatter |

## CLI Reference

```
jot
├── init                                       Initialize config, notes dir, git repo
├── note
│   ├── new --title <title> [--secure]         Create a note and open in $EDITOR
│   ├── edit --slug <slug>                     Re-open an existing note
│   ├── mv --slug <slug> --title <new-title>   Rename (updates slug, links, git)
│   ├── list [--tag <tag>]                     List notes
│   └── search --query <query>                 Full-text search
├── task
│   ├── list [--status X] [--tag X]            List tasks (open by default)
│   ├── done --slug <slug> --desc <desc>       Mark a task complete
│   └── due [--period today|week|month]        Tasks due (default: week)
├── tag
│   └── list                                   List all tags
├── git
│   ├── log [--slug <slug>]                    Commit history
│   ├── diff [--slug <slug>]                   Uncommitted changes
│   └── show --commit <hash> [--note <slug>]   Specific commit diff
└── mcp
    └── start                                  MCP server over stdio
```

## MCP Server

jot ships an MCP server that any MCP-aware agent (Claude Code, Cursor,
Continue) can spawn per session. Add it to `.mcp.json` in your project
root for auto-discovery:

```json
{
  "mcpServers": {
    "jot": {
      "command": "jot",
      "args": ["mcp", "start"]
    }
  }
}
```

Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "jot": {
      "command": "jot",
      "args": ["mcp", "start"]
    }
  }
}
```

Available tools:

| Tool              | Description                                      |
| ----------------- | ------------------------------------------------ |
| `list_notes`      | List all notes, optionally filtered by tag       |
| `get_note`        | Read the full markdown content of a note by slug |
| `create_note`     | Create a new markdown note and parse `@task`s    |
| `edit_note`       | Overwrite a note's content                       |
| `delete_note`     | Delete a note file                               |
| `search_notes`    | Full-text search across note titles and bodies   |
| `list_tasks`      | List tasks filtered by status and/or tag         |
| `complete_task`   | Mark a task as done                              |

## Configuration

`~/.config/jot/jot.yaml` (created by `jot init`):

```yaml
# jot configuration — uncomment and adjust as needed.
# All values can also be set via JOT_<KEY> environment variables.

# editor: ""          # preferred editor (falls back to $EDITOR / $VISUAL)
# notes_dir: ""       # override the notes directory (default: <config>/notes)

# git:
#   enabled: false      # track notes directory in git
#   auto_commit: false  # commit automatically on every write
```

Environment variable overrides use the `JOT_` prefix with dots replaced
by underscores: `JOT_NOTES_DIR`, `JOT_EDITOR`, `JOT_GIT_AUTO_COMMIT`.

## Documentation

- [Development][Development] — architecture, building, data layout, color palette
- [Contributing][Contributing] — commit style, lint chain, testing standards

## License

[MIT]

[badge-go-report]: https://goreportcard.com/badge/github.com/retr0h/jot
[badge-license]: https://img.shields.io/badge/License-MIT-blue.svg
[badge-build]: https://github.com/retr0h/jot/actions/workflows/go.yml/badge.svg
[badge-commits]: https://img.shields.io/badge/Conventional%20Commits-1.0.0-yellow.svg
[badge-just]: https://img.shields.io/badge/built%20with-just-blue
[badge-activity]: https://img.shields.io/github/commit-activity/m/retr0h/jot
[badge-hovnokod]: https://img.shields.io/badge/hovnokod-true-red

[go-report]: https://goreportcard.com/report/github.com/retr0h/jot
[build]: https://github.com/retr0h/jot/actions/workflows/go.yml
[commits]: https://conventionalcommits.org
[just]: https://just.systems
[activity]: https://github.com/retr0h/jot/commits/main
[hovnokod]: https://github.com/retr0h/jot

[kvlt]: https://github.com/retr0h/kvlt
[obsidian.nvim]: https://github.com/epwalsh/obsidian.nvim
[markview.nvim]: https://github.com/OXY2DEV/markview.nvim
[MIT]: LICENSE
[Development]: docs/development.md
[Contributing]: docs/contributing.md
