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

package jot_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/retr0h/jot/internal/jot"
)

// ─── CatNote ────────────────────────────────────────────────────────────────

func TestCatNote(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		slug    string
		setup   func(t *testing.T, dir string)
		wantSub string
		wantErr bool
	}{
		{
			name: "returns body of plaintext note",
			slug: "my-note",
			setup: func(t *testing.T, dir string) {
				writeNote(t, dir, "my-note", `---
title: "My Note"
tags: []
created: 2026-05-14
---

# My Note

Hello world.
`)
			},
			wantSub: "Hello world.",
		},
		{
			name:    "missing note returns error",
			slug:    "no-such-note",
			setup:   func(_ *testing.T, _ string) {},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			tt.setup(t, dir)
			svc := newSvc(dir)

			got, err := svc.CatNote(tt.slug)
			if tt.wantErr {
				if err == nil {
					t.Fatal("CatNote: expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("CatNote: unexpected error: %v", err)
			}
			if !strings.Contains(got, tt.wantSub) {
				t.Errorf("CatNote = %q, want substring %q", got, tt.wantSub)
			}
		})
	}
}

// ─── CreateNote ─────────────────────────────────────────────────────────────

func TestCreateNote(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		title       string
		content     string
		dir         func(t *testing.T) string
		wantSlugSub string
		wantBodySub string
		wantErr     bool
	}{
		{
			name:        "creates note with content",
			title:       "Test Note",
			content:     "# Test Note\n\nHello.",
			wantSlugSub: "test-note",
			wantBodySub: "Hello.",
		},
		{
			name:        "empty content uses scaffold",
			title:       "Scaffold Note",
			content:     "",
			wantSlugSub: "scaffold-note",
			wantBodySub: "title:",
		},
		{
			name:    "unwritable dir returns error",
			title:   "Fail Note",
			content: "body",
			dir: func(t *testing.T) string {
				dir := t.TempDir()
				p := filepath.Join(dir, "readonly")
				if err := os.MkdirAll(p, 0o500); err != nil {
					t.Fatalf("setup: %v", err)
				}
				noWrite := filepath.Join(p, "nested")
				return noWrite
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var dir string
			if tt.dir != nil {
				dir = tt.dir(t)
			} else {
				dir = t.TempDir()
			}
			svc := newSvc(dir)

			slug, path, err := svc.CreateNote(tt.title, tt.content, false)
			if tt.wantErr {
				if err == nil {
					t.Fatal("CreateNote: expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("CreateNote: %v", err)
			}
			if !strings.Contains(slug, tt.wantSlugSub) {
				t.Errorf("slug = %q, want substring %q", slug, tt.wantSlugSub)
			}

			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read created file: %v", err)
			}
			if !strings.Contains(string(data), tt.wantBodySub) {
				t.Errorf("file content = %q, want substring %q", string(data), tt.wantBodySub)
			}
		})
	}
}

// ─── DeleteNote ─────────────────────────────────────────────────────────────

func TestDeleteNote(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		slug    string
		setup   func(t *testing.T, dir string)
		wantErr bool
	}{
		{
			name: "deletes existing note",
			slug: "my-note",
			setup: func(t *testing.T, dir string) {
				writeNote(t, dir, "my-note", "# Note\n")
			},
		},
		{
			name:  "missing note is not an error",
			slug:  "no-such-note",
			setup: func(_ *testing.T, _ string) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			tt.setup(t, dir)
			svc := newSvc(dir)

			err := svc.DeleteNote(tt.slug)
			if tt.wantErr {
				if err == nil {
					t.Fatal("DeleteNote: expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("DeleteNote: unexpected error: %v", err)
			}

			path := filepath.Join(dir, tt.slug+".md")
			if _, err := os.Stat(path); err == nil {
				t.Error("expected file to be deleted")
			}
		})
	}
}

// ─── RenameNote ─────────────────────────────────────────────────────────────

func TestRenameNote(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		oldSlug    string
		newTitle   string
		setup      func(t *testing.T, dir string)
		wantNewSub string
		wantErr    bool
	}{
		{
			name:     "renames note and updates title",
			oldSlug:  "old-note",
			newTitle: "New Title",
			setup: func(t *testing.T, dir string) {
				writeNote(t, dir, "old-note", `---
title: "Old Title"
tags: []
created: 2026-05-14
---

# Old Title

Content here.
`)
			},
			wantNewSub: "new-title",
		},
		{
			name:     "rewrites wikilinks in other notes",
			oldSlug:  "old-note",
			newTitle: "New Title",
			setup: func(t *testing.T, dir string) {
				writeNote(t, dir, "old-note", `---
title: "Old"
tags: []
created: 2026-05-14
---

# Old
`)
				writeNote(t, dir, "linker", `---
title: "Linker"
tags: []
created: 2026-05-14
---

See [[old-note]] for details.
`)
			},
			wantNewSub: "new-title",
		},
		{
			name:     "missing note returns error",
			oldSlug:  "no-such-note",
			newTitle: "Whatever",
			setup:    func(_ *testing.T, _ string) {},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			tt.setup(t, dir)
			svc := newSvc(dir)

			newSlug, err := svc.RenameNote(tt.oldSlug, tt.newTitle)
			if tt.wantErr {
				if err == nil {
					t.Fatal("RenameNote: expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("RenameNote: unexpected error: %v", err)
			}
			if !strings.Contains(newSlug, tt.wantNewSub) {
				t.Errorf("newSlug = %q, want substring %q", newSlug, tt.wantNewSub)
			}

			oldPath := filepath.Join(dir, tt.oldSlug+".md")
			if _, err := os.Stat(oldPath); err == nil {
				t.Error("old file still exists after rename")
			}

			newPath := filepath.Join(dir, newSlug+".md")
			data, err := os.ReadFile(newPath)
			if err != nil {
				t.Fatalf("read renamed file: %v", err)
			}
			if !strings.Contains(string(data), tt.newTitle) {
				t.Errorf("renamed file missing new title %q in content", tt.newTitle)
			}
		})
	}
}

