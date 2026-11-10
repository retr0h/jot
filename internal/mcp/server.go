// Copyright (c) 2026 John Dewey
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

// Package mcp is the Model Context Protocol server for jot. Exposes every
// note, task, and tag operation as an MCP tool an LLM agent can call.
package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/retr0h/jot/internal/gitops"
	"github.com/retr0h/jot/internal/jot"
)

// version is the MCP implementation version surfaced to agents on initialize.
const version = "0.1.0"

// ─── consumer-seam interface ─────────────────────────────────────────────────

// Store is the narrow consumer-seam interface this MCP server depends on.
// Concrete *jot.Service satisfies it structurally — the compiler verifies
// at the assignment site in New().
type Store interface {
	ListNotes(tagFilter string, dir string) ([]*jot.Note, error)
	SearchNotes(query string, dir string) ([]*jot.Note, error)
	FindNoteBySlug(slug string) (*jot.Note, error)
	CatNote(slug string) (string, error)
	CreateNote(title string, content string, secure bool) (string, string, error)
	DeleteNote(slug string) error
	EncryptNote(slug string) error
	DecryptNote(slug string) error
	RenameNote(oldSlug string, newTitle string) (string, error)
	AllTasks(status string, tagFilter string, dir string) ([]jot.Task, error)
	TasksDue(from time.Time, to time.Time) ([]jot.Task, error)
	MarkTaskDone(slug string, desc string, reason string, force bool) error
	MarkTaskDoneAt(slug string, line int, reason string, force bool) error
	MarkTaskDoneByID(id string, reason string, force bool) error
	AddTask(slug string, title string, task jot.RawTask) (jot.RawTask, error)
	AppendNote(slug string, text string) error
	UpdateNote(slug string, content string) error
	AllTags() ([]string, error)
}

// History reads the notes repository's commit log; *gitops.Repo satisfies
// it. Without one, the history tool reports that there is no repository.
type History interface {
	Log(filename string, limit int) ([]gitops.LogEntry, error)
}

// ─── Config and Server ───────────────────────────────────────────────────────

// Config bundles the runtime inputs for a jot MCP server. The Store commits
// its own writes, inside its write lock.
type Config struct {
	Store   Store
	History History
	Logger  *slog.Logger
}

// Server is the jot MCP server. Holds a Store (the narrow consumer
// surface) used by every tool handler, plus the underlying mcpsdk.Server.
// Constructed via New; the wire is driven by Run.
type Server struct {
	mcp     *mcpsdk.Server
	store   Store
	history History
	logger  *slog.Logger
}

// New creates an MCP server, registers all tools, and returns a
// ready-to-Run Server.
func New(cfg Config) (*Server, error) {
	if cfg.Store == nil {
		return nil, fmt.Errorf("mcp: Store required")
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}

	mcpSrv := mcpsdk.NewServer(
		&mcpsdk.Implementation{
			Name:    "jot",
			Version: version,
		},
		&mcpsdk.ServerOptions{
			Instructions: instructions,
		},
	)

	s := &Server{
		mcp:     mcpSrv,
		store:   cfg.Store,
		history: cfg.History,
		logger:  cfg.Logger.With(slog.String("subsystem", "mcp")),
	}
	s.registerTools()
	return s, nil
}

// Run wires the MCP server to stdin/stdout and blocks until the transport
// closes or ctx cancels.
func (s *Server) Run(ctx context.Context) error {
	s.logger.Info(
		"running",
		slog.String("transport", "stdio"),
	)
	return s.mcp.Run(ctx, &mcpsdk.StdioTransport{})
}

// Close is a no-op; retained for interface compatibility.
func (s *Server) Close() error {
	return nil
}

// instructions is the MCP server's self-description, surfaced to the agent
// on initialize.
const instructions = `jot — terminal notes and tasks backed by markdown files and git.

This server exposes every jot operation over MCP so an LLM agent can read,
create, and delete notes, tasks, and tags directly. Every write is committed
to the notes git repository, like the CLI's.

## Notes

Notes are markdown files stored on disk. Each note has a slug (its path
under the notes dir without .md, such as "work/sensor-ideas"; a leading "/"
is allowed, so "/work/sensor-ideas" is the same note), a title, and
optional description and tags in YAML front-matter. A description is a
one-line summary that list_notes and search_notes return, so keep one on
notes you want to recall.

- list_notes    list note summaries, optionally filtered by tag and path (full=true for bodies);
                path is a folder ("/work"), a note, or a glob ("/work/*-ideas")
- search_notes  search titles and bodies, optionally under a path; summaries carry
                the first matching line
- get_note      read the full markdown content of a note by slug
- create_note   write a new note named from its title; "dir/Title" puts it in a
                folder; never overwrites
- append_note   add text to the end of an existing note
- update_note   replace a note's body, or the whole file if content has front-matter
- delete_note   remove a note file from disk
- rename_note   rename a note in its folder and rewrite [[wikilinks]] to it
- history       commits newest first with the notes each changed; pass since
                (YYYY-MM-DD or RFC 3339) to see what changed since your last session

Writes from every jot process are serialized by a lock, so agents sharing the
notes dir do not lose each other's edits.

## Scratch

The note with slug "scratch" is a persistent scratch pad. It always exists
(created by jot init). Use it for quick dumps — get_note with slug "scratch"
to read it, or create_note won't overwrite it.

## Secure Notes

Notes with secure: true in front-matter are encrypted via kvlt (age + SSH keys).
get_note transparently decrypts secure notes. create_note accepts a secure flag
to encrypt the body on creation.

- encrypt_note  encrypt an existing plaintext note (body moves to kvlt)
- decrypt_note  decrypt a secure note back to plaintext on disk

IMPORTANT: Secure note operations require passphrase-free SSH keys configured in
the ssh_keys config option. There is no TTY available for interactive passphrase
prompts over MCP. If decryption fails, advise the user to configure a
passphrase-free key in their jot.yaml ssh_keys list.

## Tasks

Tasks are checkbox lines embedded in notes. They carry an id (t-xxxx), an
optional due date, optional inline #tags, and optionally the ids of tasks
that must be done first (after). Each listed task has its note_slug and file
line. Closing a task takes a reason, kept on its line as reason:. A task
cannot be closed while a task it waits on is open, unless forced.

- list_tasks    list tasks filtered by status (open/done/all), tag, path and done_since
- task_add      append a task to a note, creating the note if needed; returns its id
- task_done     mark a task complete, with a reason, by id, or by line (from
                list_tasks) or description
- tasks_due     list open tasks due today, this_week, or this_month

## Tags

Tags are free-form strings from YAML front-matter, inline #tags, and task tags.

- list_tags     list all unique tags across all notes`
