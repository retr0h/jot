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

package jot

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/retr0h/kvlt/pkg/kvlt"
)

// Service holds configuration state and provides all jot domain operations
// as methods. Consumers define narrow interfaces against this type.
type Service struct {
	NotesDir  string
	ConfigDir string
	SSHKeys   []string
	Prompt    kvlt.PassphrasePrompt
}

// CatNote returns the body of a note by slug, transparently decrypting
// from kvlt if the note is marked secure.
func (s *Service) CatNote(slug string) (string, error) {
	note, err := FindNote(s.NotesDir, slug)
	if err != nil {
		return "", err
	}

	if note.Secure {
		store, err := NewSecureStore(s.ConfigDir, s.SSHKeys, s.Prompt)
		if err != nil {
			return "", fmt.Errorf("open secure store: %w", err)
		}
		return store.ReadNote(context.Background(), slug)
	}

	return note.Body, nil
}

// EncryptNote encrypts an existing plaintext note. The body is stored in
// kvlt and the .md file is rewritten with secure: true.
func (s *Service) EncryptNote(slug string) error {
	note, err := FindNote(s.NotesDir, slug)
	if err != nil {
		return err
	}
	if note.Secure {
		return fmt.Errorf("note %q is already encrypted", slug)
	}

	store, err := NewSecureStore(s.ConfigDir, s.SSHKeys, s.Prompt)
	if err != nil {
		return fmt.Errorf("open secure store: %w", err)
	}

	if err := store.WriteNote(context.Background(), slug, note.Body); err != nil {
		return err
	}

	scaffold := fmt.Sprintf(
		"---\ntitle: %q\ntags: [%s]\ncreated: %s\nsecure: true\n---\n",
		note.Title,
		FormatTags(note.Tags),
		note.Created,
	)
	if err := os.WriteFile(note.Path, []byte(scaffold), 0o600); err != nil {
		return fmt.Errorf("rewrite note file: %w", err)
	}

	return nil
}

// DecryptNote restores a secure note to plaintext. The body is read from
// kvlt, written back into the .md file, and removed from the vault.
func (s *Service) DecryptNote(slug string) error {
	note, err := FindNote(s.NotesDir, slug)
	if err != nil {
		return err
	}
	if !note.Secure {
		return fmt.Errorf("note %q is not encrypted", slug)
	}

	store, err := NewSecureStore(s.ConfigDir, s.SSHKeys, s.Prompt)
	if err != nil {
		return fmt.Errorf("open secure store: %w", err)
	}

	body, err := store.ReadNote(context.Background(), slug)
	if err != nil {
		return err
	}

	content := fmt.Sprintf(
		"---\ntitle: %q\ntags: [%s]\ncreated: %s\n---\n\n%s",
		note.Title,
		FormatTags(note.Tags),
		note.Created,
		body,
	)
	if err := os.WriteFile(note.Path, []byte(content), 0o600); err != nil {
		return fmt.Errorf("rewrite note file: %w", err)
	}

	if err := store.DeleteNote(context.Background(), slug); err != nil {
		return fmt.Errorf("remove from secure store: %w", err)
	}

	return nil
}

// RenameNote renames a note file, rewrites its frontmatter title and first
// heading, and updates all [[wikilinks]]. Returns the new slug.
func (s *Service) RenameNote(
	oldSlug string,
	newTitle string,
) (string, error) {
	newSlug := NewSlug(newTitle, time.Now())

	oldPath := filepath.Join(s.NotesDir, oldSlug+".md")
	newPath := filepath.Join(s.NotesDir, newSlug+".md")

	content, err := os.ReadFile(oldPath)
	if err != nil {
		return "", fmt.Errorf("read note %q: %w", oldSlug, err)
	}

	updated := RewriteNoteTitle(string(content), newTitle)

	if err := os.MkdirAll(filepath.Dir(newPath), 0o700); err != nil {
		return "", fmt.Errorf("create dir for %q: %w", newSlug, err)
	}
	if err := os.WriteFile(newPath, []byte(updated), 0o600); err != nil {
		return "", fmt.Errorf("write renamed note: %w", err)
	}
	if err := os.Remove(oldPath); err != nil {
		return "", fmt.Errorf("remove old note: %w", err)
	}

	_ = RewriteWikiLinks(s.NotesDir, oldSlug, newSlug)

	return newSlug, nil
}

