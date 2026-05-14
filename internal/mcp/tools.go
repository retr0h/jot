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

// registerTools wires all 14 jot MCP tools into the SDK server. Input schemas
// are derived from the arg struct's json and jsonschema tags by the SDK.
func (s *Server) registerTools() {
	mcpsdk.AddTool(s.mcp, &mcpsdk.Tool{
		Name:        "list_notes",
		Description: "List all notes, optionally filtered by label name.",
	}, s.handleListNotes)

	mcpsdk.AddTool(s.mcp, &mcpsdk.Tool{
		Name:        "get_note",
		Description: "Read the full markdown content of a note by its slug.",
	}, s.handleGetNote)

	mcpsdk.AddTool(s.mcp, &mcpsdk.Tool{
		Name:        "create_note",
		Description: "Create a new markdown note. Writes the file, indexes it for FTS, and parses any @task markers in the content.",
	}, s.handleCreateNote)

	mcpsdk.AddTool(s.mcp, &mcpsdk.Tool{
		Name:        "edit_note",
		Description: "Overwrite a note's markdown content and re-index it for FTS.",
	}, s.handleEditNote)

	mcpsdk.AddTool(s.mcp, &mcpsdk.Tool{
		Name:        "delete_note",
		Description: "Delete a note's markdown file and remove it from the store.",
	}, s.handleDeleteNote)

	mcpsdk.AddTool(s.mcp, &mcpsdk.Tool{
		Name:        "search_notes",
		Description: "Full-text search across note titles and bodies using SQLite FTS5.",
	}, s.handleSearchNotes)

	mcpsdk.AddTool(s.mcp, &mcpsdk.Tool{
		Name:        "list_tasks",
		Description: `List tasks filtered by status ("open", "done", or "all") and optionally by label.`,
	}, s.handleListTasks)

	mcpsdk.AddTool(s.mcp, &mcpsdk.Tool{
		Name:        "tasks_due",
		Description: `List open tasks due within a named period: "today", "this_week", or "this_month".`,
	}, s.handleTasksDue)

	mcpsdk.AddTool(s.mcp, &mcpsdk.Tool{
		Name:        "complete_task",
		Description: "Mark a task as done.",
	}, s.handleCompleteTask)

	mcpsdk.AddTool(s.mcp, &mcpsdk.Tool{
		Name:        "reopen_task",
		Description: "Revert a done task back to open.",
	}, s.handleReopenTask)

	mcpsdk.AddTool(s.mcp, &mcpsdk.Tool{
		Name:        "list_labels",
		Description: "List all labels ordered by name.",
	}, s.handleListLabels)

	mcpsdk.AddTool(s.mcp, &mcpsdk.Tool{
		Name:        "create_label",
		Description: "Create a new label by name.",
	}, s.handleCreateLabel)

	mcpsdk.AddTool(s.mcp, &mcpsdk.Tool{
		Name:        "delete_label",
		Description: "Delete a label by id. Cascades to note_labels and task_labels.",
	}, s.handleDeleteLabel)

	mcpsdk.AddTool(s.mcp, &mcpsdk.Tool{
		Name:        "create_task",
		Description: "Create a standalone task. Optionally link to an existing note and/or set a due date.",
	}, s.handleCreateTask)
}

// ─── argument structs ─────────────────────────────────────────────────────────

type listNotesArgs struct {
	Label string `json:"label,omitempty" jsonschema:"Filter to notes carrying this label (optional)."`
}

type getNoteArgs struct {
	Slug string `json:"slug" jsonschema:"The note slug (filename without .md)."`
}

type createNoteArgs struct {
	Title   string `json:"title"   jsonschema:"Note title (used to derive the slug)."`
	Content string `json:"content" jsonschema:"Markdown content of the note."`
}

type editNoteArgs struct {
	Slug    string `json:"slug"    jsonschema:"The note slug to edit."`
	Content string `json:"content" jsonschema:"New markdown content."`
}

type deleteNoteArgs struct {
	Slug string `json:"slug" jsonschema:"The note slug to delete."`
}

type searchNotesArgs struct {
	Query string `json:"query" jsonschema:"FTS5 query string."`
}

type listTasksArgs struct {
	Status string `json:"status,omitempty" jsonschema:"Task status filter: open, done, or all (default all)."`
	Label  string `json:"label,omitempty"  jsonschema:"Filter to tasks carrying this label (optional)."`
}

type tasksDueArgs struct {
	Period string `json:"period" jsonschema:"Period to query: today, this_week, or this_month."`
}

