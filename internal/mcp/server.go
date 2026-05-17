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
//
// Architecturally: this package talks to the notes directory on disk through
// narrow consumer-seam interfaces (noteLister, taskLister, tagLister). No
// SQLite is involved. The agent spawns `jot mcp start` as a subprocess, which
// pipes JSON-RPC over stdin/stdout. When the agent disconnects the process
// exits cleanly.
//
// Each tool is a thin adapter: extract params from CallToolParams.Arguments,
// call the appropriate interface method or file operation, return a TextContent
// result with the JSON response.
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

// ─── consumer-seam interfaces ─────────────────────────────────────────────────

// noteLister reads notes from the filesystem.
type noteLister interface {
	ListNotes(notesDir string, tagFilter string) ([]*jot.Note, error)
	SearchNotes(notesDir string, query string) ([]*jot.Note, error)
	FindNote(notesDir string, slug string) (*jot.Note, error)
}

// taskLister reads tasks aggregated across note files.
type taskLister interface {
	AllTasks(notesDir string, status string, tagFilter string) ([]jot.Task, error)
	TasksDue(notesDir string, from time.Time, to time.Time) ([]jot.Task, error)
}

// tagLister enumerates tags across the notes directory.
type tagLister interface {
	AllTags(notesDir string) ([]string, error)
}

// ─── fileNotesProvider ───────────────────────────────────────────────────────

// fileNotesProvider satisfies noteLister, taskLister, and tagLister by
// delegating directly to the jot package functions that scan the filesystem.
type fileNotesProvider struct{}

func (fileNotesProvider) ListNotes(notesDir string, tagFilter string) ([]*jot.Note, error) {
	return jot.ListNotes(notesDir, tagFilter)
}

func (fileNotesProvider) SearchNotes(notesDir string, query string) ([]*jot.Note, error) {
	return jot.SearchNotes(notesDir, query)
}

func (fileNotesProvider) FindNote(notesDir string, slug string) (*jot.Note, error) {
	return jot.FindNote(notesDir, slug)
}

func (fileNotesProvider) AllTasks(
	notesDir string,
	status string,
	tagFilter string,
) ([]jot.Task, error) {
	return jot.AllTasks(notesDir, status, tagFilter)
}

func (fileNotesProvider) TasksDue(
	notesDir string,
	from time.Time,
	to time.Time,
) ([]jot.Task, error) {
	return jot.TasksDue(notesDir, from, to)
}

func (fileNotesProvider) AllTags(notesDir string) ([]string, error) {
	return jot.AllTags(notesDir)
}

// ─── Config and Server ───────────────────────────────────────────────────────

// Config bundles the runtime inputs for a jot MCP server. NotesDir is
// required; Logger defaults to text-on-stderr when nil. ConfigDir and
// SSHKeys are needed for transparent decryption of secure notes.
type Config struct {
	NotesDir  string
	ConfigDir string
	SSHKeys   []string
	Logger    *slog.Logger
}

// Server is the jot MCP server. Holds the notes directory used by every tool
// handler, plus the underlying mcpsdk.Server. Constructed via New; the wire
// is driven by Run.
type Server struct {
	mcp       *mcpsdk.Server
	notesDir  string
	configDir string
	sshKeys   []string
	notes     noteLister
	tasks     taskLister
	tags      tagLister
	logger    *slog.Logger
}

// New creates an MCP server, registers all 8 tools, and returns a
// ready-to-Run Server. No database connection is opened.
func New(cfg Config) (*Server, error) {
	if cfg.NotesDir == "" {
		return nil, fmt.Errorf("mcp: NotesDir required")
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

	provider := fileNotesProvider{}
	s := &Server{
		mcp:       mcpSrv,
		notesDir:  cfg.NotesDir,
		configDir: cfg.ConfigDir,
		sshKeys:   cfg.SSHKeys,
		notes:     provider,
		tasks:     provider,
		tags:      provider,
		logger:    cfg.Logger.With(slog.String("subsystem", "mcp")),
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

// Close is a no-op; retained for interface compatibility with callers that
// defer s.Close(). Returns nil.
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

## Tasks

Tasks are @task markers embedded in notes. They carry an optional due date
and optional inline #tags.

- list_tasks    list tasks filtered by status (open/done/all) and tag
- tasks_due     list open tasks due today, this_week, or this_month

## Tags

Tags are free-form strings from YAML front-matter, inline #tags, and task tags.

- list_tags     list all unique tags across all notes`
