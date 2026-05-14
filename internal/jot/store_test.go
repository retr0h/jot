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

package jot_test

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/retr0h/jot/internal/jot"
)

// openTestStore opens a fresh in-temp-dir database for a single test.
func openTestStore(t *testing.T) *jot.Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := jot.OpenStore(dbPath)
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// ─── Notes ────────────────────────────────────────────────────────────────────

func TestCreateNote(t *testing.T) {
	s := openTestStore(t)

	tests := []struct {
		name   string
		slug   string
		title  string
		secure bool
	}{
		{
			name:   "public note",
			slug:   "hello-world",
			title:  "Hello World",
			secure: false,
		},
		{
			name:   "secure note",
			slug:   "secret-stuff",
			title:  "Secret Stuff",
			secure: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			n, err := s.CreateNote(tc.slug, tc.title, tc.secure)
			if err != nil {
				t.Fatalf("CreateNote: %v", err)
			}
			if n.ID == 0 {
				t.Error("expected non-zero ID")
			}
			if n.Slug != tc.slug {
				t.Errorf("slug: got %q, want %q", n.Slug, tc.slug)
			}
			if n.Title != tc.title {
				t.Errorf("title: got %q, want %q", n.Title, tc.title)
			}
			if n.Secure != tc.secure {
				t.Errorf("secure: got %v, want %v", n.Secure, tc.secure)
			}
			if n.CreatedAt.IsZero() {
				t.Error("CreatedAt should not be zero")
			}
			if n.UpdatedAt.IsZero() {
				t.Error("UpdatedAt should not be zero")
			}
		})
	}
}

func TestCreateNoteDuplicateSlug(t *testing.T) {
	s := openTestStore(t)

	if _, err := s.CreateNote("dup", "First", false); err != nil {
		t.Fatalf("first CreateNote: %v", err)
	}
	if _, err := s.CreateNote("dup", "Second", false); err == nil {
		t.Fatal("expected error for duplicate slug, got nil")
	}
}

func TestGetNote(t *testing.T) {
	s := openTestStore(t)

	created, err := s.CreateNote("my-note", "My Note", false)
	if err != nil {
		t.Fatalf("CreateNote: %v", err)
	}

	got, err := s.GetNote(created.ID)
	if err != nil {
		t.Fatalf("GetNote: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("ID: got %d, want %d", got.ID, created.ID)
	}
	if got.Slug != created.Slug {
		t.Errorf("Slug: got %q, want %q", got.Slug, created.Slug)
	}
}

func TestGetNoteBySlug(t *testing.T) {
	s := openTestStore(t)

	if _, err := s.CreateNote("sluggable", "Sluggable Note", false); err != nil {
		t.Fatalf("CreateNote: %v", err)
	}

	got, err := s.GetNoteBySlug("sluggable")
	if err != nil {
		t.Fatalf("GetNoteBySlug: %v", err)
	}
	if got.Slug != "sluggable" {
		t.Errorf("Slug: got %q, want %q", got.Slug, "sluggable")
	}
}

func TestListNotes(t *testing.T) {
	s := openTestStore(t)

	slugs := []string{"alpha", "beta", "gamma"}
	for _, sl := range slugs {
		if _, err := s.CreateNote(sl, strings.ToUpper(sl), false); err != nil {
			t.Fatalf("CreateNote %q: %v", sl, err)
		}
	}

	t.Run("all notes", func(t *testing.T) {
		notes, err := s.ListNotes("")
		if err != nil {
			t.Fatalf("ListNotes: %v", err)
		}
		if len(notes) != 3 {
			t.Errorf("count: got %d, want 3", len(notes))
		}
	})

	t.Run("filtered by label", func(t *testing.T) {
		// Create a label and attach it only to "alpha".
		lbl, err := s.CreateLabel("work")
		if err != nil {
			t.Fatalf("CreateLabel: %v", err)
		}
		alphaNote, err := s.GetNoteBySlug("alpha")
		if err != nil {
			t.Fatalf("GetNoteBySlug: %v", err)
		}
		if err := s.AddNoteLabel(alphaNote.ID, lbl.ID); err != nil {
			t.Fatalf("AddNoteLabel: %v", err)
		}

		filtered, err := s.ListNotes("work")
		if err != nil {
			t.Fatalf("ListNotes(work): %v", err)
		}
		if len(filtered) != 1 {
			t.Fatalf("count: got %d, want 1", len(filtered))
		}
		if filtered[0].Slug != "alpha" {
			t.Errorf("slug: got %q, want %q", filtered[0].Slug, "alpha")
		}
	})
}

func TestDeleteNote(t *testing.T) {
	s := openTestStore(t)

	n, err := s.CreateNote("to-delete", "To Delete", false)
	if err != nil {
		t.Fatalf("CreateNote: %v", err)
	}

	if err := s.DeleteNote(n.ID); err != nil {
		t.Fatalf("DeleteNote: %v", err)
	}

	if _, err := s.GetNote(n.ID); err == nil {
		t.Error("expected error after delete, got nil")
	}
}

// ─── Tasks ────────────────────────────────────────────────────────────────────

func TestCreateTask(t *testing.T) {
	s := openTestStore(t)

	n, err := s.CreateNote("note-for-tasks", "Note for Tasks", false)
	if err != nil {
		t.Fatalf("CreateNote: %v", err)
	}

	tests := []struct {
		name        string
		description string
		lineNumber  int
	}{
		{
			name:        "first task",
			description: "Write tests",
			lineNumber:  5,
		},
		{
			name:        "second task",
			description: "Write implementation",
			lineNumber:  10,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			task, err := s.CreateTask(n.ID, tc.description, tc.lineNumber)
			if err != nil {
				t.Fatalf("CreateTask: %v", err)
			}
			if task.ID == 0 {
				t.Error("expected non-zero ID")
			}
			if task.NoteID != n.ID {
				t.Errorf("NoteID: got %d, want %d", task.NoteID, n.ID)
			}
			if task.Description != tc.description {
				t.Errorf("Description: got %q, want %q", task.Description, tc.description)
			}
			if task.Status != "open" {
				t.Errorf("Status: got %q, want %q", task.Status, "open")
			}
			if task.LineNumber != tc.lineNumber {
				t.Errorf("LineNumber: got %d, want %d", task.LineNumber, tc.lineNumber)
			}
		})
	}
}