// ListNotes returns all notes, optionally filtered by tag.
func (s *Service) ListNotes(tagFilter string) ([]*Note, error) {
	return ListNotes(s.NotesDir, tagFilter)
}

// SearchNotes performs a case-insensitive substring search.
func (s *Service) SearchNotes(query string) ([]*Note, error) {
	return SearchNotes(s.NotesDir, query)
}

// FindNoteBySlug reads and returns the note at notesDir/slug.md.
func (s *Service) FindNoteBySlug(slug string) (*Note, error) {
	return FindNote(s.NotesDir, slug)
}

// AllTasks aggregates all tasks, filtered by status and tag.
func (s *Service) AllTasks(
	status string,
	tagFilter string,
) ([]Task, error) {
	return AllTasks(s.NotesDir, status, tagFilter)
}

// TasksDue returns open tasks due within [from, to].
func (s *Service) TasksDue(
	from time.Time,
	to time.Time,
) ([]Task, error) {
	return TasksDue(s.NotesDir, from, to)
}

// AllTags returns deduplicated tags across all notes.
func (s *Service) AllTags() ([]string, error) {
	return AllTags(s.NotesDir)
}

// MarkTaskDone marks a task as complete in the given note.
func (s *Service) MarkTaskDone(
	slug string,
	desc string,
) error {
	notePath := filepath.Join(s.NotesDir, slug+".md")
	return MarkTaskDone(notePath, desc)
}

// CreateNote creates a new note with the given title and content. If secure
// is true, the body is stored in kvlt and the file contains only frontmatter.
// Returns the slug and file path.
func (s *Service) CreateNote(
	title string,
	content string,
	secure bool,
) (string, string, error) {
	slug := NewSlug(title, time.Now())

	if err := os.MkdirAll(s.NotesDir, 0o700); err != nil {
		return "", "", fmt.Errorf("create notes dir: %w", err)
	}

	notePath := filepath.Join(s.NotesDir, slug+".md")

	if secure {
		store, err := NewSecureStore(s.ConfigDir, s.SSHKeys, s.Prompt)
		if err != nil {
			return "", "", fmt.Errorf("open secure store: %w", err)
		}
		if err := store.WriteNote(context.Background(), slug, content); err != nil {
			return "", "", fmt.Errorf("encrypt note: %w", err)
		}
		scaffold := fmt.Sprintf(
			"---\ntitle: %q\ntags: []\ncreated: %s\nsecure: true\n---\n",
			title,
			time.Now().Format("2006-01-02"),
		)
		if err := os.WriteFile(notePath, []byte(scaffold), 0o600); err != nil {
			return "", "", fmt.Errorf("write note file: %w", err)
		}
	} else {
		body := content
		if body == "" {
			body = ScaffoldFrontmatter(title, time.Now().Format("2006-01-02"))
		}
		if err := os.WriteFile(notePath, []byte(body), 0o600); err != nil {
			return "", "", fmt.Errorf("write note file: %w", err)
		}
	}

	return slug, notePath, nil
}

// DeleteNote removes the note file for the given slug.
func (s *Service) DeleteNote(slug string) error {
	notePath := filepath.Join(s.NotesDir, slug+".md")
	if err := os.Remove(notePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete note %q: %w", slug, err)
	}
	return nil
}

// RewriteNoteTitle replaces the frontmatter title: line and the first #
// heading with newTitle.
func RewriteNoteTitle(content string, newTitle string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "title:") {
			lines[i] = fmt.Sprintf("title: %q", newTitle)
			continue
		}
		if strings.HasPrefix(line, "# ") {
			lines[i] = "# " + newTitle
		}
	}
	return strings.Join(lines, "\n")
}

// RewriteWikiLinks walks notesDir and replaces [[oldSlug]] with [[newSlug]].
func RewriteWikiLinks(
	notesDir string,
	oldSlug string,
	newSlug string,
) error {
	return filepath.WalkDir(notesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".md") {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		updated := strings.ReplaceAll(
			string(content),
			"[["+oldSlug+"]]",
			"[["+newSlug+"]]",
		)
		if updated != string(content) {
			_ = os.WriteFile(path, []byte(updated), 0o600)
		}
		return nil
	})
}

// FormatTags formats a tag slice as a comma-separated string for YAML output.
func FormatTags(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	var b strings.Builder
	for i, t := range tags {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(t)
	}
	return b.String()
}
