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

package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/retr0h/jot/internal/jot"
)

// secureNoteBody retrieves the decrypted body of a secure note from kvlt.
// Returns an error string if decryption fails (e.g. passphrase-protected key
// with no TTY available). MCP requires passphrase-free SSH keys for secure
// note access.
func (s *Server) secureNoteBody(
	ctx context.Context,
	slug string,
) (string, error) {
	store, err := jot.NewSecureStore(s.configDir, s.sshKeys, nil)
	if err != nil {
		return "", fmt.Errorf("open secure store: %w", err)
	}
	return store.ReadNote(ctx, slug)
}

// registerTools wires all 8 jot MCP tools into the SDK server.
func (s *Server) registerTools() {
	mcpsdk.AddTool(s.mcp, &mcpsdk.Tool{
		Name:        "list_notes",
		Description: "List all notes, optionally filtered by tag name.",
	}, s.handleListNotes)

	mcpsdk.AddTool(s.mcp, &mcpsdk.Tool{
		Name:        "get_note",
		Description: "Read the full markdown content of a note by its slug.",
	}, s.handleGetNote)

	mcpsdk.AddTool(s.mcp, &mcpsdk.Tool{
		Name:        "create_note",
		Description: "Create a new markdown note with YAML front-matter scaffold.",
	}, s.handleCreateNote)

	mcpsdk.AddTool(s.mcp, &mcpsdk.Tool{
		Name:        "delete_note",
		Description: "Delete a note's markdown file from disk.",
	}, s.handleDeleteNote)

	mcpsdk.AddTool(s.mcp, &mcpsdk.Tool{
		Name:        "search_notes",
		Description: "Case-insensitive substring search across note titles and bodies.",
	}, s.handleSearchNotes)

	mcpsdk.AddTool(s.mcp, &mcpsdk.Tool{
		Name:        "list_tasks",
		Description: `List tasks filtered by status ("open", "done", or "all") and optionally by tag.`,
	}, s.handleListTasks)

	mcpsdk.AddTool(s.mcp, &mcpsdk.Tool{
		Name:        "tasks_due",
		Description: `List open tasks due within a named period: "today", "this_week", or "this_month".`,
	}, s.handleTasksDue)

	mcpsdk.AddTool(s.mcp, &mcpsdk.Tool{
		Name:        "list_tags",
		Description: "List all unique tags across all notes (frontmatter + inline #tags + task tags).",
	}, s.handleListTags)
}

// ─── argument structs ─────────────────────────────────────────────────────────

type listNotesArgs struct {
	Tag string `json:"tag,omitempty" jsonschema:"Filter to notes carrying this tag (optional)."`
}

type getNoteArgs struct {
	Slug string `json:"slug" jsonschema:"The note slug (filename without .md)."`
}

type createNoteArgs struct {
	Title   string `json:"title"            jsonschema:"Note title (used to derive the slug)."`
	Content string `json:"content"          jsonschema:"Markdown content of the note (optional; scaffold used when empty)."`
	Secure  bool   `json:"secure,omitempty" jsonschema:"When true, encrypt the note body via kvlt (requires passphrase-free SSH keys)."`
}

type deleteNoteArgs struct {
	Slug string `json:"slug" jsonschema:"The note slug to delete."`
}

type searchNotesArgs struct {
	Query string `json:"query" jsonschema:"Case-insensitive substring to search for."`
}

type listTasksArgs struct {
	Status string `json:"status,omitempty" jsonschema:"Task status filter: open, done, or all (default all)."`
	Tag    string `json:"tag,omitempty"    jsonschema:"Filter to tasks carrying this tag (optional)."`
}

type tasksDueArgs struct {
	Period string `json:"period" jsonschema:"Period to query: today, this_week, or this_month."`
}

type listTagsArgs struct{}

// ─── handlers ─────────────────────────────────────────────────────────────────

func (s *Server) handleListNotes(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	args listNotesArgs,
) (*mcpsdk.CallToolResult, any, error) {
	notes, err := s.notes.ListNotes(s.notesDir, args.Tag)
	if err != nil {
		return textResult("error: " + err.Error()), nil, nil
	}
	return textResult(jsonOrErr(notes)), nil, nil
}

func (s *Server) handleGetNote(
	ctx context.Context,
	_ *mcpsdk.CallToolRequest,
	args getNoteArgs,
) (*mcpsdk.CallToolResult, any, error) {
	if args.Slug == "" {
		return textResult("error: slug is required"), nil, nil
	}
	note, err := s.notes.FindNote(s.notesDir, args.Slug)
	if err != nil {
		return textResult(fmt.Sprintf("error: get note %q: %v", args.Slug, err)), nil, nil
	}

	if note.Secure {
		body, err := s.secureNoteBody(ctx, args.Slug)
		if err != nil {
			return textResult(
				fmt.Sprintf(
					"error: decrypt note %q: %v (MCP requires passphrase-free SSH keys)",
					args.Slug,
					err,
				),
			), nil, nil
		}
		return textResult(body), nil, nil
	}

	notePath := filepath.Join(s.notesDir, args.Slug+".md")
	content, err := os.ReadFile(notePath)
	if err != nil {
		return textResult(fmt.Sprintf("error: read note file %q: %v", args.Slug, err)), nil, nil
	}
	return textResult(string(content)), nil, nil
}

