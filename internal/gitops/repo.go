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

// Package gitops wraps go-git for version control on the notes directory.
// Every note write goes through Commit or CommitFile so that the full edit
// history is always available via Log, Diff, and ShowCommit.
package gitops

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// LogEntry is a single commit in the repository history.
type LogEntry struct {
	Hash    string
	Message string
	Date    time.Time // commit author date
	Author  string
	When    time.Time // alias for Date; kept for backward compatibility
}

// Repo wraps a go-git Repository bound to the notes directory.
type Repo struct {
	repo *git.Repository
	path string // notes directory path
}

// InitRepo opens the git repository at notesDir, initialising a fresh one
// if none exists yet. Safe to call multiple times on the same directory.
func InitRepo(notesDir string) (*Repo, error) {
	repo, err := git.PlainInit(notesDir, false)
	if err != nil {
		if !errors.Is(err, git.ErrRepositoryAlreadyExists) {
			return nil, fmt.Errorf("gitops: init %q: %w", notesDir, err)
		}
		// Already initialised — open instead.
		repo, err = git.PlainOpen(notesDir)
		if err != nil {
			return nil, fmt.Errorf("gitops: open after init %q: %w", notesDir, err)
		}
	}
	return &Repo{repo: repo, path: notesDir}, nil
}

// OpenRepo opens an existing git repository at notesDir.
func OpenRepo(notesDir string) (*Repo, error) {
	repo, err := git.PlainOpen(notesDir)
	if err != nil {
		return nil, fmt.Errorf("gitops: open %q: %w", notesDir, err)
	}
	return &Repo{repo: repo, path: notesDir}, nil
}

// Commit stages all changes (git add .) and creates a commit with message.
// It is a no-op when the working tree is clean.
func (r *Repo) Commit(message string) error {
	w, err := r.repo.Worktree()
	if err != nil {
		return fmt.Errorf("gitops: worktree: %w", err)
	}

	if err := w.AddGlob("."); err != nil {
		return fmt.Errorf("gitops: add glob: %w", err)
	}

	status, err := w.Status()
	if err != nil {
		return fmt.Errorf("gitops: status: %w", err)
	}
	if status.IsClean() {
		return nil
	}

	cfg, _ := r.repo.Config()
	name := "jot"
	email := "jot@localhost"
	if cfg != nil && cfg.User.Name != "" {
		name = cfg.User.Name
	}
	if cfg != nil && cfg.User.Email != "" {
		email = cfg.User.Email
	}

	if _, err := w.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  name,
			Email: email,
			When:  time.Now(),
		},
	}); err != nil {
		return fmt.Errorf("gitops: commit: %w", err)
	}
	return nil
}

// CommitFile stages a single file and creates a commit. It is a no-op when
// the file is unchanged.
func (r *Repo) CommitFile(
	filename string,
	message string,
) error {
	w, err := r.repo.Worktree()
	if err != nil {
		return fmt.Errorf("gitops: worktree: %w", err)
	}

	if _, err := w.Add(filename); err != nil {
		return fmt.Errorf("gitops: add %q: %w", filename, err)
	}

	status, err := w.Status()
	if err != nil {
		return fmt.Errorf("gitops: status: %w", err)
	}
	if status.IsClean() {
		return nil
	}

	if _, err := w.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "jot",
			Email: "jot@localhost",
			When:  time.Now(),
		},
	}); err != nil {
		return fmt.Errorf("gitops: commit file %q: %w", filename, err)
	}
	return nil
}

