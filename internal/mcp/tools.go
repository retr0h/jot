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
	"slices"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/retr0h/jot/internal/jot"
)

// registerTools wires all jot MCP tools into the SDK server.
func (s *Server) registerTools() {
	mcpsdk.AddTool(s.mcp, &mcpsdk.Tool{
		Name:        "list_notes",
		Description: "List notes as summaries (slug, title, description, tags, dates, open tasks), optionally filtered by tag and by path: a folder such as /work, a note, or a glob such as /work/*-ideas. Pass full for bodies.",
	}, s.handleListNotes)

	mcpsdk.AddTool(s.mcp, &mcpsdk.Tool{
		Name:        "get_note",
		Description: "Read the full markdown content of a note by its slug. Transparently decrypts secure notes.",
	}, s.handleGetNote)

	mcpsdk.AddTool(s.mcp, &mcpsdk.Tool{
		Name:        "create_note",
		Description: "Create a new markdown note with YAML front-matter scaffold; a dir/Title title creates it in that folder, named from the title alone. Refuses to overwrite. Supports secure (encrypted) creation.",
	}, s.handleCreateNote)

	mcpsdk.AddTool(s.mcp, &mcpsdk.Tool{
		Name:        "append_note",
		Description: "Append text to the end of an existing note's body. Works on secure notes.",
	}, s.handleAppendNote)

	mcpsdk.AddTool(s.mcp, &mcpsdk.Tool{
		Name:        "update_note",
		Description: "Replace an existing note's body, keeping its front-matter. Content that starts with its own --- front-matter replaces the whole file.",
	}, s.handleUpdateNote)

	mcpsdk.AddTool(s.mcp, &mcpsdk.Tool{
		Name:        "delete_note",
		Description: "Delete a note's markdown file from disk.",
	}, s.handleDeleteNote)

	mcpsdk.AddTool(s.mcp, &mcpsdk.Tool{
		Name:        "search_notes",
		Description: "Case-insensitive substring search across note titles and bodies. path narrows it to a folder, note, or glob, as in list_notes. Returns summaries with the first matching line; pass full for bodies.",
	}, s.handleSearchNotes)

	mcpsdk.AddTool(s.mcp, &mcpsdk.Tool{
		Name:        "rename_note",
		Description: "Rename a note (rewrite title, slug, and all [[wikilinks]] referencing it).",
	}, s.handleRenameNote)

	mcpsdk.AddTool(s.mcp, &mcpsdk.Tool{
		Name:        "encrypt_note",
		Description: "Encrypt an existing plaintext note via kvlt. Requires passphrase-free SSH keys.",
	}, s.handleEncryptNote)

	mcpsdk.AddTool(s.mcp, &mcpsdk.Tool{
		Name:        "decrypt_note",
		Description: "Decrypt a secure note back to plaintext on disk. Requires passphrase-free SSH keys.",
	}, s.handleDecryptNote)

	mcpsdk.AddTool(s.mcp, &mcpsdk.Tool{
		Name:        "list_tasks",
		Description: `List tasks filtered by status ("open", "done", or "all"), tag, path (a folder, note, or glob, as in list_notes), and the date they were done. Each task carries its id, note_slug, file line, and the ids it waits on (after).`,
	}, s.handleListTasks)

	mcpsdk.AddTool(s.mcp, &mcpsdk.Tool{
		Name:        "task_add",
		Description: "Append a checkbox task to a note, creating the note when it does not exist, and return the id it was given. Can add it already done, and can wait on other tasks by id (after).",
	}, s.handleTaskAdd)

	mcpsdk.AddTool(s.mcp, &mcpsdk.Tool{
		Name:        "task_done",
		Description: "Mark a task complete with a required reason (appends done:YYYY-MM-DD | reason:...). Name it by id, or by line or description within a note. Refuses while a task it waits on is open, unless force is set.",
	}, s.handleTaskDone)

	mcpsdk.AddTool(s.mcp, &mcpsdk.Tool{
		Name:        "tasks_due",
		Description: `List open tasks due within a named period: "today", "this_week", or "this_month".`,
	}, s.handleTasksDue)

	mcpsdk.AddTool(s.mcp, &mcpsdk.Tool{
		Name:        "list_tags",
		Description: "List all unique tags across all notes (frontmatter + inline #tags + task tags).",
	}, s.handleListTags)

	mcpsdk.AddTool(s.mcp, &mcpsdk.Tool{
		Name:        "history",
		Description: "List recent commits to the notes, newest first, with the notes each changed. Filter by note and by time to see what changed since a session.",
	}, s.handleHistory)
}

