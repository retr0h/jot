// Copyright (c) 2026 John Dewey

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
// DEALINGS IN THE SOFTWARE.

// Package mcp is the Model Context Protocol server for jot. Exposes every
// note, task, and label operation as an MCP tool an LLM agent can call.
//
// Architecturally: this package talks directly to a *jot.Store (SQLite) and
// the notes directory on disk. No HTTP daemon is involved — the agent spawns
// `jot mcp start` as a subprocess, which opens the store and pipes JSON-RPC
// over stdin/stdout. When the agent disconnects the process exits and the
// store is closed cleanly.
//
// Each tool is a thin adapter: extract params from CallToolParams.Arguments,
// call the appropriate store method or file operation, return a TextContent
// result with the JSON response.
package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/retr0h/jot/internal/jot"
)

// version is the MCP implementation version surfaced to agents on initialize.
const version = "0.1.0"

// Config bundles the runtime inputs for a jot MCP server. DBPath and
// NotesDir are required; Logger defaults to text-on-stderr when nil.
type Config struct {
	DBPath   string
	NotesDir string
	Logger   *slog.Logger
}

// Server is the jot MCP server. Holds the store and notes directory used by
// every tool handler, plus the underlying mcpsdk.Server. Constructed via
// New; the wire is driven by Run.
type Server struct {
	mcp      *mcpsdk.Server
	store    *jot.Store
	notesDir string
	logger   *slog.Logger
}

// New opens the SQLite store at cfg.DBPath, creates an MCP server, registers
// all 14 tools, and returns a ready-to-Run Server.
func New(cfg Config) (*Server, error) {
	if cfg.DBPath == "" {
		return nil, fmt.Errorf("mcp: DBPath required")
	}
	if cfg.NotesDir == "" {
		return nil, fmt.Errorf("mcp: NotesDir required")
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}

	store, err := jot.OpenStore(cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("mcp: open store: %w", err)
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
		mcp:      mcpSrv,
		store:    store,
		notesDir: cfg.NotesDir,
		logger:   cfg.Logger.With(slog.String("subsystem", "mcp")),
	}
	s.registerTools()
	return s, nil
}

// Run wires the MCP server to stdin/stdout and blocks until the transport
// closes (the spawning agent disconnects) or ctx cancels. Returns nil on
// clean shutdown.
func (s *Server) Run(ctx context.Context) error {
	s.logger.Info(
		"running",
		slog.String("transport", "stdio"),
	)
	return s.mcp.Run(ctx, &mcpsdk.StdioTransport{})
}

// Close releases the underlying SQLite store connection.
func (s *Server) Close() error {
	return s.store.Close()
}

// instructions is the MCP server's self-description, surfaced to the agent
// on initialize. Keep it short and concrete; agents read this to understand
// what the server can do without paging through every tool's description.
const instructions = `jot — terminal notes and tasks with linked labels.

This server exposes every jot operation over MCP so an LLM agent can read,
create, edit, and delete notes, tasks, and labels directly.

## Notes

Notes are markdown files stored on disk. Each note has a slug (the filename
without .md), a title, and optional labels. Full-text search is backed by
SQLite FTS5.

- list_notes    list all notes, optionally filtered by label
- get_note      read the full markdown content of a note by slug
- create_note   write a new markdown note and index it; parses @task markers
- edit_note     overwrite a note's content and re-index it
- delete_note   remove a note file and its store entry
- search_notes  full-text search across note titles and bodies

## Tasks

Tasks are @task markers embedded in notes. They carry an optional due date
and optional labels.

- list_tasks    list tasks filtered by status (open/done/all) and label
- tasks_due     list open tasks due today, this_week, or this_month
- complete_task mark a task as done
- reopen_task   revert a done task to open
- create_task   create a standalone task (not linked to a note body)

## Labels

Labels are free-form tags attached to notes and tasks.

- list_labels   list all labels
- create_label  create a new label by name
- delete_label  remove a label by id

## Secure notes

Notes flagged secure=true are stored encrypted on disk via age/kvlt. The MCP
server does not expose the encryption key — prefer get_note on non-secure
notes, or ensure the runtime environment has the key available.`