type completeTaskArgs struct {
	ID int64 `json:"id" jsonschema:"Task ID to complete."`
}

type reopenTaskArgs struct {
	ID int64 `json:"id" jsonschema:"Task ID to reopen."`
}

type listLabelsArgs struct{}

type createLabelArgs struct {
	Name string `json:"name" jsonschema:"Label name (must be unique)."`
}

type deleteLabelArgs struct {
	ID int64 `json:"id" jsonschema:"Label ID to delete."`
}

type createTaskArgs struct {
	Description string `json:"description"        jsonschema:"Task description."`
	NoteSlug    string `json:"note_slug,omitempty" jsonschema:"Slug of an existing note to link this task to (optional)."`
	DueDate     string `json:"due_date,omitempty"  jsonschema:"Due date: YYYY-MM-DD or natural language (today, tomorrow, next week, weekday name)."`
}

// ─── handlers ─────────────────────────────────────────────────────────────────

func (s *Server) handleListNotes(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	args listNotesArgs,
) (*mcpsdk.CallToolResult, any, error) {
	notes, err := s.store.ListNotes(args.Label)
	if err != nil {
		return textResult("error: " + err.Error()), nil, nil
	}
	return textResult(jsonOrErr(notes)), nil, nil
}

func (s *Server) handleGetNote(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	args getNoteArgs,
) (*mcpsdk.CallToolResult, any, error) {
	if args.Slug == "" {
		return textResult("error: slug is required"), nil, nil
	}
	notePath := filepath.Join(s.notesDir, args.Slug+".md")
	content, err := os.ReadFile(notePath)
	if err != nil {
		return textResult(fmt.Sprintf("error: read note %q: %v", args.Slug, err)), nil, nil
	}
	return textResult(string(content)), nil, nil
}

func (s *Server) handleCreateNote(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	args createNoteArgs,
) (*mcpsdk.CallToolResult, any, error) {
	if args.Title == "" {
		return textResult("error: title is required"), nil, nil
	}
	if args.Content == "" {
		return textResult("error: content is required"), nil, nil
	}

	slug := jot.NewSlug(args.Title, time.Now())

	if err := os.MkdirAll(s.notesDir, 0o700); err != nil {
		return textResult(fmt.Sprintf("error: create notes dir: %v", err)), nil, nil
	}
	notePath := filepath.Join(s.notesDir, slug+".md")
	if err := os.WriteFile(notePath, []byte(args.Content), 0o600); err != nil {
		return textResult(fmt.Sprintf("error: write note file: %v", err)), nil, nil
	}

	note, err := s.store.CreateNote(slug, args.Title, false)
	if err != nil {
		return textResult(fmt.Sprintf("error: create note in store: %v", err)), nil, nil
	}

	if err := s.store.IndexNote(note.ID, args.Title, args.Content); err != nil {
		return textResult(fmt.Sprintf("error: index note: %v", err)), nil, nil
	}

	// Parse @task markers and create tasks in the store.
	rawTasks := jot.ParseTasks(args.Content)
	for _, raw := range rawTasks {
		task, err := s.store.CreateTask(note.ID, raw.Description, raw.Line)
		if err != nil {
			continue
		}
		if raw.DueDate != "" {
			due, err := jot.ParseDate(raw.DueDate, time.Now())
			if err == nil {
				_ = s.store.SetTaskDue(task.ID, &due)
			}
		}
		for _, labelName := range raw.Labels {
			label, err := s.store.CreateLabel(labelName)
			if err != nil {
				// Label may already exist; look it up.
				labels, lerr := s.store.ListLabels()
				if lerr != nil {
					continue
				}
				for _, l := range labels {
					if l.Name == labelName {
						label = l
						break
					}
				}
				if label == nil {
					continue
				}
			}
			_ = s.store.AddTaskLabel(task.ID, label.ID)
		}
	}

	return textResult(jsonOrErr(note)), nil, nil
}

func (s *Server) handleEditNote(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	args editNoteArgs,
) (*mcpsdk.CallToolResult, any, error) {
	if args.Slug == "" {
		return textResult("error: slug is required"), nil, nil
	}
	if args.Content == "" {
		return textResult("error: content is required"), nil, nil
	}

	note, err := s.store.GetNoteBySlug(args.Slug)
	if err != nil {
		return textResult(fmt.Sprintf("error: get note %q: %v", args.Slug, err)), nil, nil
	}

	notePath := filepath.Join(s.notesDir, args.Slug+".md")
	if err := os.WriteFile(notePath, []byte(args.Content), 0o600); err != nil {
		return textResult(fmt.Sprintf("error: write note file: %v", err)), nil, nil
	}

	if err := s.store.IndexNote(note.ID, note.Title, args.Content); err != nil {
		return textResult(fmt.Sprintf("error: re-index note: %v", err)), nil, nil
	}

	return textResult(jsonOrErr(note)), nil, nil
}

