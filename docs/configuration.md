# Configuration

## Config File

`~/.config/jot/jot.yaml` (created by `jot init`):

```yaml
# notes_dir: ""       # override notes directory
# ssh_keys: []        # SSH key paths for kvlt (empty = auto-discover ~/.ssh/)
```

All keys can also be set via environment variables or CLI flags.

## Environment Variables

| Variable         | Description                   | Default          |
| ---------------- | ----------------------------- | ---------------- |
| `JOT_CONFIG_DIR` | Config directory              | `~/.config/jot`  |
| `JOT_NOTES_DIR`  | Notes directory               | `<config>/notes` |
| `JOT_SSH_KEYS`   | Comma-separated SSH key paths | auto-discover    |
| `JOT_DEBUG`      | Enable debug logging          | `false`          |

## CLI Flags

Global flags available on all commands:

| Flag          | Short | Description                        |
| ------------- | ----- | ---------------------------------- |
| `--config`    |       | Config directory                   |
| `--notes-dir` |       | Notes directory                    |
| `--ssh-key`   |       | SSH key path for kvlt (repeatable) |
| `--debug`     | `-d`  | Enable debug logging               |
| `--json`      | `-j`  | Emit logs as JSON                  |

## Precedence

Flag > environment variable > config file > default.

## SSH Keys

jot uses [kvlt](https://github.com/retr0h/kvlt) (age + SSH keys) for note
encryption. By default, jot auto-discovers keys from `~/.ssh/` (ed25519, rsa).
To pin specific keys:

```yaml
ssh_keys:
  - ~/.ssh/id_ed25519
  - ~/.ssh/id_work
```

Or via flags:

```bash
jot note decrypt -s secret-stuff --ssh-key ~/.ssh/id_ed25519 --ssh-key ~/.ssh/id_work
```

Passphrase-protected keys prompt via `/dev/tty`.

## Data Directory Layout

```
~/.config/jot/
├── jot.yaml           # user config
└── notes/
    ├── .git/          # git repo tracking the notes directory
    ├── .kvlt/jot/     # age-encrypted note bodies (tracked by git)
    ├── scratch.md     # persistent scratch pad (created by init)
    ├── *.md           # plain notes
    └── subdir/*.md    # nested notes
```

## Editor

jot hardcodes nvim. Notes are designed around obsidian.nvim for wiki-link
navigation, tag completion, and snippet expansion. There is no editor config
option — nvim must be available on `$PATH`.
