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
	"testing"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/retr0h/jot/internal/jot"
)

// mockStore implements the Store interface for testing MCP handlers.
type mockStore struct {
	listNotesFn    func(string) ([]*jot.Note, error)
	searchNotesFn  func(string) ([]*jot.Note, error)
	findNoteBySlug func(string) (*jot.Note, error)
	catNoteFn      func(string) (string, error)
	createNoteFn   func(string, string, bool) (string, string, error)
	deleteNoteFn   func(string) error
	encryptNoteFn  func(string) error
	decryptNoteFn  func(string) error
	renameNoteFn   func(string, string) (string, error)
	allTasksFn     func(string, string) ([]jot.Task, error)
	tasksDueFn     func(time.Time, time.Time) ([]jot.Task, error)
	markTaskDoneFn func(string, string) error
	allTagsFn      func() ([]string, error)
}

func (m *mockStore) ListNotes(tag string) ([]*jot.Note, error) {
	if m.listNotesFn != nil {
		return m.listNotesFn(tag)
	}
	return nil, nil
}

func (m *mockStore) SearchNotes(q string) ([]*jot.Note, error) {
	if m.searchNotesFn != nil {
		return m.searchNotesFn(q)
	}
	return nil, nil
}

func (m *mockStore) FindNoteBySlug(slug string) (*jot.Note, error) {
	if m.findNoteBySlug != nil {
		return m.findNoteBySlug(slug)
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockStore) CatNote(slug string) (string, error) {
	if m.catNoteFn != nil {
		return m.catNoteFn(slug)
	}
	return "", nil
}

func (m *mockStore) CreateNote(title, content string, secure bool) (string, string, error) {
	if m.createNoteFn != nil {
		return m.createNoteFn(title, content, secure)
	}
	return "slug", "/path/slug.md", nil
}

func (m *mockStore) DeleteNote(slug string) error {
	if m.deleteNoteFn != nil {
		return m.deleteNoteFn(slug)
	}
	return nil
}

func (m *mockStore) EncryptNote(slug string) error {
	if m.encryptNoteFn != nil {
		return m.encryptNoteFn(slug)
	}
	return nil
}

func (m *mockStore) DecryptNote(slug string) error {
	if m.decryptNoteFn != nil {
		return m.decryptNoteFn(slug)
	}
	return nil
}

func (m *mockStore) RenameNote(oldSlug, newTitle string) (string, error) {
	if m.renameNoteFn != nil {
		return m.renameNoteFn(oldSlug, newTitle)
	}
	return "new-slug", nil
}

func (m *mockStore) AllTasks(status, tag string) ([]jot.Task, error) {
	if m.allTasksFn != nil {
		return m.allTasksFn(status, tag)
	}
	return nil, nil
}

func (m *mockStore) TasksDue(from, to time.Time) ([]jot.Task, error) {
	if m.tasksDueFn != nil {
		return m.tasksDueFn(from, to)
	}
	return nil, nil
}

func (m *mockStore) MarkTaskDone(slug, desc string) error {
	if m.markTaskDoneFn != nil {
		return m.markTaskDoneFn(slug, desc)
	}
	return nil
}

func (m *mockStore) AllTags() ([]string, error) {
	if m.allTagsFn != nil {
		return m.allTagsFn()
	}
	return nil, nil
}

// resultText extracts the text from the first TextContent block.
func resultText(r *mcpsdk.CallToolResult) string {
	if r == nil || len(r.Content) == 0 {
		return ""
	}
	if tc, ok := r.Content[0].(*mcpsdk.TextContent); ok {
		return tc.Text
	}
	return ""
}

// ─── New ─────────────────────────────────────────────────────────────────────

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name:    "nil store returns error",
			cfg:     Config{},
			wantErr: true,
		},
		{
			name:    "valid store succeeds",
			cfg:     Config{Store: &mockStore{}},
			wantErr: false,
		},
		{
			name:    "nil logger uses default",
			cfg:     Config{Store: &mockStore{}, Logger: nil},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv, err := New(tt.cfg)
			if tt.wantErr {
				if err == nil {
					t.Fatal("New: expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("New: unexpected error: %v", err)
			}
			if srv == nil {
				t.Fatal("expected non-nil Server")
			}
		})
	}
}

