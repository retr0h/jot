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
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/retr0h/jot/internal/jot"
)

const testNote = `---
title: "Test Note"
tags: [work, sprint]
created: 2026-05-14
---

# Test Note

See [[other-note]] for details.

- [ ] Review PR | due:2026-05-20 #code-review
- [x] Write docs | due:2026-05-22 #docs
- [ ] Deploy staging #ops
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

// newSvc returns a Service rooted at dir with no encryption config.
func newSvc(dir string) *jot.Service {
	return &jot.Service{NotesDir: dir}
}

// symlink creates link pointing at target.
func symlink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}
}

// newSecureSvc returns a Service whose notes dir holds a kvlt vault keyed to
// a throwaway ed25519 SSH key, so secure-note paths run without real keys.
func newSecureSvc(t *testing.T, dir string) *jot.Service {
	t.Helper()

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	block, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	keyPath := filepath.Join(t.TempDir(), "id_ed25519")
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(block), 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatalf("public key: %v", err)
	}
	recipient := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPub)))
	if err := jot.InitVault(dir, []string{recipient}); err != nil {
		t.Fatalf("init vault: %v", err)
	}

	return &jot.Service{NotesDir: dir, SSHKeys: []string{keyPath}}
}

// ─── ReadNoteByPath ───────────────────────────────────────────────────────────

func TestReadNoteByPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		slug      string
		content   string
		setup     func(t *testing.T, dir string) string // returns path
		wantSlug  string
		wantTitle string
		wantDesc  string
		wantTags  []string
		wantTasks int
		wantLine  int // file line of the first task, when set
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
			wantSlug:  "test-note",
			wantTitle: "Test Note",
			wantTags:  []string{"work", "sprint"},
			wantTasks: 3,
			wantLine:  11, // counted from the top of the file, front-matter included
			wantLinks: []string{"other-note"},
		},
		{
			name: "note in a folder keeps the folder in its slug",
			setup: func(t *testing.T, dir string) string {
				if err := os.MkdirAll(filepath.Join(dir, "done"), 0o700); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				return writeNote(t, dir, "done/2026-11-02", "---\ntitle: \"Done\"\n"+
					"description: \"What closed on 11/02\"\ntags: []\n---\n\n- [x] shipped | done:2026-11-02\n")
			},
			wantSlug:  "done/2026-11-02",
			wantTitle: "Done",
			wantDesc:  "What closed on 11/02",
			wantTasks: 1,
			wantLine:  7,
		},
		{
			name: "note outside the notes dir falls back to its file name",
			setup: func(t *testing.T, _ string) string {
				return writeNote(t, t.TempDir(), "elsewhere", "# Elsewhere\n")
			},
			wantSlug:  "elsewhere",
			wantTitle: "Elsewhere",
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
			svc := newSvc(dir)

			got, err := svc.ReadNoteByPath(path)
			if tt.wantErr {
				if err == nil {
					t.Fatal("ReadNoteByPath: expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ReadNoteByPath: unexpected error: %v", err)
			}

			if got.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", got.Title, tt.wantTitle)
			}
			if tt.wantSlug != "" && got.Slug != tt.wantSlug {
				t.Errorf("Slug = %q, want %q", got.Slug, tt.wantSlug)
			}
			if got.Description != tt.wantDesc {
				t.Errorf("Description = %q, want %q", got.Description, tt.wantDesc)
			}
			if got.Updated == "" {
				t.Error("Updated is empty")
			}
			if tt.wantLine != 0 && got.Tasks[0].Line != tt.wantLine {
				t.Errorf("first task Line = %d, want %d", got.Tasks[0].Line, tt.wantLine)
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
			svc := newSvc(dir)

			got, err := svc.ReadAllNotes()
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

// ─── matchStatus ────────────────────────────────────────────────────────────

func TestMatchStatus(t *testing.T) {
	t.Parallel()

	setupDir := func(t *testing.T) string {
		t.Helper()
		dir := t.TempDir()
		writeNote(t, dir, "mixed", `---
title: "Mixed"
tags: []
created: 2026-05-14
---

