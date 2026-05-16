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
	"sort"
	"testing"
	"time"

	"github.com/retr0h/jot/internal/jot"
)

const testNote = `---
title: "Test Note"
tags: [work, sprint]
created: 2026-05-14
---

# Test Note

See [[other-note]] for details.

@task(Review PR | due:2026-05-20) #code-review
@task(Write docs | due:2026-05-22 | done:2026-05-21) #docs
@task Deploy staging #ops
`

// writeNote is a helper that writes content to slug.md inside dir.
func writeNote(t *testing.T, dir, slug, content string) string {
	t.Helper()
	path := filepath.Join(dir, slug+".md")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writeNote: %v", err)
	}
	return path
}

// ─── ReadNote ────────────────────────────────────────────────────────────────

func TestReadNote(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		slug      string
		content   string
		setup     func(t *testing.T, dir string) string // returns path
		wantTitle string
		wantTags  []string
		wantTasks int
		wantLinks []string
		wantErr   bool
	}{
		{
			name:    "valid note with frontmatter tasks and links",
			slug:    "test-note",
			content: testNote,
			setup: func(t *testing.T, dir string) string {
				return writeNote(t, dir, "test-note", testNote)
			},
			wantTitle: "Test Note",
			wantTags:  []string{"work", "sprint"},
			wantTasks: 3,
			wantLinks: []string{"other-note"},
		},
		{
			name: "note without frontmatter uses heading as title",
			slug: "no-fm",
			setup: func(t *testing.T, dir string) string {
				return writeNote(t, dir, "no-fm", "# My Heading\n\nsome text\n")
			},
			wantTitle: "My Heading",
			wantTags:  nil,
			wantTasks: 0,
			wantLinks: nil,
		},
		{
			name: "empty file falls back to slug for title",
			slug: "empty-note",
			setup: func(t *testing.T, dir string) string {
				return writeNote(t, dir, "empty-note", "")
			},
			wantTitle: "empty-note",
			wantTags:  nil,
			wantTasks: 0,
			wantLinks: nil,
		},
		{
			name: "missing file returns error",
			slug: "no-such-note",
			setup: func(_ *testing.T, dir string) string {
				return filepath.Join(dir, "no-such-note.md")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path := tt.setup(t, dir)

			got, err := jot.ReadNote(path)
			if tt.wantErr {
				if err == nil {
					t.Fatal("ReadNote: expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ReadNote: unexpected error: %v", err)
			}

			if got.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", got.Title, tt.wantTitle)
			}

			if len(got.Tags) != len(tt.wantTags) {
				t.Errorf("Tags = %v, want %v", got.Tags, tt.wantTags)
			} else {
				for i, tag := range tt.wantTags {
					if got.Tags[i] != tag {
						t.Errorf("Tags[%d] = %q, want %q", i, got.Tags[i], tag)
					}
				}
			}

			if len(got.Tasks) != tt.wantTasks {
				t.Errorf("Tasks count = %d, want %d", len(got.Tasks), tt.wantTasks)
			}

			if len(got.Links) != len(tt.wantLinks) {
				t.Errorf("Links = %v, want %v", got.Links, tt.wantLinks)
			} else {
				for i, link := range tt.wantLinks {
					if got.Links[i] != link {
						t.Errorf("Links[%d] = %q, want %q", i, got.Links[i], link)
					}
				}
			}
		})
	}
}

// ─── ReadAllNotes ─────────────────────────────────────────────────────────────

func TestReadAllNotes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setup     func(t *testing.T) string // returns notesDir
		wantCount int
		wantErr   bool
	}{
		{
			name: "dir with multiple md files",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				writeNote(t, dir, "alpha", testNote)
				writeNote(t, dir, "beta", "# Beta\n")
				return dir
			},
			wantCount: 2,
		},
		{
			name: "skips non-md files",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				writeNote(t, dir, "real", "# Real\n")
				if err := os.WriteFile(filepath.Join(dir, "ignore.txt"), []byte("txt"), 0o600); err != nil {
					t.Fatalf("setup: %v", err)
				}
				return dir
			},
			wantCount: 1,
		},
		{
			name: "skips .git directory",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				writeNote(t, dir, "note", "# Note\n")
				gitDir := filepath.Join(dir, ".git")
				if err := os.Mkdir(gitDir, 0o700); err != nil {
					t.Fatalf("setup mkdir .git: %v", err)
				}
				if err := os.WriteFile(filepath.Join(gitDir, "hidden.md"), []byte("# Hidden\n"), 0o600); err != nil {
					t.Fatalf("setup: %v", err)
				}
				return dir
			},
			wantCount: 1,
		},
		{
			name: "empty dir returns empty slice",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			wantCount: 0,
		},
		{
			name: "nonexistent dir returns error",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "does-not-exist")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := tt.setup(t)

			got, err := jot.ReadAllNotes(dir)
			if tt.wantErr {
				if err == nil {
					t.Fatal("ReadAllNotes: expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ReadAllNotes: unexpected error: %v", err)
			}
			if len(got) != tt.wantCount {
				t.Errorf("count = %d, want %d", len(got), tt.wantCount)
			}
		})
	}
}

