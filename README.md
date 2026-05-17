[![go report card](https://goreportcard.com/badge/github.com/retr0h/jot?style=for-the-badge)](https://goreportcard.com/report/github.com/retr0h/jot)
[![license](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=for-the-badge)](LICENSE)
[![build](https://img.shields.io/github/actions/workflow/status/retr0h/jot/go.yml?style=for-the-badge)](https://github.com/retr0h/jot/actions/workflows/go.yml)
[![codecov](https://img.shields.io/codecov/c/github/retr0h/jot?style=for-the-badge)](https://codecov.io/gh/retr0h/jot)
[![conventional commits](https://img.shields.io/badge/Conventional%20Commits-1.0.0-yellow.svg?style=for-the-badge)](https://conventionalcommits.org)
[![built with just](https://img.shields.io/badge/Built_with-Just-black?style=for-the-badge&logo=just&logoColor=white)](https://just.systems)
![github commit activity](https://img.shields.io/github/commit-activity/m/retr0h/jot?style=for-the-badge)
[![hovnokod](https://raw.githubusercontent.com/tekk/hovnokod-badge/main/assets/badges/hovnokod-for-the-badge.svg)](https://github.com/tekk/hovnokod-badge)

# jot

Terminal-first markdown notes + todos with linked tasks.

Plain `.md` files with YAML frontmatter, edited in nvim with
obsidian.nvim. Write `[[slug]]` wiki links to connect notes, embed
`@task` markers with `#tags` to create linked todos, and let jot track
everything in git — every save is a commit. Sensitive notes encrypt at
rest via [kvlt][] (age + SSH keys). An MCP server exposes the full
surface to LLM agents. No database: files are the source of truth.

## 📋 Prerequisites

- [neovim](https://neovim.io) — jot is opinionated; all editing happens in nvim
- [obsidian.nvim](https://github.com/epwalsh/obsidian.nvim) — `[[wiki-link]]` navigation
- [markview.nvim](https://github.com/OXY2DEV/markview.nvim) — markdown rendering

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
| Markdown notes       | `.md` files with YAML frontmatter, edited in nvim       |
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
| `<leader>j?`  | Show keybindings (`:JotHelp`)       |
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

Type `@task` then Enter to expand with tab-stop placeholders:

```
@task(description | due:today) #tag
```

Tab jumps between placeholders. Date completions (next 7 days + day
names) appear automatically while typing the due field.

## 🤖 MCP Server

Auto-discovered by Claude Code via [`.mcp.json`](.mcp.json):

```json
{
  "mcpServers": {
    "jot": { "command": "jot", "args": ["mcp", "start"] }
  }
}
```

| Tool           | Description                                                      |
| -------------- | ---------------------------------------------------------------- |
| `list_notes`   | List notes, optionally filtered by tag                           |
| `get_note`     | Read full markdown by slug (transparently decrypts secure notes) |
| `create_note`  | Create a note with frontmatter (supports `secure: true`)         |
| `delete_note`  | Remove a note file                                               |
| `search_notes` | Full-text search                                                 |
| `rename_note`  | Rename a note and rewrite all `[[wikilinks]]`                    |
| `encrypt_note` | Encrypt an existing note via kvlt                                |
| `decrypt_note` | Decrypt a secure note back to plaintext                          |
| `list_tasks`   | Tasks filtered by status/tag                                     |
| `task_done`    | Mark a task as complete                                          |
| `tasks_due`    | Tasks due in a date range                                        |
| `list_tags`    | All tags across notes                                            |

Secure notes over MCP require passphrase-free SSH keys configured in
`ssh_keys` — there is no TTY for interactive prompts.

## ⚙️ Configuration

`~/.config/jot/jot.yaml`:

```yaml
# notes_dir: ""       # override notes directory
# ssh_keys: []        # SSH key paths for kvlt (empty = auto-discover ~/.ssh/)
```

Git is always on — every edit auto-commits. See [Configuration][] for
the full reference (env vars, flags, precedence, SSH keys).

## 📖 Documentation

- [Configuration][] — env vars, flags, SSH keys, data layout
- [Development][] — architecture, testing conventions, adding commands
- [Contributing][] — commit style, lint chain, PR checklist

## 📄 License

[MIT][]

[kvlt]: https://github.com/retr0h/kvlt
[obsidian.nvim]: https://github.com/epwalsh/obsidian.nvim
[markview.nvim]: https://github.com/OXY2DEV/markview.nvim
[MIT]: LICENSE
[Configuration]: docs/configuration.md
[Development]: docs/development.md
[Contributing]: docs/contributing.md