func (s *Server) handleCreateNote(
	ctx context.Context,
	_ *mcpsdk.CallToolRequest,
	args createNoteArgs,
) (*mcpsdk.CallToolResult, any, error) {
	if args.Title == "" {
		return textResult("error: title is required"), nil, nil
	}

	slug := jot.NewSlug(args.Title, time.Now())

	if err := os.MkdirAll(s.notesDir, 0o700); err != nil {
		return textResult(fmt.Sprintf("error: create notes dir: %v", err)), nil, nil
	}

	notePath := filepath.Join(s.notesDir, slug+".md")
	body := args.Content

	if args.Secure {
		store, err := jot.NewSecureStore(s.configDir, s.sshKeys, nil)
		if err != nil {
			return textResult(fmt.Sprintf("error: open secure store: %v", err)), nil, nil
		}
		if err := store.WriteNote(ctx, slug, body); err != nil {
			return textResult(fmt.Sprintf("error: encrypt note: %v", err)), nil, nil
		}
		scaffold := fmt.Sprintf(
			"---\ntitle: %q\ntags: []\ncreated: %s\nsecure: true\n---\n",
			args.Title,
			time.Now().Format("2006-01-02"),
		)
		if err := os.WriteFile(notePath, []byte(scaffold), 0o600); err != nil {
			return textResult(fmt.Sprintf("error: write note file: %v", err)), nil, nil
		}
	} else {
		content := body
		if content == "" {
			content = jot.ScaffoldFrontmatter(args.Title, time.Now().Format("2006-01-02"))
		}
		if err := os.WriteFile(notePath, []byte(content), 0o600); err != nil {
			return textResult(fmt.Sprintf("error: write note file: %v", err)), nil, nil
		}
	}

	return textResult(
		fmt.Sprintf(`{"slug": %q, "path": %q, "secure": %t}`, slug, notePath, args.Secure),
	), nil, nil
}

func (s *Server) handleDeleteNote(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	args deleteNoteArgs,
) (*mcpsdk.CallToolResult, any, error) {
	if args.Slug == "" {
		return textResult("error: slug is required"), nil, nil
	}

	notePath := filepath.Join(s.notesDir, args.Slug+".md")
	if err := os.Remove(notePath); err != nil && !os.IsNotExist(err) {
		return textResult(fmt.Sprintf("error: delete note %q: %v", args.Slug, err)), nil, nil
	}

	return textResult(fmt.Sprintf(`{"deleted": %q}`, args.Slug)), nil, nil
}

func (s *Server) handleSearchNotes(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	args searchNotesArgs,
) (*mcpsdk.CallToolResult, any, error) {
	if args.Query == "" {
		return textResult("error: query is required"), nil, nil
	}
	results, err := s.notes.SearchNotes(s.notesDir, args.Query)
	if err != nil {
		return textResult("error: " + err.Error()), nil, nil
	}
	return textResult(jsonOrErr(results)), nil, nil
}

func (s *Server) handleListTasks(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	args listTasksArgs,
) (*mcpsdk.CallToolResult, any, error) {
	status := args.Status
	if status == "" {
		status = "all"
	}
	tasks, err := s.tasks.AllTasks(s.notesDir, status, args.Tag)
	if err != nil {
		return textResult("error: " + err.Error()), nil, nil
	}
	return textResult(jsonOrErr(tasks)), nil, nil
}

func (s *Server) handleTasksDue(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	args tasksDueArgs,
) (*mcpsdk.CallToolResult, any, error) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	var from, to time.Time
	switch args.Period {
	case "today":
		from = today
		to = today
	case "this_week":
		from = today
		to = today.AddDate(0, 0, 6)
	case "this_month":
		from = today
		to = time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location())
	default:
		return textResult("error: period must be today, this_week, or this_month"), nil, nil
	}

	tasks, err := s.tasks.TasksDue(s.notesDir, from, to)
	if err != nil {
		return textResult("error: " + err.Error()), nil, nil
	}
	return textResult(jsonOrErr(tasks)), nil, nil
}

func (s *Server) handleListTags(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	_ listTagsArgs,
) (*mcpsdk.CallToolResult, any, error) {
	tags, err := s.tags.AllTags(s.notesDir)
	if err != nil {
		return textResult("error: " + err.Error()), nil, nil
	}
	return textResult(jsonOrErr(tags)), nil, nil
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// textResult wraps a string as an MCP CallToolResult with a single TextContent
// block — the canonical response shape for every tool.
func textResult(text string) *mcpsdk.CallToolResult {
	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{
			&mcpsdk.TextContent{Text: text},
		},
	}
}

// jsonOrErr renders a value as pretty JSON for an MCP TextContent response,
// or returns a descriptive error string if marshaling fails.
func jsonOrErr(v any) string {
	b, err := jsonMarshalIndent(v)
	if err != nil {
		return "error: marshal response: " + err.Error()
	}
	return string(b)
}