// Log returns the commit history for the repository. When filename is
// non-empty only commits that touched that file are returned. Pass limit <= 0
// for no cap (up to 1000 entries). Returns nil, nil when the repository has
// no commits yet.
func (r *Repo) Log(
	filename string,
	limit int,
) ([]LogEntry, error) {
	if limit <= 0 {
		limit = 1000
	}

	opts := &git.LogOptions{
		Order: git.LogOrderCommitterTime,
	}
	if filename != "" {
		opts.FileName = &filename
	}

	iter, err := r.repo.Log(opts)
	if err != nil {
		if errors.Is(err, plumbing.ErrReferenceNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("gitops: log: %w", err)
	}
	defer iter.Close()

	var entries []LogEntry
	for range limit {
		c, err := iter.Next()
		if err != nil {
			break
		}
		when := c.Author.When
		entries = append(entries, LogEntry{
			Hash:    c.Hash.String()[:7],
			Message: strings.TrimSpace(c.Message),
			Date:    when,
			Author:  c.Author.Name,
			When:    when,
		})
	}
	return entries, nil
}

// Diff returns the working-tree diff versus HEAD for the given file path.
// Returns an empty string when there are no changes or the repo has no
// commits.
func (r *Repo) Diff(path string) (string, error) {
	w, err := r.repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("gitops: worktree: %w", err)
	}

	status, err := w.Status()
	if err != nil {
		return "", fmt.Errorf("gitops: status: %w", err)
	}
	if status.IsClean() {
		return "", nil
	}

	ref, err := r.repo.Head()
	if err != nil {
		// No commits yet.
		return "", nil
	}

	headCommit, err := r.repo.CommitObject(ref.Hash())
	if err != nil {
		return "", fmt.Errorf("gitops: commit object: %w", err)
	}

	headTree, err := headCommit.Tree()
	if err != nil {
		return "", fmt.Errorf("gitops: head tree: %w", err)
	}

	var sb strings.Builder
	for filePath, fileStatus := range status {
		if path != "" && filePath != path {
			continue
		}
		fmt.Fprintf(&sb, "--- %s (status: %v)\n", filePath, fileStatus.Worktree)
		f, err := headTree.File(filePath)
		if err == nil {
			content, err := f.Contents()
			if err == nil {
				fmt.Fprintf(&sb, "+++ %s\n%s\n", filePath, content)
			}
		}
	}
	return sb.String(), nil
}

// ShowCommit returns the unified diff introduced by the commit whose hash
// starts with hashPrefix. When filename is non-empty only the diff for that
// path is included.
func (r *Repo) ShowCommit(
	hash string,
	path string,
) (string, error) {
	// Try exact hash first.
	h := plumbing.NewHash(hash)
	if !h.IsZero() {
		c, err := r.repo.CommitObject(h)
		if err == nil {
			return formatCommitPatch(c, path)
		}
	}

	// Fall back to prefix search.
	iter, err := r.repo.CommitObjects()
	if err != nil {
		return "", fmt.Errorf("gitops: commit objects: %w", err)
	}
	defer iter.Close()

	for {
		c, err := iter.Next()
		if err != nil {
			break
		}
		if strings.HasPrefix(c.Hash.String(), hash) {
			return formatCommitPatch(c, path)
		}
	}
	return "", fmt.Errorf("gitops: commit %q not found", hash)
}

func formatCommitPatch(c *object.Commit, path string) (string, error) {
	patch, err := c.Patch(nil)
	if err != nil {
		return "", fmt.Errorf("gitops: patch: %w", err)
	}
	raw := patch.String()
	if path == "" {
		return raw, nil
	}
	// Filter to lines related to path.
	var sb strings.Builder
	for _, line := range strings.Split(raw, "\n") {
		if strings.Contains(line, path) ||
			strings.HasPrefix(line, "@@") ||
			strings.HasPrefix(line, "---") ||
			strings.HasPrefix(line, "+++") ||
			strings.HasPrefix(line, "+") ||
			strings.HasPrefix(line, "-") {
			sb.WriteString(line + "\n")
		}
	}
	return sb.String(), nil
}

// FormatCommitMessage formats a commit message following the 50/72 rule.
// The subject is trimmed of trailing periods and truncated to 50 chars
// (appending "..." when truncated). The body text is appended after a blank
// separator line and wrapped at 72 chars.
func FormatCommitMessage(subject string, body string) string {
	subject = strings.TrimRight(strings.TrimSpace(subject), ".")
	if len(subject) > 50 {
		subject = subject[:47] + "..."
	}

	if body == "" {
		return subject
	}

	var sb strings.Builder
	sb.WriteString(subject)
	sb.WriteString("\n\n")

	words := strings.Fields(body)
	lineLen := 0
	for i, w := range words {
		wl := len(w)
		if lineLen > 0 && lineLen+1+wl > 72 {
			sb.WriteByte('\n')
			lineLen = 0
		} else if i > 0 {
			sb.WriteByte(' ')
			lineLen++
		}
		sb.WriteString(w)
		lineLen += wl
	}

	return sb.String()
}