func TestCompleteTask(t *testing.T) {
	s := openTestStore(t)

	n, _ := s.CreateNote("note", "Note", false)
	task, err := s.CreateTask(n.ID, "Do something", 1)
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	if err := s.CompleteTask(task.ID); err != nil {
		t.Fatalf("CompleteTask: %v", err)
	}

	got, err := s.GetTask(task.ID)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if got.Status != "done" {
		t.Errorf("Status: got %q, want %q", got.Status, "done")
	}
	if got.DoneAt == nil {
		t.Error("DoneAt should not be nil after completion")
	}
}

func TestReopenTask(t *testing.T) {
	s := openTestStore(t)

	n, _ := s.CreateNote("note", "Note", false)
	task, err := s.CreateTask(n.ID, "Do something", 1)
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	if err := s.CompleteTask(task.ID); err != nil {
		t.Fatalf("CompleteTask: %v", err)
	}
	if err := s.ReopenTask(task.ID); err != nil {
		t.Fatalf("ReopenTask: %v", err)
	}

	got, err := s.GetTask(task.ID)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if got.Status != "open" {
		t.Errorf("Status: got %q, want %q", got.Status, "open")
	}
	if got.DoneAt != nil {
		t.Error("DoneAt should be nil after reopen")
	}
}

func TestSetTaskDue(t *testing.T) {
	s := openTestStore(t)

	n, _ := s.CreateNote("note", "Note", false)
	task, err := s.CreateTask(n.ID, "Deadline task", 3)
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	due := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	t.Run("set due date", func(t *testing.T) {
		if err := s.SetTaskDue(task.ID, &due); err != nil {
			t.Fatalf("SetTaskDue: %v", err)
		}
		got, err := s.GetTask(task.ID)
		if err != nil {
			t.Fatalf("GetTask: %v", err)
		}
		if got.DueDate == nil {
			t.Fatal("DueDate should not be nil")
		}
		if !got.DueDate.Equal(due) {
			t.Errorf("DueDate: got %v, want %v", got.DueDate, due)
		}
	})

	t.Run("clear due date", func(t *testing.T) {
		if err := s.SetTaskDue(task.ID, nil); err != nil {
			t.Fatalf("SetTaskDue nil: %v", err)
		}
		got, err := s.GetTask(task.ID)
		if err != nil {
			t.Fatalf("GetTask: %v", err)
		}
		if got.DueDate != nil {
			t.Errorf("DueDate: got %v, want nil", got.DueDate)
		}
	})
}