func TestClose(t *testing.T) {
	t.Parallel()

	srv, err := New(Config{Store: &mockStore{}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := srv.Close(); err != nil {
		t.Errorf("Close: unexpected error: %v", err)
	}
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func TestTextResult(t *testing.T) {
	t.Parallel()

	r := textResult("hello world")
	got := resultText(r)
	if got != "hello world" {
		t.Errorf("textResult = %q, want %q", got, "hello world")
	}
}

func TestJsonOrErr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   any
		wantSub string
		wantErr bool
	}{
		{
			name:    "marshals slice",
			input:   []string{"a", "b"},
			wantSub: `"a"`,
		},
		{
			name:    "marshals struct",
			input:   struct{ Name string }{"jot"},
			wantSub: `"Name"`,
		},
		{
			name:    "unmarshalable returns error",
			input:   make(chan int),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := jsonOrErr(tt.input)
			if tt.wantErr {
				if got == "" || got[:5] != "error" {
					t.Errorf("expected error string, got %q", got)
				}
				return
			}
			if len(got) == 0 {
				t.Fatal("expected non-empty JSON")
			}
			if tt.wantSub != "" && !containsStr(got, tt.wantSub) {
				t.Errorf("jsonOrErr = %q, want substring %q", got, tt.wantSub)
			}
		})
	}
}

func TestJsonMarshalIndent(t *testing.T) {
	t.Parallel()

	data := map[string]int{"count": 42}
	b, err := jsonMarshalIndent(data)
	if err != nil {
		t.Fatalf("jsonMarshalIndent: %v", err)
	}
	got := string(b)
	if !containsStr(got, "  ") {
		t.Error("expected 2-space indentation")
	}
	if !containsStr(got, `"count": 42`) {
		t.Errorf("unexpected output: %s", got)
	}
}

// ─── handlers ────────────────────────────────────────────────────────────────

func newTestServer(store *mockStore) *Server {
	srv, _ := New(Config{Store: store})
	return srv
}

func TestHandleListNotes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		store   *mockStore
		args    listNotesArgs
		wantSub string
	}{
		{
			name: "returns notes as JSON",
			store: &mockStore{
				listNotesFn: func(_ string) ([]*jot.Note, error) {
					return []*jot.Note{{Slug: "my-note", Title: "My Note"}}, nil
				},
			},
			args:    listNotesArgs{},
			wantSub: `"my-note"`,
		},
		{
			name: "passes tag filter",
			store: &mockStore{
				listNotesFn: func(tag string) ([]*jot.Note, error) {
					if tag != "work" {
						return nil, fmt.Errorf("unexpected tag %q", tag)
					}
					return []*jot.Note{{Slug: "work-note"}}, nil
				},
			},
			args:    listNotesArgs{Tag: "work"},
			wantSub: `"work-note"`,
		},
		{
			name: "store error returned as text",
			store: &mockStore{
				listNotesFn: func(_ string) ([]*jot.Note, error) {
					return nil, fmt.Errorf("disk full")
				},
			},
			args:    listNotesArgs{},
			wantSub: "error: disk full",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := newTestServer(tt.store)
			r, _, err := srv.handleListNotes(context.Background(), nil, tt.args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := resultText(r)
			if !containsStr(got, tt.wantSub) {
				t.Errorf("got %q, want substring %q", got, tt.wantSub)
			}
		})
	}
}