- [ ] open task #work
- [x] done task #work
`)
		return dir
	}

	tests := []struct {
		name      string
		status    string
		wantCount int
	}{
		{name: "empty status means all", status: "", wantCount: 2},
		{name: "all returns everything", status: "all", wantCount: 2},
		{name: "open returns open only", status: "open", wantCount: 1},
		{name: "done returns done only", status: "done", wantCount: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := setupDir(t)
			svc := newSvc(dir)

			got, err := svc.AllTasks(tt.status, "", "")
			if err != nil {
				t.Fatalf("AllTasks: %v", err)
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

	// Build a shared dir: two notes at the root, tagged "work" and "personal",
	// and a small folder tree under work/.
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
		if err := os.MkdirAll(filepath.Join(dir, "work", "infra"), 0o700); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		writeNote(t, dir, "work/sensor-ideas", "---\ntags: [work]\n---\n# Sensor\n")
		writeNote(t, dir, "work/infra/deploy", "# Deploy\n")
		return dir
	}

	tests := []struct {
		name      string
		tagFilter string
		dir       string
		want      []string // slugs, sorted
		wantErr   bool
	}{
		{
			name: "no filter returns all notes",
			want: []string{"personal-note", "work-note", "work/infra/deploy", "work/sensor-ideas"},
		},
		{
			name:      "tag filter matches subset",
			tagFilter: "work",
			want:      []string{"work-note", "work/sensor-ideas"},
		},
		{
			name:      "tag filter with no matches returns empty",
			tagFilter: "nonexistent",
		},
		{
			name: "root lists every note",
			dir:  "/",
			want: []string{"personal-note", "work-note", "work/infra/deploy", "work/sensor-ideas"},
		},
		{
			name: "a folder lists every note under it",
			dir:  "/work",
			want: []string{"work/infra/deploy", "work/sensor-ideas"},
		},
		{
			name: "a folder without the leading slash, with a trailing one",
			dir:  "work/infra/",
			want: []string{"work/infra/deploy"},
		},
		{
			name: "a note names just itself",
			dir:  "/work/sensor-ideas",
			want: []string{"work/sensor-ideas"},
		},
		{
			name: "a folder name does not match a note sharing its prefix",
			dir:  "/work-",
		},
		{
			name: "a glob stays within one folder",
			dir:  "/work/*",
			want: []string{"work/infra/deploy", "work/sensor-ideas"},
		},
		{
			name: "a glob on names",
			dir:  "/*-note",
			want: []string{"personal-note", "work-note"},
		},
		{
			name: "a glob matching a folder lists under it",
			dir:  "/wo?k",
			want: []string{"work/infra/deploy", "work/sensor-ideas"},
		},
		{
			name:      "path and tag filters combine",
			tagFilter: "work",
			dir:       "/work",
			want:      []string{"work/sensor-ideas"},
		},
		{
			name: "dot-dot cannot climb above the root",
			dir:  "/../work/infra",
			want: []string{"work/infra/deploy"},
		},
		{
			name: "no such folder returns empty",
			dir:  "/nope",
		},
		{
			name:    "a malformed glob returns error",
			dir:     "/work/[",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := setupDir(t)
			svc := newSvc(dir)

			got, err := svc.ListNotes(tt.tagFilter, tt.dir)
			if tt.wantErr {
				if err == nil {
					t.Fatal("ListNotes: expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ListNotes: unexpected error: %v", err)
			}
			slugs := make([]string, 0, len(got))
			for _, n := range got {
				slugs = append(slugs, n.Slug)
			}
			slices.Sort(slugs)
			if !slices.Equal(slugs, tt.want) {
				t.Errorf("slugs = %v, want %v", slugs, tt.want)
			}
		})
	}
}

// ─── DirEntries ──────────────────────────────────────────────────────────────

func TestDirEntries(t *testing.T) {
	t.Parallel()

	slugs := []string{
		"personal",
		"work",
		"done/2026-11-09",
		"done/2026-11-10",
		"work/sensor-ideas",
		"work/infra/deploy",
		"work/infra/k8s/rbac",
	}
	notes := make([]*jot.Note, 0, len(slugs))
	for _, s := range slugs {
		notes = append(notes, &jot.Note{Slug: s})
	}

	// render writes an entry as "dir/ N" or the note's slug.
	render := func(entries []jot.Entry) []string {
		got := make([]string, 0, len(entries))
		for _, e := range entries {
			if e.Note != nil {
				got = append(got, e.Note.Slug)
				continue
			}
			got = append(got, fmt.Sprintf("%s/ %d", e.Dir, e.Count))
		}
		return got
	}

	tests := []struct {
		name    string
		notes   []*jot.Note
		dir     string
		want    []string
		wantErr bool
	}{
		{
			name:  "the root lists its folders, sorted with counts, then its notes",
			notes: notes,
			want:  []string{"done/ 2", "work/ 3", "personal", "work"},
		},
		{
			name:  "slash is the root",
			notes: notes,
			dir:   "/",
			want:  []string{"done/ 2", "work/ 3", "personal", "work"},
		},
		{
			name:  "a folder lists one level down, beside a note of the same name",
			notes: notes,
			dir:   "/work",
			want:  []string{"work/infra/ 2", "work", "work/sensor-ideas"},
		},
		{
			name:  "a nested folder, without the leading slash, with a trailing one",
			notes: notes,
			dir:   "work/infra/",
			want:  []string{"work/infra/k8s/ 1", "work/infra/deploy"},
		},
		{
			name:  "a note lists just itself",
			notes: notes,
			dir:   "/work/sensor-ideas",
			want:  []string{"work/sensor-ideas"},
		},
		{
			name:  "a glob lists matching notes and what matching folders hold",
			notes: notes,
			dir:   "/work/*",
			want:  []string{"work/infra/k8s/ 1", "work/sensor-ideas", "work/infra/deploy"},
		},
		{
			name:  "a glob at the root",
			notes: notes,
			dir:   "/d*",
			want:  []string{"done/2026-11-09", "done/2026-11-10"},
		},
		{
			name:  "no such folder lists nothing",
			notes: notes,
			dir:   "/nope",
			want:  []string{},
		},
		{
			name: "no notes lists nothing",
			want: []string{},
		},
		{
			name:    "a malformed glob returns error",
			notes:   notes,
			dir:     "/work/[",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			entries, err := jot.DirEntries(tt.notes, tt.dir)
			if tt.wantErr {
				if err == nil {
					t.Fatal("DirEntries: expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("DirEntries: unexpected error: %v", err)
			}
			if got := render(entries); !slices.Equal(got, tt.want) {
				t.Errorf("DirEntries(%q) = %v, want %v", tt.dir, got, tt.want)
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
		dir       string
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
		{
			name:      "a path narrows the search",
			query:     "deployment",
			dir:       "/beta",
			wantCount: 0,
		},
		{
			name:      "a glob path keeps its matches",
			query:     "deployment",
			dir:       "/al*",
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := setupDir(t)
			svc := newSvc(dir)

			got, err := svc.SearchNotes(tt.query, tt.dir)
			if err != nil {
				t.Fatalf("SearchNotes: unexpected error: %v", err)
			}
			if len(got) != tt.wantCount {
				t.Errorf("query %q: count = %d, want %d", tt.query, len(got), tt.wantCount)
			}
		})
	}
}

// ─── FindNoteBySlug ──────────────────────────────────────────────────────────

func TestFindNoteBySlug(t *testing.T) {
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
			name: "slug in a folder, given with .md, returns the note",
			slug: "ops/runbook.md",
			setup: func(t *testing.T, dir string) {
				if err := os.MkdirAll(filepath.Join(dir, "ops"), 0o700); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				writeNote(t, dir, "ops/runbook", testNote)
			},
		},
		{
			name:    "missing slug returns error",
			slug:    "no-such-slug",
			setup:   func(_ *testing.T, _ string) {},
			wantErr: true,
		},
		{
			name:    "slug escaping the notes dir returns error",
			slug:    "../../etc/passwd",
			setup:   func(_ *testing.T, _ string) {},
			wantErr: true,
		},
		{
			name: "a folder symlinked outside the notes dir is refused",
			slug: "escape/secret",
			setup: func(t *testing.T, dir string) {
				outside := t.TempDir()
				writeNote(t, outside, "secret", testNote)
				symlink(t, outside, filepath.Join(dir, "escape"))
			},
			wantErr: true,
		},
		{
			name: "a note file symlinked outside the notes dir is refused",
			slug: "leak",
			setup: func(t *testing.T, dir string) {
				outside := writeNote(t, t.TempDir(), "secret", testNote)
				symlink(t, outside, filepath.Join(dir, "leak.md"))
			},
			wantErr: true,
		},
		{
			name: "a broken symlink a write would follow is refused",
			slug: "dangling",
			setup: func(t *testing.T, dir string) {
				symlink(t, filepath.Join(t.TempDir(), "gone.md"), filepath.Join(dir, "dangling.md"))
			},
			wantErr: true,
		},
		{
			name: "a symlink that stays inside the notes dir works",
			slug: "alias",
			setup: func(t *testing.T, dir string) {
				target := writeNote(t, dir, "real", testNote)
				symlink(t, target, filepath.Join(dir, "alias.md"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			tt.setup(t, dir)
			svc := newSvc(dir)

			got, err := svc.FindNoteBySlug(tt.slug)
			if tt.wantErr {
				if err == nil {
					t.Fatal("FindNoteBySlug: expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("FindNoteBySlug: unexpected error: %v", err)
			}
			if want := strings.TrimSuffix(tt.slug, ".md"); got.Slug != want {
				t.Errorf("Slug = %q, want %q", got.Slug, want)
			}
		})
	}
}

// ─── CleanSlug ───────────────────────────────────────────────────────────────

func TestCleanSlug(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		slug    string
		want    string
		wantErr bool
	}{
		{name: "plain slug", slug: "todo", want: "todo"},
		{name: "folder slug", slug: "done/2026-11-03", want: "done/2026-11-03"},
		{name: "trailing .md and redundant parts", slug: "./ops//runbook.md", want: "ops/runbook"},
		{name: "dot-dot inside the dir stays inside", slug: "ops/../todo", want: "todo"},
		{name: "empty", slug: "", wantErr: true},
		{name: "blank", slug: "   ", wantErr: true},
		{name: "current dir", slug: ".", wantErr: true},
		{name: "leading slash is the notes root", slug: "/work/ideas", want: "work/ideas"},
		{name: "repeated leading slashes", slug: "//etc/passwd", want: "etc/passwd"},
		{name: "root alone", slug: "/", wantErr: true},
		{name: "leading slash escaping the notes dir", slug: "/../x", wantErr: true},
		{name: "parent dir", slug: "..", wantErr: true},
		{name: "escapes the notes dir", slug: "ops/../../x", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := jot.CleanSlug(tt.slug)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("CleanSlug(%q): expected error, got %q", tt.slug, got)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Errorf("CleanSlug(%q) = %q, %v; want %q", tt.slug, got, err, tt.want)
			}
		})
	}
}

// ─── NoteSlugForPath ─────────────────────────────────────────────────────────

func TestNoteSlugForPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		path   string
		want   string
		wantOK bool
	}{
		{name: "a note's file", path: "done/2026-11-03.md", want: "done/2026-11-03", wantOK: true},
		{
			name:   "a secure note's body in the vault",
			path:   ".kvlt/secrets/local/jot/ops%2Fkeys.age",
			want:   "ops/keys",
			wantOK: true,
		},
		{name: "a vault file that is not a body", path: ".kvlt/secrets/local/jot/.keys.age.tmp"},
		{name: "a vault key that does not unescape", path: ".kvlt/secrets/local/jot/bad%zz.age"},
		{name: "vault config", path: ".kvlt/vaults/local/jot.yaml"},
		{name: "any other file", path: "README"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := jot.NoteSlugForPath(tt.path)
			if ok != tt.wantOK || (ok && got != tt.want) {
				t.Errorf(
					"NoteSlugForPath(%q) = %q, %t; want %q, %t",
					tt.path,
					got,
					ok,
					tt.want,
					tt.wantOK,
				)
			}
		})
	}
}

// ─── Summarize ───────────────────────────────────────────────────────────────

func TestSummarize(t *testing.T) {
	t.Parallel()

	note := &jot.Note{
		Slug:        "ops/runbook",
		Title:       "Runbook",
		Description: "Deploy steps",
		Tags:        []string{"ops"},
		Created:     "2026-05-14",
		Updated:     "2026-11-03T10:00:00-08:00",
		Secure:      true,
		Body:        "intro\nRestart the API pods\n" + strings.Repeat("x", 250) + " needle\n",
		Tasks:       []jot.Task{{Done: false}, {Done: true}, {Done: false}},
	}

	tests := []struct {
		name      string
		query     string
		wantMatch string
	}{
		{name: "no query leaves Match empty"},
		{
			name:      "first matching line, case-insensitive",
			query:     "restart",
			wantMatch: "Restart the API pods",
		},
		{
			name:      "long matching line is capped",
			query:     "needle",
			wantMatch: strings.Repeat("x", 200) + "…",
		},
		{name: "title-only match leaves Match empty", query: "runbook"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			want := jot.NoteSummary{
				Slug:        note.Slug,
				Title:       note.Title,
				Description: note.Description,
				Tags:        note.Tags,
				Created:     note.Created,
				Updated:     note.Updated,
				Secure:      true,
				OpenTasks:   2,
				Match:       tt.wantMatch,
			}
			if got := jot.Summarize(note, tt.query); !reflect.DeepEqual(got, want) {
				t.Errorf("Summarize() = %+v, want %+v", got, want)
			}
		})
	}
}

// ─── IndexByID ───────────────────────────────────────────────────────────────

func TestIndexByID(t *testing.T) {
	t.Parallel()

	tasks := []jot.Task{
		{ID: "t-0001", Description: "write docs", NoteSlug: "a"},
		{ID: "t-0002", Description: "ship it", NoteSlug: "a", After: []string{"t-0001", "t-gone"}},
		{ID: "t-0002", Description: "clashing clone", NoteSlug: "b", Done: true},
		{Description: "no id"},
	}
	byID := jot.IndexByID(tasks)

	if len(byID) != 2 || len(byID["t-0001"]) != 1 || len(byID["t-0002"]) != 2 {
		t.Fatalf("IndexByID = %v, want t-0001 once and t-0002 twice, nothing without an id", byID)
	}
	open := jot.OpenBlockers(jot.RawTask{After: tasks[1].After}, byID)
	if len(open) != 1 || open[0].Description != "write docs" {
		t.Errorf("OpenBlockers = %v, want only the open, existing blocker", open)
	}
}

// ─── DoneSince ───────────────────────────────────────────────────────────────

func TestDoneSince(t *testing.T) {
	t.Parallel()

	tasks := []jot.Task{
		{Description: "old", Done: true, DoneDate: "2026-10-30"},
		{Description: "on the day", Done: true, DoneDate: "2026-11-02"},
		{Description: "later", Done: true, DoneDate: "2026-11-03"},
		{Description: "open", DueDate: "2026-11-05"},
	}

	tests := []struct {
		name  string
		since string
		want  []string
	}{
		{
			name:  "keeps done tasks on or after the date",
			since: "2026-11-02",
			want:  []string{"on the day", "later"},
		},
		{name: "a future date keeps none", since: "2027-01-01", want: []string{}},
		{
			name:  "an empty date keeps every done task",
			since: "",
			want:  []string{"old", "on the day", "later"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := make([]string, 0, len(tasks))
			for _, task := range jot.DoneSince(tasks, tt.since) {
				got = append(got, task.Description)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("DoneSince(%q) = %v, want %v", tt.since, got, tt.want)
			}
		})
	}
}

// ─── AllTasks ────────────────────────────────────────────────────────────────

func TestAllTasks(t *testing.T) {
	t.Parallel()

	// testNote has 3 tasks:
	//   - [ ] Review PR | due:2026-05-20 #code-review   — open
	//   - [x] Write docs | due:2026-05-22 #docs          — done
	//   - [ ] Deploy staging #ops                         — open

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
		dir       string
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
			name:      "a path keeps the tasks of the notes under it",
			status:    "all",
			dir:       "/test-note",
			wantCount: 3,
			setup:     setupDir,
		},
		{
			name:      "a path with no notes under it returns no tasks",
			status:    "all",
			dir:       "/other",
			wantCount: 0,
			setup:     setupDir,
		},
		{
			name:      "empty notes dir returns no tasks",
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
			svc := newSvc(dir)

			got, err := svc.AllTasks(tt.status, tt.tagFilter, tt.dir)
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
				writeNote(t, dir, "no-due", "# No Due\n\n- [ ] simple task\n")
				return dir
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := tt.setup(t)
			svc := newSvc(dir)

			got, err := svc.TasksDue(tt.from, tt.to)
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

Some text #work and - [ ] do thing #work
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
			svc := newSvc(dir)

			got, err := svc.AllTags()
			if err != nil {
				t.Fatalf("AllTags: unexpected error: %v", err)
			}

			slices.Sort(got)

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