func TestTasksDue(t *testing.T) {
	s := openTestStore(t)

	n, _ := s.CreateNote("note", "Note", false)

	dates := []time.Time{
		time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC), // before range
		time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC), // in range
		time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC), // in range
		time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC), // after range
	}

	for i, d := range dates {
		d := d
		task, err := s.CreateTask(n.ID, "task", i+1)
		if err != nil {
			t.Fatalf("CreateTask: %v", err)
		}
		if err := s.SetTaskDue(task.ID, &d); err != nil {
			t.Fatalf("SetTaskDue: %v", err)
		}
	}

	from := time.Date(2026, 5, 14, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 5, 21, 0, 0, 0, 0, time.UTC)

	tasks, err := s.TasksDue(from, to)
	if err != nil {
		t.Fatalf("TasksDue: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("count: got %d, want 2", len(tasks))
	}
}

func TestListTasks(t *testing.T) {
	s := openTestStore(t)

	n, _ := s.CreateNote("note", "Note", false)

	open1, _ := s.CreateTask(n.ID, "open one", 1)
	open2, _ := s.CreateTask(n.ID, "open two", 2)
	done1, _ := s.CreateTask(n.ID, "done one", 3)
	_ = s.CompleteTask(done1.ID)

	t.Run("all", func(t *testing.T) {
		tasks, err := s.ListTasks("all", "")
		if err != nil {
			t.Fatalf("ListTasks all: %v", err)
		}
		if len(tasks) != 3 {
			t.Errorf("count: got %d, want 3", len(tasks))
		}
	})

	t.Run("open", func(t *testing.T) {
		tasks, err := s.ListTasks("open", "")
		if err != nil {
			t.Fatalf("ListTasks open: %v", err)
		}
		if len(tasks) != 2 {
			t.Errorf("count: got %d, want 2", len(tasks))
		}
	})

	t.Run("done", func(t *testing.T) {
		tasks, err := s.ListTasks("done", "")
		if err != nil {
			t.Fatalf("ListTasks done: %v", err)
		}
		if len(tasks) != 1 {
			t.Errorf("count: got %d, want 1", len(tasks))
		}
	})

	t.Run("by label", func(t *testing.T) {
		lbl, _ := s.CreateLabel("urgent")
		_ = s.AddTaskLabel(open1.ID, lbl.ID)
		_ = s.AddTaskLabel(open2.ID, lbl.ID)

		tasks, err := s.ListTasks("open", "urgent")
		if err != nil {
			t.Fatalf("ListTasks open+urgent: %v", err)
		}
		if len(tasks) != 2 {
			t.Errorf("count: got %d, want 2", len(tasks))
		}
	})
}

// ─── Labels ───────────────────────────────────────────────────────────────────

func TestLabels(t *testing.T) {
	s := openTestStore(t)

	t.Run("create and list", func(t *testing.T) {
		names := []string{"work", "personal", "research"}
		for _, name := range names {
			lbl, err := s.CreateLabel(name)
			if err != nil {
				t.Fatalf("CreateLabel %q: %v", name, err)
			}
			if lbl.ID == 0 {
				t.Error("expected non-zero label ID")
			}
			if lbl.Name != name {
				t.Errorf("Name: got %q, want %q", lbl.Name, name)
			}
		}

		labels, err := s.ListLabels()
		if err != nil {
			t.Fatalf("ListLabels: %v", err)
		}
		if len(labels) != 3 {
			t.Fatalf("count: got %d, want 3", len(labels))
		}
		// ListLabels returns alphabetical order.
		if labels[0].Name != "personal" {
			t.Errorf("first label: got %q, want %q", labels[0].Name, "personal")
		}
	})

	t.Run("delete", func(t *testing.T) {
		lbl, _ := s.CreateLabel("ephemeral")
		if err := s.DeleteLabel(lbl.ID); err != nil {
			t.Fatalf("DeleteLabel: %v", err)
		}
		labels, _ := s.ListLabels()
		for _, l := range labels {
			if l.ID == lbl.ID {
				t.Error("deleted label still present in list")
			}
		}
	})
}