func (s *Server) handleDeleteNote(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	args deleteNoteArgs,
) (*mcpsdk.CallToolResult, any, error) {
	if args.Slug == "" {
		return textResult("error: slug is required"), nil, nil
	}

	note, err := s.store.GetNoteBySlug(args.Slug)
	if err != nil {
		return textResult(fmt.Sprintf("error: get note %q: %v", args.Slug, err)), nil, nil
	}

	notePath := filepath.Join(s.notesDir, args.Slug+".md")
	// Best-effort file removal; proceed even if the file is already gone.
	_ = os.Remove(notePath)

	if err := s.store.DeleteNote(note.ID); err != nil {
		return textResult(fmt.Sprintf("error: delete note from store: %v", err)), nil, nil
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
	results, err := s.store.SearchNotes(args.Query)
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
	tasks, err := s.store.ListTasks(status, args.Label)
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

	tasks, err := s.store.TasksDue(from, to)
	if err != nil {
		return textResult("error: " + err.Error()), nil, nil
	}
	return textResult(jsonOrErr(tasks)), nil, nil
}

func (s *Server) handleCompleteTask(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	args completeTaskArgs,
) (*mcpsdk.CallToolResult, any, error) {
	if args.ID == 0 {
		return textResult("error: id is required"), nil, nil
	}
	if err := s.store.CompleteTask(args.ID); err != nil {
		return textResult("error: " + err.Error()), nil, nil
	}
	return textResult(fmt.Sprintf(`{"completed": %d}`, args.ID)), nil, nil
}

func (s *Server) handleReopenTask(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	args reopenTaskArgs,
) (*mcpsdk.CallToolResult, any, error) {
	if args.ID == 0 {
		return textResult("error: id is required"), nil, nil
	}
	if err := s.store.ReopenTask(args.ID); err != nil {
		return textResult("error: " + err.Error()), nil, nil
	}
	return textResult(fmt.Sprintf(`{"reopened": %d}`, args.ID)), nil, nil
}

func (s *Server) handleListLabels(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	_ listLabelsArgs,
) (*mcpsdk.CallToolResult, any, error) {
	labels, err := s.store.ListLabels()
	if err != nil {
		return textResult("error: " + err.Error()), nil, nil
	}
	return textResult(jsonOrErr(labels)), nil, nil
}

func (s *Server) handleCreateLabel(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	args createLabelArgs,
) (*mcpsdk.CallToolResult, any, error) {
	if args.Name == "" {
		return textResult("error: name is required"), nil, nil
	}
	label, err := s.store.CreateLabel(args.Name)
	if err != nil {
		return textResult("error: " + err.Error()), nil, nil
	}
	return textResult(jsonOrErr(label)), nil, nil
}

func (s *Server) handleDeleteLabel(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	args deleteLabelArgs,
) (*mcpsdk.CallToolResult, any, error) {
	if args.ID == 0 {
		return textResult("error: id is required"), nil, nil
	}
	if err := s.store.DeleteLabel(args.ID); err != nil {
		return textResult("error: " + err.Error()), nil, nil
	}
	return textResult(fmt.Sprintf(`{"deleted": %d}`, args.ID)), nil, nil
}

func (s *Server) handleCreateTask(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	args createTaskArgs,
) (*mcpsdk.CallToolResult, any, error) {
	if args.Description == "" {
		return textResult("error: description is required"), nil, nil
	}

	var noteID int64
	if args.NoteSlug != "" {
		note, err := s.store.GetNoteBySlug(args.NoteSlug)
		if err != nil {
			return textResult(fmt.Sprintf("error: get note %q: %v", args.NoteSlug, err)), nil, nil
		}
		noteID = note.ID
	}

	task, err := s.store.CreateTask(noteID, args.Description, 0)
	if err != nil {
		return textResult(fmt.Sprintf("error: create task: %v", err)), nil, nil
	}

	if args.DueDate != "" {
		due, err := jot.ParseDate(args.DueDate, time.Now())
		if err == nil {
			_ = s.store.SetTaskDue(task.ID, &due)
		}
	}

	return textResult(jsonOrErr(task)), nil, nil
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