// ─── argument structs ─────────────────────────────────────────────────────────

type listNotesArgs struct {
	Tag  string `json:"tag,omitempty"  jsonschema:"Filter to notes carrying this tag (optional)."`
	Path string `json:"path,omitempty" jsonschema:"Only notes at or under this path: a folder such as /work, a note, or a glob such as /work/*-ideas whose * stays in one folder (optional)."`
	Full bool   `json:"full,omitempty" jsonschema:"Return whole notes with bodies, tasks and links instead of summaries."`
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
	Query string `json:"query"          jsonschema:"Case-insensitive substring to search for."`
	Path  string `json:"path,omitempty" jsonschema:"Only notes at or under this path or glob, as in list_notes (optional)."`
	Full  bool   `json:"full,omitempty" jsonschema:"Return whole notes with bodies instead of summaries."`
}

type appendNoteArgs struct {
	Slug    string `json:"slug"    jsonschema:"Slug of an existing note, such as memory/projects."`
	Content string `json:"content" jsonschema:"Markdown to add at the end of the note's body."`
}

type updateNoteArgs struct {
	Slug    string `json:"slug"    jsonschema:"Slug of an existing note."`
	Content string `json:"content" jsonschema:"New body. Start it with a --- front-matter block to replace the whole file."`
}

type historyArgs struct {
	Slug  string `json:"slug,omitempty"  jsonschema:"Only commits that changed this note (optional)."`
	Since string `json:"since,omitempty" jsonschema:"Only commits at or after this time: YYYY-MM-DD or RFC 3339 (optional)."`
	Limit int    `json:"limit,omitempty" jsonschema:"Most commits to return, newest first (default 20, max 200)."`
}

type renameNoteArgs struct {
	Slug  string `json:"slug"  jsonschema:"Current note slug to rename."`
	Title string `json:"title" jsonschema:"New title for the note (slug is derived from this)."`
}

type encryptNoteArgs struct {
	Slug string `json:"slug" jsonschema:"Note slug to encrypt."`
}

type decryptNoteArgs struct {
	Slug string `json:"slug" jsonschema:"Note slug to decrypt."`
}

type listTasksArgs struct {
	Status    string `json:"status,omitempty"     jsonschema:"Task status filter: open, done, or all (default all)."`
	Tag       string `json:"tag,omitempty"        jsonschema:"Filter to tasks carrying this tag (optional)."`
	DoneSince string `json:"done_since,omitempty" jsonschema:"Only tasks done on or after this date, YYYY-MM-DD (optional; implies done)."`
	Path      string `json:"path,omitempty"       jsonschema:"Only tasks in notes at or under this path or glob, as in list_notes (optional)."`
}

type taskDoneArgs struct {
	ID     string `json:"id,omitempty"    jsonschema:"The task's id, as list_tasks reports it; needs no slug and takes precedence over line and desc."`
	Slug   string `json:"slug,omitempty"  jsonschema:"Note slug containing the task; required without an id."`
	Desc   string `json:"desc,omitempty"  jsonschema:"Task description: an exact match wins, else the only task containing it (case-insensitive)."`
	Reason string `json:"reason"          jsonschema:"Why the task is closed: one line, without | or #tags. Required."`
	Line   int    `json:"line,omitempty"  jsonschema:"The task's line in the note, as list_tasks reports it; takes precedence over desc."`
	Force  bool   `json:"force,omitempty" jsonschema:"Close it even though a task it waits on (after) is still open."`
}

