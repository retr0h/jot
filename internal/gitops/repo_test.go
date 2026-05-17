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

package gitops_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/retr0h/jot/internal/gitops"
)

func writeFile(
	t *testing.T,
	dir string,
	name string,
	content string,
) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writeFile %q: %v", path, err)
	}
	return path
}

func TestInitRepo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(t *testing.T) string
		wantErr bool
	}{
		{
			name: "initializes new repo",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
		},
		{
			name: "idempotent on existing repo",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				if _, err := gitops.InitRepo(dir); err != nil {
					t.Fatalf("pre-init: %v", err)
				}
				return dir
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := tt.setup(t)

			r, err := gitops.InitRepo(dir)
			if tt.wantErr {
				if err == nil {
					t.Fatal("InitRepo: expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("InitRepo: unexpected error: %v", err)
			}
			if r == nil {
				t.Fatal("expected non-nil Repo")
			}
		})
	}
}

func TestRepo_Commit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setup       func(t *testing.T, dir string)
		message     string
		wantCommits int
	}{
		{
			name: "stages and commits new file",
			setup: func(t *testing.T, dir string) {
				writeFile(t, dir, "note.md", "# Hello\n")
			},
			message:     "docs: add note",
			wantCommits: 1,
		},
		{
			name: "no-op on clean worktree",
			setup: func(t *testing.T, dir string) {
				writeFile(t, dir, "note.md", "# Hello\n")
				r, _ := gitops.InitRepo(dir)
				_ = r.Commit("docs: initial")
			},
			message:     "docs: should be skipped",
			wantCommits: 1,
		},
		{
			name: "commits modification to existing file",
			setup: func(t *testing.T, dir string) {
				writeFile(t, dir, "note.md", "# Hello\n")
				r, _ := gitops.InitRepo(dir)
				_ = r.Commit("docs: initial")
				writeFile(t, dir, "note.md", "# Hello World\n")
			},
			message:     "docs: update note",
			wantCommits: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()

			r, err := gitops.InitRepo(dir)
			if err != nil {
				t.Fatalf("InitRepo: %v", err)
			}

			tt.setup(t, dir)

			if err := r.Commit(tt.message); err != nil {
				t.Fatalf("Commit: %v", err)
			}

			entries, err := r.Log("", 0)
			if err != nil {
				t.Fatalf("Log: %v", err)
			}
			if len(entries) != tt.wantCommits {
				t.Errorf("commits = %d, want %d", len(entries), tt.wantCommits)
			}
		})
	}
}

func TestRepo_CommitFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setup       func(t *testing.T, dir string)
		filename    string
		message     string
		wantCommits int
		wantMessage string
	}{
		{
			name: "stages only the named file",
			setup: func(t *testing.T, dir string) {
				writeFile(t, dir, "a.md", "# A\n")
				writeFile(t, dir, "b.md", "# B\n")
			},
			filename:    "a.md",
			message:     "docs: add a",
			wantCommits: 1,
			wantMessage: "docs: add a",
		},
		{
			name: "no-op when file is unchanged",
			setup: func(t *testing.T, dir string) {
				writeFile(t, dir, "a.md", "# A\n")
				r, _ := gitops.InitRepo(dir)
				_ = r.CommitFile("a.md", "docs: initial")
			},
			filename:    "a.md",
			message:     "docs: should not appear",
			wantCommits: 1,
			wantMessage: "docs: initial",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()

			r, err := gitops.InitRepo(dir)
			if err != nil {
				t.Fatalf("InitRepo: %v", err)
			}

			tt.setup(t, dir)

			if err := r.CommitFile(tt.filename, tt.message); err != nil {
				t.Fatalf("CommitFile: %v", err)
			}

			entries, err := r.Log("", 0)
			if err != nil {
				t.Fatalf("Log: %v", err)
			}
			if len(entries) != tt.wantCommits {
				t.Errorf("commits = %d, want %d", len(entries), tt.wantCommits)
			}
			if tt.wantMessage != "" && len(entries) > 0 {
				if entries[0].Message != tt.wantMessage {
					t.Errorf("message = %q, want %q", entries[0].Message, tt.wantMessage)
				}
			}
		})
	}
}