func TestHandleGetNote(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		store   *mockStore
		args    getNoteArgs
		wantSub string
	}{
		{
			name:    "empty slug returns error",
			store:   &mockStore{},
			args:    getNoteArgs{Slug: ""},
			wantSub: "error: slug is required",
		},
		{
			name: "returns note body",
			store: &mockStore{
				catNoteFn: func(_ string) (string, error) {
					return "# Hello World\n\nSome content.", nil
				},
			},
			args:    getNoteArgs{Slug: "hello"},
			wantSub: "# Hello World",
		},
		{
			name: "store error returned as text",
			store: &mockStore{
				catNoteFn: func(_ string) (string, error) {
					return "", fmt.Errorf("not found")
				},
			},
			args:    getNoteArgs{Slug: "missing"},
			wantSub: `error: get note "missing"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := newTestServer(tt.store)
			r, _, err := srv.handleGetNote(context.Background(), nil, tt.args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := resultText(r)
			if !containsStr(got, tt.wantSub) {
				t.Errorf("got %q, want substring %q", got, tt.wantSub)
			}
		})
	}
}

func TestHandleCreateNote(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		store   *mockStore
		args    createNoteArgs
		wantSub string
	}{
		{
			name:    "empty title returns error",
			store:   &mockStore{},
			args:    createNoteArgs{Title: ""},
			wantSub: "error: title is required",
		},
		{
			name: "creates note and returns slug and path",
			store: &mockStore{
				createNoteFn: func(_, _ string, _ bool) (string, string, error) {
					return "2026-05-17-my-note", "/notes/2026-05-17-my-note.md", nil
				},
			},
			args:    createNoteArgs{Title: "My Note", Content: "body"},
			wantSub: `"slug": "2026-05-17-my-note"`,
		},
		{
			name: "secure flag is reflected",
			store: &mockStore{
				createNoteFn: func(_, _ string, secure bool) (string, string, error) {
					if !secure {
						return "", "", fmt.Errorf("expected secure=true")
					}
					return "s", "/s.md", nil
				},
			},
			args:    createNoteArgs{Title: "Secret", Secure: true},
			wantSub: `"secure": true`,
		},
		{
			name: "store error returned as text",
			store: &mockStore{
				createNoteFn: func(_, _ string, _ bool) (string, string, error) {
					return "", "", fmt.Errorf("permission denied")
				},
			},
			args:    createNoteArgs{Title: "Fail"},
			wantSub: "error: create note",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := newTestServer(tt.store)
			r, _, err := srv.handleCreateNote(context.Background(), nil, tt.args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := resultText(r)
			if !containsStr(got, tt.wantSub) {
				t.Errorf("got %q, want substring %q", got, tt.wantSub)
			}
		})
	}
}

func TestHandleDeleteNote(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		store   *mockStore
		args    deleteNoteArgs
		wantSub string
	}{
		{
			name:    "empty slug returns error",
			store:   &mockStore{},
			args:    deleteNoteArgs{Slug: ""},
			wantSub: "error: slug is required",
		},
		{
			name:    "deletes note",
			store:   &mockStore{},
			args:    deleteNoteArgs{Slug: "my-note"},
			wantSub: `"deleted": "my-note"`,
		},
		{
			name: "store error returned as text",
			store: &mockStore{
				deleteNoteFn: func(_ string) error {
					return fmt.Errorf("not found")
				},
			},
			args:    deleteNoteArgs{Slug: "missing"},
			wantSub: `error: delete note "missing"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := newTestServer(tt.store)
			r, _, err := srv.handleDeleteNote(context.Background(), nil, tt.args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := resultText(r)
			if !containsStr(got, tt.wantSub) {
				t.Errorf("got %q, want substring %q", got, tt.wantSub)
			}
		})
	}
}

func TestHandleSearchNotes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		store   *mockStore
		args    searchNotesArgs
		wantSub string
	}{
		{
			name:    "empty query returns error",
			store:   &mockStore{},
			args:    searchNotesArgs{Query: ""},
			wantSub: "error: query is required",
		},
		{
			name: "returns matching notes",
			store: &mockStore{
				searchNotesFn: func(_ string) ([]*jot.Note, error) {
					return []*jot.Note{{Slug: "result"}}, nil
				},
			},
			args:    searchNotesArgs{Query: "test"},
			wantSub: `"result"`,
		},
		{
			name: "store error returned as text",
			store: &mockStore{
				searchNotesFn: func(_ string) ([]*jot.Note, error) {
					return nil, fmt.Errorf("walk failed")
				},
			},
			args:    searchNotesArgs{Query: "x"},
			wantSub: "error: walk failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := newTestServer(tt.store)
			r, _, err := srv.handleSearchNotes(context.Background(), nil, tt.args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := resultText(r)
			if !containsStr(got, tt.wantSub) {
				t.Errorf("got %q, want substring %q", got, tt.wantSub)
			}
		})
	}
}

func TestHandleRenameNote(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		store   *mockStore
		args    renameNoteArgs
		wantSub string
	}{
		{
			name:    "empty slug returns error",
			store:   &mockStore{},
			args:    renameNoteArgs{Slug: "", Title: "New"},
			wantSub: "error: slug is required",
		},
		{
			name:    "empty title returns error",
			store:   &mockStore{},
			args:    renameNoteArgs{Slug: "old", Title: ""},
			wantSub: "error: title is required",
		},
		{
			name: "renames note",
			store: &mockStore{
				renameNoteFn: func(_, _ string) (string, error) {
					return "2026-05-17-new-title", nil
				},
			},
			args:    renameNoteArgs{Slug: "old-slug", Title: "New Title"},
			wantSub: `"new_slug": "2026-05-17-new-title"`,
		},
		{
			name: "store error returned as text",
			store: &mockStore{
				renameNoteFn: func(_, _ string) (string, error) {
					return "", fmt.Errorf("file not found")
				},
			},
			args:    renameNoteArgs{Slug: "old", Title: "New"},
			wantSub: "error: rename note",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := newTestServer(tt.store)
			r, _, err := srv.handleRenameNote(context.Background(), nil, tt.args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := resultText(r)
			if !containsStr(got, tt.wantSub) {
				t.Errorf("got %q, want substring %q", got, tt.wantSub)
			}
		})
	}
}