type taskAddArgs struct {
	Slug  string   `json:"slug"            jsonschema:"Note slug to add the task to, such as todo or done/2026-11-03; created when missing."`
	Desc  string   `json:"desc"            jsonschema:"Task description: one line, without | or #tags."`
	Title string   `json:"title,omitempty" jsonschema:"Title for a new note (default: the slug's last segment)."`
	Due   string   `json:"due,omitempty"   jsonschema:"Due date: YYYY-MM-DD, today, tomorrow, a weekday name, or next week (optional)."`
	Done  string   `json:"done,omitempty"  jsonschema:"Add the task already done, on this date (same forms as due); omit for an open task."`
	Tags  []string `json:"tags,omitempty"  jsonschema:"Tags, without the leading # (optional)."`
	After []string `json:"after,omitempty" jsonschema:"IDs of tasks that must be done before this one (optional); each must exist."`
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
	notes, err := s.store.ListNotes(args.Tag, args.Path)
	if err != nil {
		return textResult("error: " + err.Error()), nil, nil
	}
	return textResult(notesResult(notes, "", args.Full)), nil, nil
}

// notesResult renders notes in full, or as summaries so an agent pays only
// for what it needs to pick a note.
func notesResult(
	notes []*jot.Note,
	query string,
	full bool,
) string {
	if full {
		return jsonOrErr(notes)
	}
	summaries := make([]jot.NoteSummary, 0, len(notes))
	for _, n := range notes {
		summaries = append(summaries, jot.Summarize(n, query))
	}
	return jsonOrErr(summaries)
}

func (s *Server) handleGetNote(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	args getNoteArgs,
) (*mcpsdk.CallToolResult, any, error) {
	if args.Slug == "" {
		return textResult("error: slug is required"), nil, nil
	}

	body, err := s.store.CatNote(args.Slug)
	if err != nil {
		return textResult(fmt.Sprintf("error: get note %q: %v", args.Slug, err)), nil, nil
	}
	return textResult(body), nil, nil
}

func (s *Server) handleCreateNote(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	args createNoteArgs,
) (*mcpsdk.CallToolResult, any, error) {
	if args.Title == "" {
		return textResult("error: title is required"), nil, nil
	}

	slug, notePath, err := s.store.CreateNote(args.Title, args.Content, args.Secure)
	if err != nil {
		return textResult(fmt.Sprintf("error: create note: %v", err)), nil, nil
	}

	return textResult(
		fmt.Sprintf(`{"slug": %q, "path": %q, "secure": %t}`, slug, notePath, args.Secure),
	), nil, nil
}

func (s *Server) handleAppendNote(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	args appendNoteArgs,
) (*mcpsdk.CallToolResult, any, error) {
	if args.Slug == "" {
		return textResult("error: slug is required"), nil, nil
	}
	if err := s.store.AppendNote(args.Slug, args.Content); err != nil {
		return textResult(fmt.Sprintf("error: append to note %q: %v", args.Slug, err)), nil, nil
	}
	return textResult(fmt.Sprintf(`{"appended": %q}`, args.Slug)), nil, nil
}

func (s *Server) handleUpdateNote(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	args updateNoteArgs,
) (*mcpsdk.CallToolResult, any, error) {
	if args.Slug == "" {
		return textResult("error: slug is required"), nil, nil
	}
	if err := s.store.UpdateNote(args.Slug, args.Content); err != nil {
		return textResult(fmt.Sprintf("error: update note %q: %v", args.Slug, err)), nil, nil
	}
	return textResult(fmt.Sprintf(`{"updated": %q}`, args.Slug)), nil, nil
}

