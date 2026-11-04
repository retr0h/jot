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

Plain `.md` files with YAML frontmatter, edited in nvim with obsidian.nvim.
Write `[[slug]]` wiki links to connect notes, embed checkbox tasks with `#tags`
to create linked todos, and let jot track everything in git — every save is a
commit. Sensitive notes encrypt at rest via [kvlt] (age + SSH keys). An MCP
server exposes the full surface to LLM agents. No database: files are the source
of truth.

## 📦 Install

> Requires [neovim](https://neovim.io) with plugins — see
> [nvim Setup](#%EF%B8%8F-nvim-setup) below. Also requires
> [ripgrep](https://github.com/BurntSushi/ripgrep) and
> [fzf](https://github.com/junegunn/fzf) for search.

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
jot note new --title "memory/Projects" --undated
cat meeting.md | jot note new --title "Meeting Notes"
echo "secret stuff" | jot note new --title "Credentials" --secure
jot note list
jot note search --query "kubernetes"
jot note append --slug todo --text "Sam prefers mornings"
jot task list
jot task list --done-since 2026-11-01
jot task add --slug todo --desc "call Sam" --due friday --tag work
jot task done --slug todo --line 9
jot task due
jot git log
```

## ✨ Features

| Feature              | Description                                             |
| -------------------- | ------------------------------------------------------- |
| Markdown notes       | `.md` files with YAML frontmatter, edited in nvim       |
| `[[wiki links]]`     | Cross-reference notes by slug; obsidian.nvim navigates  |
| Checkbox tasks       | `- [ ] desc \| due:date #tag` with due dates + tags     |
| Git-backed           | Every edit auto-commits; `jot git log/diff/show`        |
| Secure vaults        | `--secure` encrypts notes via [kvlt] (age + SSH keys)   |
| Full-text search     | Concurrent goroutine file scan — no index, no database  |
| MCP server           | 16 tools over stdio for Claude / Codex / any MCP client |
| Agent memory         | One git-backed memory shared by all your agents         |
| Subdirectory nesting | `jot note new --title "ops/runbook"` for organization   |
| nvim plugin          | Ships `lua/jot/` — auto-configures obsidian.nvim        |
| Max Headroom palette | Magenta `#c678dd` accent across CLI, installer, nvim    |

## 📝 Note Format

```markdown
---
title: Team standup 2026-05-14
tags: [work, meetings]
created: 2026-05-14
---

# Team standup 2026-05-14

Discussed the [[deployment-runbook]] and next steps.

- [ ] follow up on infra ticket | due:friday #work #ops
- [ ] review PR before EOD #work
- [x] Send meeting summary | done:2026-05-14
- [ ] Update [[deployment-runbook]]
```

`[[slug]]` links connect notes. Checkbox tasks create linked todos. `#tags` are
searchable via CLI (`jot tag list`) and nvim (`<leader>jt`). Frontmatter `tags:`
arrays are also recognized.

## 🖥️ nvim Setup

```lua
vim.pack.add({"https://github.com/epwalsh/obsidian.nvim.git"})
vim.pack.add({"https://github.com/OXY2DEV/markview.nvim.git"})
vim.pack.add({"https://github.com/retr0h/jot.git"})
```

jot ships an nvim plugin that auto-configures [obsidian.nvim] and
[markview.nvim] when `$JOT_NOTES_DIR` is set (jot sets this automatically when
spawning the editor). Zero manual config.

### Keybindings

| Key          | Action                         |
| ------------ | ------------------------------ |
| `<leader>j?` | Show keybindings (`:JotHelp`)  |
| `<leader>jf` | Follow `[[link]]` under cursor |
| `<leader>jj` | Jump to note (fuzzy switch)    |
| `<leader>js` | Search notes (ripgrep)         |
| `<leader>jt` | Browse `#tags`                 |
| `<leader>jb` | Backlinks to current note      |
| `<leader>jl` | Links in current note          |
| `<leader>jn` | New note                       |
| `<leader>jx` | Toggle `[ ]` / `[x]` done      |
| `[[`         | Autocomplete note slugs        |
| `#`          | Autocomplete tags              |

### Snippets

Type `task` then Enter to expand with tab-stop placeholders:

```
- [ ] description | due:today #tag
```

Tab jumps between placeholders. Date completions (next 7 days + day names)
appear automatically while typing the due field.

## 🤖 MCP Server

Auto-discovered by Claude Code via [`.mcp.json`](.mcp.json):

```json
{
  "mcpServers": {
    "jot": { "command": "jot", "args": ["mcp", "start"] }
  }
}
```

| Tool           | Description                                                                 |
| -------------- | --------------------------------------------------------------------------- |
| `list_notes`   | Note summaries (title, description, tags, dates); `full` adds bodies        |
| `get_note`     | Read full markdown by slug (transparently decrypts secure notes)            |
| `create_note`  | Create a note, in a folder with `dir/Title`; `undated` drops the slug date  |
| `append_note`  | Add text to the end of an existing note                                     |
| `update_note`  | Replace a note's body, or the whole file when content has frontmatter       |
| `delete_note`  | Remove a note file                                                          |
| `search_notes` | Full-text search; summaries carry the first matching line                   |
| `rename_note`  | Rename a note in its folder, dated or not as before, and fix `[[links]]`    |
| `encrypt_note` | Encrypt an existing note via kvlt, keeping every frontmatter field          |
| `decrypt_note` | Decrypt a secure note back to plaintext                                     |
| `history`      | Commits newest first with the notes each changed, filtered by note and time |
| `list_tasks`   | Tasks filtered by status, tag and done date, with each task's file line     |
| `task_add`     | Append a task to a note, creating the note if needed                        |
| `task_done`    | Mark a task complete by line or description                                 |
| `tasks_due`    | Tasks due in a date range                                                   |
| `list_tags`    | All tags across notes                                                       |

Every write tool commits to the notes repo, like the CLI, and writers from every
jot process take the same lock, so agents sharing a notes dir do not lose each
other's edits. The lock does not cover edits made in your own editor. A note's
slug is its path under the notes dir without `.md`, such as `done/2026-11-03`.
Give notes a one-line `description:` in frontmatter so agents can recall them
from `list_notes` and `search_notes` without reading bodies. Secure notes over
MCP require passphrase-free SSH keys configured in `ssh_keys` — there is no TTY
for interactive prompts.

### 🧠 Agent memory

jot works as long-term memory shared by every agent on the machine. Claude Code,
Codex and any other MCP client read and write the same notes dir, every write is
a git commit, and `history` shows what changed and when. Register the server
once per agent:

```bash
claude mcp add jot --scope user -- jot mcp start
```

```toml
# ~/.codex/config.toml
[mcp_servers.jot]
command = "jot"
args = ["mcp", "start"]
```

Then tell the agents when to use it, in `~/.claude/CLAUDE.md` and
`~/.codex/AGENTS.md`:

```markdown
- jot (MCP server `jot`) is the memory shared across agents. Before a task,
  `search_notes` for its topic. Save lasting facts, decisions and preferences
  with `append_note` to the matching note under `memory/`, or `create_note`
  `memory/<Topic>` with `undated: true` and a one-line `description:` when
  none fits
- secrets go only in a secure note's body, never in a title, description or tag
```

To keep memory to one project instead, register the server with `--scope local`
from that project, put the Codex block in the project's `.codex/config.toml`,
and put the instructions in its `AGENTS.md`.

Secure note bodies stay encrypted, so search does not see them; titles,
descriptions and tags are plaintext. The write lock covers one machine. To share
memory across machines, push and pull the notes repo with git.

## ⚙️ Configuration

`~/.config/jot/jot.yaml`:

```yaml
# notes_dir: ""       # override notes directory
# ssh_keys: []        # SSH key paths for kvlt (empty = auto-discover ~/.ssh/)
```

Git is always on — every edit auto-commits. See [Configuration] for the full
reference (env vars, flags, precedence, SSH keys).

## 📖 Documentation

- [Configuration] — env vars, flags, SSH keys, data layout
- [Development] — architecture, testing conventions, adding commands
- [Contributing] — commit style, lint chain, PR checklist

## 📄 License

[MIT]

[configuration]: docs/configuration.md
[contributing]: docs/contributing.md
[development]: docs/development.md
[kvlt]: https://github.com/retr0h/kvlt
[markview.nvim]: https://github.com/OXY2DEV/markview.nvim
[mit]: LICENSE
[obsidian.nvim]: https://github.com/epwalsh/obsidian.nvim