// ─── ListNotes ───────────────────────────────────────────────────────────────

func TestListNotes(t *testing.T) {
	t.Parallel()

	// Build a shared dir with two notes: one tagged "work", one tagged "personal".
	setupDir := func(t *testing.T) string {
		t.Helper()
		dir := t.TempDir()
		writeNote(t, dir, "work-note", `---
title: "Work Note"
tags: [work]
created: 2026-05-14
---
# Work Note
`)
		writeNote(t, dir, "personal-note", `---
title: "Personal Note"
tags: [personal]
created: 2026-05-14
---
# Personal Note
`)
		return dir
	}

	tests := []struct {
		name      string
		tagFilter string
		wantCount int
	}{
		{
			name:      "no filter returns all notes",
			tagFilter: "",
			wantCount: 2,
		},
		{
			name:      "tag filter matches subset",
			tagFilter: "work",
			wantCount: 1,
		},
		{
			name:      "tag filter with no matches returns empty",
			tagFilter: "nonexistent",
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := setupDir(t)

			got, err := jot.ListNotes(dir, tt.tagFilter)
			if err != nil {
				t.Fatalf("ListNotes: unexpected error: %v", err)
			}
			if len(got) != tt.wantCount {
				t.Errorf("count = %d, want %d", len(got), tt.wantCount)
			}
		})
	}
}

// ─── SearchNotes ─────────────────────────────────────────────────────────────

func TestSearchNotes(t *testing.T) {
	t.Parallel()

	setupDir := func(t *testing.T) string {
		t.Helper()
		dir := t.TempDir()
		writeNote(t, dir, "alpha", `---
title: "Alpha Project"
tags: []
created: 2026-05-14
---
# Alpha Project

This discusses deployment pipelines.
`)
		writeNote(t, dir, "beta", `---
title: "Beta Notes"
tags: []
created: 2026-05-14
---
# Beta Notes

Random thoughts about coffee.
`)
		return dir
	}

	tests := []struct {
		name      string
		query     string
		wantCount int
	}{
		{
			name:      "matches by title",
			query:     "Alpha",
			wantCount: 1,
		},
		{
			name:      "matches by body content",
			query:     "deployment",
			wantCount: 1,
		},
		{
			name:      "case-insensitive match",
			query:     "COFFEE",
			wantCount: 1,
		},
		{
			name:      "no matches returns empty",
			query:     "xyzzy",
			wantCount: 0,
		},
		{
			name:      "empty query matches all notes",
			query:     "",
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := setupDir(t)

			got, err := jot.SearchNotes(dir, tt.query)
			if err != nil {
				t.Fatalf("SearchNotes: unexpected error: %v", err)
			}
			if len(got) != tt.wantCount {
				t.Errorf("query %q: count = %d, want %d", tt.query, len(got), tt.wantCount)
			}
		})
	}
}

// ─── FindNote ────────────────────────────────────────────────────────────────

func TestFindNote(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		slug    string
		setup   func(t *testing.T, dir string)
		wantErr bool
	}{
		{
			name: "existing slug returns note",
			slug: "my-note",
			setup: func(t *testing.T, dir string) {
				writeNote(t, dir, "my-note", testNote)
			},
		},
		{
			name:    "missing slug returns error",
			slug:    "no-such-slug",
			setup:   func(_ *testing.T, _ string) {},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			tt.setup(t, dir)

			got, err := jot.FindNote(dir, tt.slug)
			if tt.wantErr {
				if err == nil {
					t.Fatal("FindNote: expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("FindNote: unexpected error: %v", err)
			}
			if got.Slug != tt.slug {
				t.Errorf("Slug = %q, want %q", got.Slug, tt.slug)
			}
		})
	}
}

// ─── AllTasks ────────────────────────────────────────────────────────────────