// ─── Note label association ───────────────────────────────────────────────────

func TestNoteLabelAssociation(t *testing.T) {
	s := openTestStore(t)

	note, _ := s.CreateNote("assoc-note", "Assoc Note", false)
	lbl, _ := s.CreateLabel("tagged")

	// Idempotent — two calls must not error.
	if err := s.AddNoteLabel(note.ID, lbl.ID); err != nil {
		t.Fatalf("first AddNoteLabel: %v", err)
	}
	if err := s.AddNoteLabel(note.ID, lbl.ID); err != nil {
		t.Fatalf("second AddNoteLabel (idempotent): %v", err)
	}

	filtered, err := s.ListNotes("tagged")
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}
	if len(filtered) != 1 {
		t.Fatalf("count: got %d, want 1", len(filtered))
	}
	if filtered[0].ID != note.ID {
		t.Errorf("note ID: got %d, want %d", filtered[0].ID, note.ID)
	}
}

// ─── Task label association ───────────────────────────────────────────────────

func TestTaskLabelAssociation(t *testing.T) {
	s := openTestStore(t)

	note, _ := s.CreateNote("task-label-note", "Task Label Note", false)
	task, _ := s.CreateTask(note.ID, "labelled task", 1)
	lbl, _ := s.CreateLabel("priority")

	// Idempotent.
	if err := s.AddTaskLabel(task.ID, lbl.ID); err != nil {
		t.Fatalf("first AddTaskLabel: %v", err)
	}
	if err := s.AddTaskLabel(task.ID, lbl.ID); err != nil {
		t.Fatalf("second AddTaskLabel (idempotent): %v", err)
	}

	tasks, err := s.ListTasks("open", "priority")
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("count: got %d, want 1", len(tasks))
	}
	if tasks[0].ID != task.ID {
		t.Errorf("task ID: got %d, want %d", tasks[0].ID, task.ID)
	}
}

// ─── FTS5 ─────────────────────────────────────────────────────────────────────

func TestIndexAndSearchNote(t *testing.T) {
	s := openTestStore(t)

	n, err := s.CreateNote("go-concurrency", "Go Concurrency", false)
	if err != nil {
		t.Fatalf("CreateNote: %v", err)
	}
	if err := s.IndexNote(n.ID, n.Title, "goroutines channels select pipelines"); err != nil {
		t.Fatalf("IndexNote: %v", err)
	}

	// Unrelated second note — should not appear in results.
	n2, _ := s.CreateNote("python-basics", "Python Basics", false)
	_ = s.IndexNote(n2.ID, n2.Title, "lists dicts loops comprehensions")

	tests := []struct {
		name        string
		query       string
		wantCount   int
		wantNoteID  int64
	}{
		{
			name:       "match title word",
			query:      "goroutines",
			wantCount:  1,
			wantNoteID: n.ID,
		},
		{
			name:       "match body word",
			query:      "channels",
			wantCount:  1,
			wantNoteID: n.ID,
		},
		{
			name:      "no match",
			query:     "javascript",
			wantCount: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			results, err := s.SearchNotes(tc.query)
			if err != nil {
				t.Fatalf("SearchNotes: %v", err)
			}
			if len(results) != tc.wantCount {
				t.Fatalf("count: got %d, want %d", len(results), tc.wantCount)
			}
			if tc.wantCount > 0 && results[0].NoteID != tc.wantNoteID {
				t.Errorf("NoteID: got %d, want %d", results[0].NoteID, tc.wantNoteID)
			}
		})
	}
}

func TestDeleteNoteRemovesFTS(t *testing.T) {
	s := openTestStore(t)

	n, _ := s.CreateNote("fts-delete", "FTS Delete", false)
	_ = s.IndexNote(n.ID, n.Title, "unique phrase that only this note has")

	// Sanity check — note is findable before deletion.
	results, err := s.SearchNotes("unique")
	if err != nil {
		t.Fatalf("SearchNotes before delete: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("pre-delete count: got %d, want 1", len(results))
	}

	if err := s.DeleteNote(n.ID); err != nil {
		t.Fatalf("DeleteNote: %v", err)
	}

	// FTS entry must also be gone.
	results, err = s.SearchNotes("unique")
	if err != nil {
		t.Fatalf("SearchNotes after delete: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("post-delete count: got %d, want 0", len(results))
	}
}