func TestRepo_Log(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setup     func(t *testing.T, r *gitops.Repo, dir string)
		filename  string
		limit     int
		wantCount int
	}{
		{
			name: "returns all commits unfiltered",
			setup: func(t *testing.T, r *gitops.Repo, dir string) {
				writeFile(t, dir, "a.md", "# A\n")
				_ = r.CommitFile("a.md", "docs: add a")
				writeFile(t, dir, "b.md", "# B\n")
				_ = r.CommitFile("b.md", "docs: add b")
			},
			wantCount: 2,
		},
		{
			name: "filters by filename",
			setup: func(t *testing.T, r *gitops.Repo, dir string) {
				writeFile(t, dir, "a.md", "# A\n")
				_ = r.CommitFile("a.md", "docs: add a")
				writeFile(t, dir, "b.md", "# B\n")
				_ = r.CommitFile("b.md", "docs: add b")
			},
			filename:  "a.md",
			wantCount: 1,
		},
		{
			name: "respects limit",
			setup: func(t *testing.T, r *gitops.Repo, dir string) {
				writeFile(t, dir, "a.md", "# A\n")
				_ = r.CommitFile("a.md", "docs: first")
				writeFile(t, dir, "a.md", "# A updated\n")
				_ = r.CommitFile("a.md", "docs: second")
				writeFile(t, dir, "a.md", "# A final\n")
				_ = r.CommitFile("a.md", "docs: third")
			},
			limit:     2,
			wantCount: 2,
		},
		{
			name:      "empty repo returns nil",
			setup:     func(_ *testing.T, _ *gitops.Repo, _ string) {},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()

			r, err := gitops.InitRepo(dir)
			if err != nil {
				t.Fatalf("InitRepo: %v", err)
			}

			tt.setup(t, r, dir)

			entries, err := r.Log(tt.filename, tt.limit)
			if err != nil {
				t.Fatalf("Log: %v", err)
			}
			if len(entries) != tt.wantCount {
				t.Errorf("count = %d, want %d", len(entries), tt.wantCount)
			}
		})
	}
}

func TestRepo_Diff(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setup     func(t *testing.T, r *gitops.Repo, dir string)
		path      string
		wantEmpty bool
	}{
		{
			name: "clean worktree returns empty",
			setup: func(t *testing.T, r *gitops.Repo, dir string) {
				writeFile(t, dir, "note.md", "# Hello\n")
				_ = r.Commit("docs: initial")
			},
			path:      "note.md",
			wantEmpty: true,
		},
		{
			name: "modified file returns diff content",
			setup: func(t *testing.T, r *gitops.Repo, dir string) {
				writeFile(t, dir, "note.md", "# Hello\n")
				_ = r.Commit("docs: initial")
				writeFile(t, dir, "note.md", "# Hello World\n")
			},
			path:      "note.md",
			wantEmpty: false,
		},
		{
			name:      "no commits returns empty",
			setup:     func(_ *testing.T, _ *gitops.Repo, _ string) {},
			path:      "note.md",
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()

			r, err := gitops.InitRepo(dir)
			if err != nil {
				t.Fatalf("InitRepo: %v", err)
			}

			tt.setup(t, r, dir)

			got, err := r.Diff(tt.path)
			if err != nil {
				t.Fatalf("Diff: %v", err)
			}
			if tt.wantEmpty && got != "" {
				t.Errorf("expected empty diff, got %q", got)
			}
			if !tt.wantEmpty && got == "" {
				t.Error("expected non-empty diff, got empty")
			}
		})
	}
}

func TestFormatCommitMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		subject string
		body    string
		want    string
	}{
		{
			name:    "subject only",
			subject: "add meeting notes",
			body:    "",
			want:    "add meeting notes",
		},
		{
			name:    "subject and body",
			subject: "add meeting notes",
			body:    "Covered Q2 roadmap and team updates.",
			want:    "add meeting notes\n\nCovered Q2 roadmap and team updates.",
		},
		{
			name:    "long subject truncated",
			subject: "this is a very long subject line that definitely exceeds the fifty char limit",
			body:    "",
			want:    "this is a very long subject line that definitel...",
		},
		{
			name:    "trailing period removed",
			subject: "fix: correct typo in readme.",
			body:    "",
			want:    "fix: correct typo in readme",
		},
		{
			name:    "body wraps at 72 chars",
			subject: "feat: add feature",
			body:    "This is a body that contains enough words to eventually exceed the seventy two character line wrapping limit that we enforce.",
			want:    "feat: add feature\n\nThis is a body that contains enough words to eventually exceed the\nseventy two character line wrapping limit that we enforce.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := gitops.FormatCommitMessage(tt.subject, tt.body)
			if got != tt.want {
				t.Errorf(
					"FormatCommitMessage(%q, %q)\n  got  %q\n  want %q",
					tt.subject,
					tt.body,
					got,
					tt.want,
				)
			}
		})
	}
}