func TestHandleEncryptNote(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		store   *mockStore
		args    encryptNoteArgs
		wantSub string
	}{
		{
			name:    "empty slug returns error",
			store:   &mockStore{},
			args:    encryptNoteArgs{Slug: ""},
			wantSub: "error: slug is required",
		},
		{
			name:    "encrypts note",
			store:   &mockStore{},
			args:    encryptNoteArgs{Slug: "secret"},
			wantSub: `"encrypted": "secret"`,
		},
		{
			name: "store error returned as text",
			store: &mockStore{
				encryptNoteFn: func(_ string) error {
					return fmt.Errorf("no SSH keys")
				},
			},
			args:    encryptNoteArgs{Slug: "fail"},
			wantSub: `error: encrypt note "fail"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := newTestServer(tt.store)
			r, _, err := srv.handleEncryptNote(context.Background(), nil, tt.args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := resultText(r)
			if !containsStr(got, tt.wantSub) {
				t.Errorf("got %q, want substring %q", got, tt.wantSub)
			}
		})
	}
}

func TestHandleDecryptNote(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		store   *mockStore
		args    decryptNoteArgs
		wantSub string
	}{
		{
			name:    "empty slug returns error",
			store:   &mockStore{},
			args:    decryptNoteArgs{Slug: ""},
			wantSub: "error: slug is required",
		},
		{
			name:    "decrypts note",
			store:   &mockStore{},
			args:    decryptNoteArgs{Slug: "secret"},
			wantSub: `"decrypted": "secret"`,
		},
		{
			name: "store error returned as text",
			store: &mockStore{
				decryptNoteFn: func(_ string) error {
					return fmt.Errorf("bad key")
				},
			},
			args:    decryptNoteArgs{Slug: "fail"},
			wantSub: `error: decrypt note "fail"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := newTestServer(tt.store)
			r, _, err := srv.handleDecryptNote(context.Background(), nil, tt.args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := resultText(r)
			if !containsStr(got, tt.wantSub) {
				t.Errorf("got %q, want substring %q", got, tt.wantSub)
			}
		})
	}
}

func TestHandleListTasks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		store   *mockStore
		args    listTasksArgs
		wantSub string
	}{
		{
			name: "returns tasks as JSON",
			store: &mockStore{
				allTasksFn: func(status, _ string) ([]jot.Task, error) {
					if status != "all" {
						return nil, fmt.Errorf("expected status=all, got %q", status)
					}
					return []jot.Task{{Description: "do thing"}}, nil
				},
			},
			args:    listTasksArgs{},
			wantSub: `"do thing"`,
		},
		{
			name: "defaults empty status to all",
			store: &mockStore{
				allTasksFn: func(status, _ string) ([]jot.Task, error) {
					if status != "all" {
						return nil, fmt.Errorf("expected all, got %q", status)
					}
					return nil, nil
				},
			},
			args:    listTasksArgs{Status: ""},
			wantSub: "null",
		},
		{
			name: "passes status and tag",
			store: &mockStore{
				allTasksFn: func(status, tagFilter string) ([]jot.Task, error) {
					if status != "open" || tagFilter != "work" {
						return nil, fmt.Errorf("unexpected: status=%q tag=%q", status, tagFilter)
					}
					return []jot.Task{{Description: "task"}}, nil
				},
			},
			args:    listTasksArgs{Status: "open", Tag: "work"},
			wantSub: `"task"`,
		},
		{
			name: "store error returned as text",
			store: &mockStore{
				allTasksFn: func(_, _ string) ([]jot.Task, error) {
					return nil, fmt.Errorf("broken")
				},
			},
			args:    listTasksArgs{},
			wantSub: "error: broken",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := newTestServer(tt.store)
			r, _, err := srv.handleListTasks(context.Background(), nil, tt.args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := resultText(r)
			if !containsStr(got, tt.wantSub) {
				t.Errorf("got %q, want substring %q", got, tt.wantSub)
			}
		})
	}
}