func TestAllTasks(t *testing.T) {
	t.Parallel()

	// testNote has 3 tasks:
	//   @task(Review PR | due:2026-05-20) #code-review   — open
	//   @task(Write docs | due:2026-05-22 | done:2026-05-21) #docs — done
	//   @task Deploy staging #ops                         — open

	setupDir := func(t *testing.T) string {
		t.Helper()
		dir := t.TempDir()
		writeNote(t, dir, "test-note", testNote)
		return dir
	}

	tests := []struct {
		name      string
		status    string
		tagFilter string
		wantCount int
		setup     func(t *testing.T) string
	}{
		{
			name:      "status open returns open tasks",
			status:    "open",
			tagFilter: "",
			wantCount: 2,
			setup:     setupDir,
		},
		{
			name:      "status done returns done tasks",
			status:    "done",
			tagFilter: "",
			wantCount: 1,
			setup:     setupDir,
		},
		{
			name:      "status all returns every task",
			status:    "all",
			tagFilter: "",
			wantCount: 3,
			setup:     setupDir,
		},
		{
			name:      "tag filter restricts results",
			status:    "all",
			tagFilter: "docs",
			wantCount: 1,
			setup:     setupDir,
		},
		{
			name:      "empty dir returns no tasks",
			status:    "all",
			tagFilter: "",
			wantCount: 0,
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := tt.setup(t)

			got, err := jot.AllTasks(dir, tt.status, tt.tagFilter)
			if err != nil {
				t.Fatalf("AllTasks: unexpected error: %v", err)
			}
			if len(got) != tt.wantCount {
				t.Errorf("count = %d, want %d", len(got), tt.wantCount)
			}
		})
	}
}

// ─── TasksDue ────────────────────────────────────────────────────────────────

func TestTasksDue(t *testing.T) {
	t.Parallel()

	// testNote open tasks with due dates:
	//   Review PR  due:2026-05-20
	// (Deploy staging has no due date)
	// Write docs is done — excluded.

	setupDir := func(t *testing.T) string {
		t.Helper()
		dir := t.TempDir()
		writeNote(t, dir, "test-note", testNote)
		return dir
	}

	mustDate := func(s string) time.Time {
		d, err := time.Parse("2006-01-02", s)
		if err != nil {
			t.Fatalf("mustDate(%q): %v", s, err)
		}
		return d
	}

	tests := []struct {
		name      string
		from      time.Time
		to        time.Time
		wantCount int
		setup     func(t *testing.T) string
	}{
		{
			name:      "task falls within range",
			from:      mustDate("2026-05-18"),
			to:        mustDate("2026-05-21"),
			wantCount: 1,
			setup:     setupDir,
		},
		{
			name:      "task outside range returns nothing",
			from:      mustDate("2026-05-01"),
			to:        mustDate("2026-05-10"),
			wantCount: 0,
			setup:     setupDir,
		},
		{
			name:      "no due dates on any task",
			from:      mustDate("2026-05-01"),
			to:        mustDate("2026-05-31"),
			wantCount: 0,
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				writeNote(t, dir, "no-due", "# No Due\n\n@task simple task\n")
				return dir
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := tt.setup(t)

			got, err := jot.TasksDue(dir, tt.from, tt.to)
			if err != nil {
				t.Fatalf("TasksDue: unexpected error: %v", err)
			}
			if len(got) != tt.wantCount {
				t.Errorf("count = %d, want %d", len(got), tt.wantCount)
			}
		})
	}
}

// ─── AllTags ─────────────────────────────────────────────────────────────────

func TestAllTags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setup    func(t *testing.T) string
		wantTags []string // sorted; nil means expect empty
	}{
		{
			name: "collects tags from frontmatter inline and tasks",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				writeNote(t, dir, "test-note", testNote)
				return dir
			},
			// frontmatter: work, sprint
			// inline body: (none outside tasks in testNote)
			// task tags: code-review, docs, ops
			wantTags: []string{"code-review", "docs", "ops", "sprint", "work"},
		},
		{
			name: "deduplicates tags appearing in multiple sources",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				writeNote(t, dir, "dup", `---
title: "Dup"
tags: [work]
created: 2026-05-14
---
# Dup

Some text #work and @task do thing #work
`)
				return dir
			},
			wantTags: []string{"work"},
		},
		{
			name: "empty dir returns no tags",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			wantTags: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := tt.setup(t)

			got, err := jot.AllTags(dir)
			if err != nil {
				t.Fatalf("AllTags: unexpected error: %v", err)
			}

			sort.Strings(got)

			if len(got) != len(tt.wantTags) {
				t.Errorf("tags = %v, want %v", got, tt.wantTags)
				return
			}
			for i, tag := range tt.wantTags {
				if got[i] != tag {
					t.Errorf("tags[%d] = %q, want %q", i, got[i], tag)
				}
			}
		})
	}
}