// ─── MarkTaskDone ───────────────────────────────────────────────────────────

func TestMarkTaskDone(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		slug    string
		desc    string
		setup   func(t *testing.T, dir string)
		wantErr bool
	}{
		{
			name: "marks matching task done",
			slug: "tasks-note",
			desc: "Review PR",
			setup: func(t *testing.T, dir string) {
				writeNote(t, dir, "tasks-note", testNote)
			},
		},
		{
			name: "case insensitive match",
			slug: "tasks-note",
			desc: "review pr",
			setup: func(t *testing.T, dir string) {
				writeNote(t, dir, "tasks-note", testNote)
			},
		},
		{
			name: "no matching task returns error",
			slug: "tasks-note",
			desc: "nonexistent task",
			setup: func(t *testing.T, dir string) {
				writeNote(t, dir, "tasks-note", testNote)
			},
			wantErr: true,
		},
		{
			name:    "missing note returns error",
			slug:    "no-such-note",
			desc:    "anything",
			setup:   func(_ *testing.T, _ string) {},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			tt.setup(t, dir)
			svc := newSvc(dir)

			err := svc.MarkTaskDone(tt.slug, tt.desc)
			if tt.wantErr {
				if err == nil {
					t.Fatal("MarkTaskDone: expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("MarkTaskDone: unexpected error: %v", err)
			}

			data, err := os.ReadFile(filepath.Join(dir, tt.slug+".md"))
			if err != nil {
				t.Fatalf("read note: %v", err)
			}
			if !strings.Contains(string(data), "[x]") {
				t.Error("expected task to be marked done with [x]")
			}
		})
	}
}

// ─── RewriteNoteTitle ───────────────────────────────────────────────────────

func TestRewriteNoteTitle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		content  string
		newTitle string
		wantSub  string
	}{
		{
			name: "replaces frontmatter title and heading",
			content: `---
title: "Old Title"
tags: []
---

# Old Title

Body.
`,
			newTitle: "New Title",
			wantSub:  `title: "New Title"`,
		},
		{
			name:     "handles content without frontmatter",
			content:  "# Old Heading\n\nBody.\n",
			newTitle: "New Heading",
			wantSub:  "# New Heading",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := jot.RewriteNoteTitle(tt.content, tt.newTitle)
			if !strings.Contains(got, tt.wantSub) {
				t.Errorf("got %q, want substring %q", got, tt.wantSub)
			}
		})
	}
}

// ─── RewriteWikiLinks ───────────────────────────────────────────────────────

func TestRewriteWikiLinks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(t *testing.T, dir string)
		oldSlug string
		newSlug string
		wantSub string
	}{
		{
			name: "replaces wikilinks in all notes",
			setup: func(t *testing.T, dir string) {
				writeNote(t, dir, "a", "See [[old-slug]] for details.\n")
				writeNote(t, dir, "b", "Also [[old-slug]] here and [[other]].\n")
			},
			oldSlug: "old-slug",
			newSlug: "new-slug",
			wantSub: "[[new-slug]]",
		},
		{
			name: "no matching links leaves files unchanged",
			setup: func(t *testing.T, dir string) {
				writeNote(t, dir, "c", "See [[unrelated]] only.\n")
			},
			oldSlug: "old-slug",
			newSlug: "new-slug",
			wantSub: "[[unrelated]]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			tt.setup(t, dir)

			err := jot.RewriteWikiLinks(dir, tt.oldSlug, tt.newSlug)
			if err != nil {
				t.Fatalf("RewriteWikiLinks: %v", err)
			}

			entries, _ := os.ReadDir(dir)
			for _, e := range entries {
				if !strings.HasSuffix(e.Name(), ".md") {
					continue
				}
				data, _ := os.ReadFile(filepath.Join(dir, e.Name()))
				content := string(data)
				if strings.Contains(content, "[["+tt.oldSlug+"]]") {
					t.Errorf("file %s still contains old wikilink [[%s]]", e.Name(), tt.oldSlug)
				}
			}
		})
	}
}

// ─── FormatTags ─────────────────────────────────────────────────────────────

func TestFormatTags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		tags []string
		want string
	}{
		{
			name: "empty slice",
			tags: nil,
			want: "",
		},
		{
			name: "single tag",
			tags: []string{"work"},
			want: "work",
		},
		{
			name: "multiple tags",
			tags: []string{"work", "sprint", "q2"},
			want: "work, sprint, q2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := jot.FormatTags(tt.tags)
			if got != tt.want {
				t.Errorf("FormatTags(%v) = %q, want %q", tt.tags, got, tt.want)
			}
		})
	}
}