func TestHandleTaskDone(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		store   *mockStore
		args    taskDoneArgs
		wantSub string
	}{
		{
			name:    "empty slug returns error",
			store:   &mockStore{},
			args:    taskDoneArgs{Slug: "", Desc: "task"},
			wantSub: "error: slug is required",
		},
		{
			name:    "empty desc returns error",
			store:   &mockStore{},
			args:    taskDoneArgs{Slug: "note", Desc: ""},
			wantSub: "error: desc is required",
		},
		{
			name:    "marks task done",
			store:   &mockStore{},
			args:    taskDoneArgs{Slug: "note", Desc: "review PR"},
			wantSub: `"done": true`,
		},
		{
			name: "store error returned as text",
			store: &mockStore{
				markTaskDoneFn: func(_, _ string) error {
					return fmt.Errorf("no matching task")
				},
			},
			args:    taskDoneArgs{Slug: "note", Desc: "missing"},
			wantSub: "error: mark task done",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := newTestServer(tt.store)
			r, _, err := srv.handleTaskDone(context.Background(), nil, tt.args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := resultText(r)
			if !containsStr(got, tt.wantSub) {
				t.Errorf("got %q, want substring %q", got, tt.wantSub)
			}
		})
	}
}

func TestHandleTasksDue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		store   *mockStore
		args    tasksDueArgs
		wantSub string
	}{
		{
			name: "today period",
			store: &mockStore{
				tasksDueFn: func(from, to time.Time) ([]jot.Task, error) {
					if !from.Equal(to) {
						return nil, fmt.Errorf("expected from==to for today")
					}
					return []jot.Task{{Description: "due today"}}, nil
				},
			},
			args:    tasksDueArgs{Period: "today"},
			wantSub: "due today",
		},
		{
			name: "this_week period",
			store: &mockStore{
				tasksDueFn: func(from, to time.Time) ([]jot.Task, error) {
					diff := to.Sub(from)
					if diff.Hours() < 120 {
						return nil, fmt.Errorf("expected ~6 day range, got %v", diff)
					}
					return []jot.Task{{Description: "weekly"}}, nil
				},
			},
			args:    tasksDueArgs{Period: "this_week"},
			wantSub: "weekly",
		},
		{
			name: "this_month period",
			store: &mockStore{
				tasksDueFn: func(_, _ time.Time) ([]jot.Task, error) {
					return []jot.Task{{Description: "monthly"}}, nil
				},
			},
			args:    tasksDueArgs{Period: "this_month"},
			wantSub: "monthly",
		},
		{
			name:    "invalid period returns error",
			store:   &mockStore{},
			args:    tasksDueArgs{Period: "invalid"},
			wantSub: "error: period must be today, this_week, or this_month",
		},
		{
			name: "store error returned as text",
			store: &mockStore{
				tasksDueFn: func(_, _ time.Time) ([]jot.Task, error) {
					return nil, fmt.Errorf("broken")
				},
			},
			args:    tasksDueArgs{Period: "today"},
			wantSub: "error: broken",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := newTestServer(tt.store)
			r, _, err := srv.handleTasksDue(context.Background(), nil, tt.args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := resultText(r)
			if !containsStr(got, tt.wantSub) {
				t.Errorf("got %q, want substring %q", got, tt.wantSub)
			}
		})
	}
}

func TestHandleListTags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		store   *mockStore
		wantSub string
	}{
		{
			name: "returns tags as JSON",
			store: &mockStore{
				allTagsFn: func() ([]string, error) {
					return []string{"work", "personal"}, nil
				},
			},
			wantSub: `"work"`,
		},
		{
			name: "empty tags",
			store: &mockStore{
				allTagsFn: func() ([]string, error) {
					return nil, nil
				},
			},
			wantSub: "null",
		},
		{
			name: "store error returned as text",
			store: &mockStore{
				allTagsFn: func() ([]string, error) {
					return nil, fmt.Errorf("io error")
				},
			},
			wantSub: "error: io error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := newTestServer(tt.store)
			r, _, err := srv.handleListTags(context.Background(), nil, listTagsArgs{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := resultText(r)
			if !containsStr(got, tt.wantSub) {
				t.Errorf("got %q, want substring %q", got, tt.wantSub)
			}
		})
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (sub == "" || findSubstr(s, sub))
}

func findSubstr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
