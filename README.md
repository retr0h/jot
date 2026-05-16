[![go report card](https://goreportcard.com/badge/github.com/retr0h/jot?style=for-the-badge)](https://goreportcard.com/report/github.com/retr0h/jot)
[![license](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=for-the-badge)](LICENSE)
[![build](https://img.shields.io/github/actions/workflow/status/retr0h/jot/go.yml?style=for-the-badge)](https://github.com/retr0h/jot/actions/workflows/go.yml)
[![conventional commits](https://img.shields.io/badge/Conventional%20Commits-1.0.0-yellow.svg?style=for-the-badge)](https://conventionalcommits.org)
[![built with just](https://img.shields.io/badge/Built_with-Just-black?style=for-the-badge&logo=just&logoColor=white)](https://just.systems)
![github commit activity](https://img.shields.io/github/commit-activity/m/retr0h/jot?style=for-the-badge)
[![hovnokod](https://raw.githubusercontent.com/tekk/hovnokod-badge/main/assets/badges/hovnokod-for-the-badge.svg)](https://github.com/tekk/hovnokod-badge)

# jot

Terminal-first markdown notes + todos with linked tasks.

Plain `.md` files with YAML frontmatter, opened in `$EDITOR`. Write
`[[slug]]` wiki links to connect notes, embed `@task` markers with
`#tags` to create linked todos, and let jot track everything in git —
every save is a commit. Sensitive notes encrypt at rest via [kvlt][]
(age + SSH keys). An MCP server exposes the full surface to LLM agents.
No database: files are the source of truth.

## 📦 Install

```bash
curl -fsSL https://github.com/retr0h/jot/raw/main/install.sh | bash
```

### 🔨 Build from source

```bash
git clone https://github.com/retr0h/jot.git
cd jot
go build -o jot .
```

## 🚀 Quick Start

```bash
jot init
jot note new --title "Sprint Planning"
jot note new --title "network/Switch Config"
jot note new --title "secrets/API Keys" --secure
jot note list
jot note search --query "kubernetes"
jot task list
jot task due
jot git log
```

## ✨ Features

| Feature              | Description                                              |
| -------------------- | -------------------------------------------------------- |
| Markdown notes       | `.md` files with YAML frontmatter, edited in `$EDITOR`  |
| `[[wiki links]]`     | Cross-reference notes by slug; obsidian.nvim navigates   |
| `@task` todos        | Due dates + `#tags`; tracked by jot, visible in nvim     |
| Git-backed           | Every edit auto-commits; `jot git log/diff/show`         |
| Secure vaults        | `--secure` encrypts notes via [kvlt][] (age + SSH keys)  |
| Full-text search     | Concurrent goroutine file scan — no index, no database   |
| MCP server           | 8 tools over stdio for Claude / Cursor / any MCP client  |
| Subdirectory nesting | `jot note new --title "ops/runbook"` for organization    |
| nvim plugin          | Ships `lua/jot/` — auto-configures obsidian.nvim         |
| Max Headroom palette | Magenta `#c678dd` accent across CLI, installer, nvim     |

## 📝 Note Format

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

`[[slug]]` links connect notes. `@task` markers create linked todos.
`#tags` are searchable via CLI (`jot tag list`) and nvim (`<leader>jt`).
Frontmatter `tags:` arrays are also recognized.

## 🖥️ nvim Setup

```lua
vim.pack.add({"https://github.com/epwalsh/obsidian.nvim.git"})
vim.pack.add({"https://github.com/OXY2DEV/markview.nvim.git"})
vim.pack.add({"https://github.com/retr0h/jot.git"})
```

jot ships an nvim plugin that auto-configures [obsidian.nvim][] and
[markview.nvim][] when `$JOT_NOTES_DIR` is set (jot sets this
automatically when spawning the editor). Zero manual config.

### Keybindings

| Key           | Action                              |
| ------------- | ----------------------------------- |
| `<leader>jf`  | Follow `[[link]]` under cursor      |
| `<leader>jj`  | Jump to note (fuzzy switch)         |
| `<leader>js`  | Search notes (ripgrep)              |
| `<leader>jt`  | Browse `#tags`                      |
| `<leader>jb`  | Backlinks to current note           |
| `<leader>jl`  | Links in current note               |
| `<leader>jn`  | New note                            |
| `<leader>jx`  | Toggle `[ ]` / `[x]` done          |
| `[[`          | Autocomplete note slugs             |
| `#`           | Autocomplete tags                   |

### Snippets

Type `@task` + Tab to expand with tab-stop placeholders:

```
@task(description | due:friday) #tag
```

## 🤖 MCP Server

Auto-discovered by Claude Code via [`.mcp.json`](.mcp.json):

```json
{
  "mcpServers": {
    "jot": { "command": "jot", "args": ["mcp", "start"] }
  }
}
```

| Tool           | Description                                        |
| -------------- | -------------------------------------------------- |
| `list_notes`   | List notes, optionally filtered by tag             |
| `get_note`     | Read full markdown by slug                         |
| `create_note`  | Create a note with frontmatter                     |
| `delete_note`  | Remove a note file                                 |
| `search_notes` | Full-text search                                   |
| `list_tasks`   | Tasks filtered by status/tag                       |
| `tasks_due`    | Tasks due in a date range                          |
| `list_tags`    | All tags across notes                              |

## ⚙️ Configuration

`~/.config/jot/jot.yaml`:

```yaml
# editor: ""          # override $EDITOR
# notes_dir: ""       # override notes directory
```

Git is always on — every edit auto-commits. Override the config dir with
`--config` or `JOT_CONFIG_DIR`.

## 📖 Documentation

- [Development][] — architecture, testing conventions, adding commands
- [Contributing][] — commit style, lint chain, PR checklist

## 📄 License

[MIT][]

[kvlt]: https://github.com/retr0h/kvlt
[obsidian.nvim]: https://github.com/epwalsh/obsidian.nvim
[markview.nvim]: https://github.com/OXY2DEV/markview.nvim
[MIT]: LICENSE
[Development]: docs/development.md
[Contributing]: docs/contributing.md
