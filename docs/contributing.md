# Contributing

Thanks for wanting to make jot better. This guide covers the conventions
you need to follow for a smooth PR review.

## Prerequisites

- Go 1.25+
- [just] command runner (`brew install just` / `cargo install just`)
- golangci-lint (`brew install golangci-lint`)

## Workflow

1. Fork the repo and create a branch from `main`.
2. Make your changes with tests.
3. Run `just ready` before pushing — it formats, lints, and runs the full
   test suite.
4. Open a pull request against `main`.

## Commit Style

jot follows [Conventional Commits]. Every commit message must have a
type prefix:

```
feat: add tag autocomplete to nvim plugin
fix: handle missing notes_dir on first run
docs: update @task syntax in contributing guide
refactor: extract gitops commit helper
test: add table-driven cases for slug collision
chore: bump go-sdk to v1.7.0
```

Breaking changes use a `!` after the type or a `BREAKING CHANGE:` footer:

```
feat!: change @task label syntax from label: to #tag
```

Scope is optional but appreciated when the change is narrow:

```
fix(parser): handle @task with no trailing newline
```

## Code Standards

### Multi-line function signatures

Any function with two or more parameters must use a multi-line signature:

```go
// good
func CreateNote(
    slug string,
    title string,
    secure bool,
) (*Note, error) {

// bad
func CreateNote(slug string, title string, secure bool) (*Note, error) {
```

### Error handling

Wrap errors with context at every boundary:

```go
if err := store.IndexNote(id, title, content); err != nil {
    return fmt.Errorf("index note %q: %w", slug, err)
}
```

Sentinel errors use `errors.New` at package level; error types implement
the `error` interface and are named with an `Err` prefix or `Error`
suffix per `errname` lint rules.

### No inline hex colors

All palette values live in `internal/cli/theme.go` as named constants.
Never pass a raw hex string to lipgloss in command or handler code.

### Interfaces where consumed

Define interfaces in the package that consumes them, not the package
that provides the implementation. The MCP server's `storer` interface
lives in `internal/mcp/`, not in `internal/jot/`.

## Linting

The full lint chain runs via:

```bash
just ready
```

Individual steps:

```bash
just go::fmt    # gofmt + goimports
just go::vet    # go vet
just go::lint   # golangci-lint run
```

Enabled linters: `errcheck`, `errname`, `govet`, `prealloc`,
`predeclared`, `revive`, `staticcheck`.

## Testing

Every `.go` file must have a paired `_test.go` file. Tests are
table-driven using subtests:

```go
func TestParseTasks(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    []RawTask
    }{
        {
            name:  "bare form",
            input: "@task fix the thing",
            want:  []RawTask{{Description: "fix the thing", Line: 1}},
        },
        {
            name:  "resolved form with due",
            input: "@task(fix the thing | due:friday)",
            want:  []RawTask{{Description: "fix the thing", DueDate: "friday", Line: 1, Resolved: true}},
        },
    }

    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            got := ParseTasks(tc.input)
            // assert ...
        })
    }
}
```

Run tests with the race detector:

```bash
just go::test
# or directly:
go test -race ./...
```

## Pull Request Checklist

- [ ] `just ready` passes with no errors
- [ ] New or changed behavior has table-driven tests
- [ ] No inline hex color values
- [ ] Multi-line signatures for functions with 2+ params
- [ ] Errors wrapped with `fmt.Errorf("context: %w", err)`
- [ ] Commit messages follow Conventional Commits

[just]: https://just.systems
[Conventional Commits]: https://conventionalcommits.org
