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

	"github.com/retr0h/jot/internal/jot"
)

// version is the MCP implementation version surfaced to agents on initialize.
const version = "0.1.0"

// ─── consumer-seam interface ─────────────────────────────────────────────────

// Store is the narrow consumer-seam interface this MCP server depends on.
// Concrete *jot.Service satisfies it structurally — the compiler verifies
// at the assignment site in New().
type Store interface {
	ListNotes(tagFilter string) ([]*jot.Note, error)
	SearchNotes(query string) ([]*jot.Note, error)
	FindNoteBySlug(slug string) (*jot.Note, error)
	CatNote(slug string) (string, error)
	CreateNote(title string, content string, secure bool) (string, string, error)
	DeleteNote(slug string) error
	EncryptNote(slug string) error
	DecryptNote(slug string) error
	RenameNote(oldSlug string, newTitle string) (string, error)
	AllTasks(status string, tagFilter string) ([]jot.Task, error)
	TasksDue(from time.Time, to time.Time) ([]jot.Task, error)
	MarkTaskDone(slug string, desc string) error
	AllTags() ([]string, error)
}

// ─── Config and Server ───────────────────────────────────────────────────────

// Config bundles the runtime inputs for a jot MCP server.
type Config struct {
	Store  Store
	Logger *slog.Logger
}

// Server is the jot MCP server. Holds a Store (the narrow consumer
// surface) used by every tool handler, plus the underlying mcpsdk.Server.
// Constructed via New; the wire is driven by Run.
type Server struct {
	mcp    *mcpsdk.Server
	store  Store
	logger *slog.Logger
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
		mcp:    mcpSrv,
		store:  cfg.Store,
		logger: cfg.Logger.With(slog.String("subsystem", "mcp")),
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
create, and delete notes, tasks, and tags directly.

## Notes

Notes are markdown files stored on disk. Each note has a slug (the filename
without .md), a title, and optional tags in YAML front-matter.

- list_notes    list all notes, optionally filtered by tag
- get_note      read the full markdown content of a note by slug
- create_note   write a new markdown note with front-matter scaffold
- delete_note   remove a note file from disk
- search_notes  case-insensitive substring search across note titles and bodies
- rename_note   rename a note and rewrite all [[wikilinks]] referencing it

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

Tasks are @task markers embedded in notes. They carry an optional due date
and optional inline #tags.

- list_tasks    list tasks filtered by status (open/done/all) and tag
- task_done     mark a task as complete (appends done:YYYY-MM-DD)
- tasks_due     list open tasks due today, this_week, or this_month

## Tags

Tags are free-form strings from YAML front-matter, inline #tags, and task tags.

- list_tags     list all unique tags across all notes`