func (s *Server) handleDeleteNote(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	args deleteNoteArgs,
) (*mcpsdk.CallToolResult, any, error) {
	if args.Slug == "" {
		return textResult("error: slug is required"), nil, nil
	}

	if err := s.store.DeleteNote(args.Slug); err != nil {
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
	results, err := s.store.SearchNotes(args.Query, args.Path)
	if err != nil {
		return textResult("error: " + err.Error()), nil, nil
	}
	return textResult(notesResult(results, args.Query, args.Full)), nil, nil
}

func (s *Server) handleRenameNote(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	args renameNoteArgs,
) (*mcpsdk.CallToolResult, any, error) {
	if args.Slug == "" {
		return textResult("error: slug is required"), nil, nil
	}
	if args.Title == "" {
		return textResult("error: title is required"), nil, nil
	}

	newSlug, err := s.store.RenameNote(args.Slug, args.Title)
	if err != nil {
		return textResult(fmt.Sprintf("error: rename note: %v", err)), nil, nil
	}

	return textResult(fmt.Sprintf(`{"old_slug": %q, "new_slug": %q}`, args.Slug, newSlug)), nil, nil
}

func (s *Server) handleEncryptNote(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	args encryptNoteArgs,
) (*mcpsdk.CallToolResult, any, error) {
	if args.Slug == "" {
		return textResult("error: slug is required"), nil, nil
	}

	if err := s.store.EncryptNote(args.Slug); err != nil {
		return textResult(fmt.Sprintf("error: encrypt note %q: %v", args.Slug, err)), nil, nil
	}

	return textResult(fmt.Sprintf(`{"encrypted": %q}`, args.Slug)), nil, nil
}

func (s *Server) handleDecryptNote(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	args decryptNoteArgs,
) (*mcpsdk.CallToolResult, any, error) {
	if args.Slug == "" {
		return textResult("error: slug is required"), nil, nil
	}

	if err := s.store.DecryptNote(args.Slug); err != nil {
		return textResult(fmt.Sprintf("error: decrypt note %q: %v", args.Slug, err)), nil, nil
	}

	return textResult(fmt.Sprintf(`{"decrypted": %q}`, args.Slug)), nil, nil
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
	if args.DoneSince != "" {
		if _, err := time.Parse("2006-01-02", args.DoneSince); err != nil {
			return textResult("error: done_since must be a YYYY-MM-DD date"), nil, nil
		}
		status = "done"
	}
	tasks, err := s.store.AllTasks(status, args.Tag, args.Path)
	if err != nil {
		return textResult("error: " + err.Error()), nil, nil
	}
	if args.DoneSince != "" {
		tasks = jot.DoneSince(tasks, args.DoneSince)
	}
	return textResult(jsonOrErr(tasks)), nil, nil
}

func (s *Server) handleTaskDone(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	args taskDoneArgs,
) (*mcpsdk.CallToolResult, any, error) {
	if args.ID != "" {
		if err := s.store.MarkTaskDoneByID(args.ID, args.Reason, args.Force); err != nil {
			return textResult(fmt.Sprintf("error: mark task done: %v", err)), nil, nil
		}
		return textResult(fmt.Sprintf(`{"id": %q, "done": true}`, args.ID)), nil, nil
	}
	if args.Slug == "" {
		return textResult("error: slug is required"), nil, nil
	}
	if args.Line > 0 {
		if err := s.store.MarkTaskDoneAt(args.Slug, args.Line, args.Reason, args.Force); err != nil {
			return textResult(fmt.Sprintf("error: mark task done: %v", err)), nil, nil
		}
		return textResult(
			fmt.Sprintf(`{"slug": %q, "line": %d, "done": true}`, args.Slug, args.Line),
		), nil, nil
	}
	if args.Desc == "" {
		return textResult("error: desc or line is required"), nil, nil
	}

	if err := s.store.MarkTaskDone(args.Slug, args.Desc, args.Reason, args.Force); err != nil {
		return textResult(fmt.Sprintf("error: mark task done: %v", err)), nil, nil
	}

	return textResult(
		fmt.Sprintf(`{"slug": %q, "task": %q, "done": true}`, args.Slug, args.Desc),
	), nil, nil
}

func (s *Server) handleTaskAdd(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	args taskAddArgs,
) (*mcpsdk.CallToolResult, any, error) {
	if args.Slug == "" {
		return textResult("error: slug is required"), nil, nil
	}
	if args.Desc == "" {
		return textResult("error: desc is required"), nil, nil
	}

	task, err := jot.NewTask(args.Desc, args.Due, args.Done, args.Tags, time.Now())
	if err != nil {
		return textResult(fmt.Sprintf("error: add task: %v", err)), nil, nil
	}
	task.After = args.After
	added, err := s.store.AddTask(args.Slug, args.Title, task)
	if err != nil {
		return textResult(fmt.Sprintf("error: add task: %v", err)), nil, nil
	}

	return textResult(jsonOrErr(map[string]any{
		"slug": args.Slug,
		"id":   added.ID,
		"line": jot.FormatTask(added),
	})), nil, nil
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

func (s *Server) handleListTags(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	_ listTagsArgs,
) (*mcpsdk.CallToolResult, any, error) {
	tags, err := s.store.AllTags()
	if err != nil {
		return textResult("error: " + err.Error()), nil, nil
	}
	return textResult(jsonOrErr(tags)), nil, nil
}

// historyEntry is one commit as the history tool reports it.
type historyEntry struct {
	Hash    string   `json:"hash"`
	Date    string   `json:"date"`
	Author  string   `json:"author"`
	Message string   `json:"message"`
	Notes   []string `json:"notes"` // slugs of the notes the commit changed
}

func (s *Server) handleHistory(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	args historyArgs,
) (*mcpsdk.CallToolResult, any, error) {
	if s.history == nil {
		return textResult("error: the notes dir is not a git repository"), nil, nil
	}

	slug := ""
	if args.Slug != "" {
		clean, err := jot.CleanSlug(args.Slug)
		if err != nil {
			return textResult("error: " + err.Error()), nil, nil
		}
		slug = clean
	}

	var since time.Time
	if args.Since != "" {
		t, err := parseSince(args.Since)
		if err != nil {
			return textResult("error: since must be YYYY-MM-DD or RFC 3339"), nil, nil
		}
		since = t
	}

	limit := min(max(args.Limit, 0), 200)
	if limit == 0 {
		limit = 20
	}
	// A secure note's edits change its vault file, not its .md, so filters
	// match on the notes each commit touched rather than on one file path.
	// ponytail: filtered after reading 1000 commits; page gitops.Log if repos outgrow it.
	fetch := limit
	if slug != "" || !since.IsZero() {
		fetch = 1000
	}

	entries, err := s.history.Log("", fetch)
	if err != nil {
		return textResult("error: " + err.Error()), nil, nil
	}

	out := make([]historyEntry, 0, min(len(entries), limit))
	for _, e := range entries {
		if len(out) == limit {
			break
		}
		notes := make([]string, 0, len(e.Files))
		for _, f := range e.Files {
			if note, ok := jot.NoteSlugForPath(f); ok {
				notes = append(notes, note)
			}
		}
		// Author dates need not follow log order, so an old one is skipped, not a stop.
		if e.Date.Before(since) || (slug != "" && !slices.Contains(notes, slug)) {
			continue
		}
		out = append(out, historyEntry{
			Hash:    e.Hash,
			Date:    e.Date.Format(time.RFC3339),
			Author:  e.Author,
			Message: e.Message,
			Notes:   notes,
		})
	}
	return textResult(jsonOrErr(out)), nil, nil
}

// parseSince reads an RFC 3339 time, or a YYYY-MM-DD date at local midnight.
func parseSince(value string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, nil
	}
	return time.ParseInLocation("2006-01-02", value, time.Local)
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
