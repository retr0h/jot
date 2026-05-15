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

// writeFile is a test helper that writes content to path inside dir.
func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writeFile %q: %v", path, err)
	}
	return path
}

func TestInitRepo_Idempotent(t *testing.T) {
	dir := t.TempDir()

	r1, err := gitops.InitRepo(dir)
	if err != nil {
		t.Fatalf("first InitRepo: %v", err)
	}
	if r1 == nil {
		t.Fatal("expected non-nil Repo")
	}

	// Second call on the same directory must succeed without error.
	r2, err := gitops.InitRepo(dir)
	if err != nil {
		t.Fatalf("second InitRepo (idempotent): %v", err)
	}
	if r2 == nil {
		t.Fatal("expected non-nil Repo on second call")
	}
}

func TestCommitAndLog(t *testing.T) {
	dir := t.TempDir()

	r, err := gitops.InitRepo(dir)
	if err != nil {
		t.Fatalf("InitRepo: %v", err)
	}

	writeFile(t, dir, "note.md", "# Hello\n")

	if err := r.Commit("docs: add note"); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	entries, err := r.Log("", 0)
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}
	if entries[0].Message != "docs: add note" {
		t.Errorf("message = %q, want %q", entries[0].Message, "docs: add note")
	}
	if entries[0].Hash == "" {
		t.Error("hash must not be empty")
	}
}

func TestCommitSkipsClean(t *testing.T) {
	dir := t.TempDir()

	r, err := gitops.InitRepo(dir)
	if err != nil {
		t.Fatalf("InitRepo: %v", err)
	}

	writeFile(t, dir, "note.md", "# Hello\n")
	if err := r.Commit("docs: initial"); err != nil {
		t.Fatalf("first Commit: %v", err)
	}

	// No changes — second commit must be a no-op.
	if err := r.Commit("docs: no-op"); err != nil {
		t.Fatalf("second Commit (clean): %v", err)
	}

	entries, err := r.Log("", 0)
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected exactly 1 commit, got %d", len(entries))
	}
}

func TestCommitFile(t *testing.T) {
	dir := t.TempDir()

	r, err := gitops.InitRepo(dir)
	if err != nil {
		t.Fatalf("InitRepo: %v", err)
	}

	writeFile(t, dir, "a.md", "# A\n")
	writeFile(t, dir, "b.md", "# B\n")

	// CommitFile only stages a.md.
	if err := r.CommitFile("a.md", "docs: add a"); err != nil {
		t.Fatalf("CommitFile: %v", err)
	}

	entries, err := r.Log("", 0)
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 commit, got %d", len(entries))
	}
	if entries[0].Message != "docs: add a" {
		t.Errorf("message = %q, want %q", entries[0].Message, "docs: add a")
	}
}

func TestLogFilterByFile(t *testing.T) {
	dir := t.TempDir()

	r, err := gitops.InitRepo(dir)
	if err != nil {
		t.Fatalf("InitRepo: %v", err)
	}

	writeFile(t, dir, "a.md", "# A\n")
	if err := r.CommitFile("a.md", "docs: add a"); err != nil {
		t.Fatalf("CommitFile a: %v", err)
	}

	writeFile(t, dir, "b.md", "# B\n")
	if err := r.CommitFile("b.md", "docs: add b"); err != nil {
		t.Fatalf("CommitFile b: %v", err)
	}

	entries, err := r.Log("a.md", 0)
	if err != nil {
		t.Fatalf("Log(a.md): %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry for a.md, got %d", len(entries))
	}
	if entries[0].Message != "docs: add a" {
		t.Errorf("message = %q, want %q", entries[0].Message, "docs: add a")
	}
}

func TestDiff_ReturnsString(t *testing.T) {
	dir := t.TempDir()

	r, err := gitops.InitRepo(dir)
	if err != nil {
		t.Fatalf("InitRepo: %v", err)
	}

	writeFile(t, dir, "note.md", "# Hello\n")
	if err := r.Commit("docs: initial"); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	// Diff should return without error after at least one commit.
	_, err = r.Diff("note.md")
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
}

func TestFormatCommitMessage(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := gitops.FormatCommitMessage(tt.subject, tt.body)
			if got != tt.want {
				t.Errorf("FormatCommitMessage(%q, %q)\n  got  %q\n  want %q", tt.subject, tt.body, got, tt.want)
			}
		})
	}
}
