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

// CatNote returns the body of a note by slug, transparently decrypting
// from kvlt if the note is marked secure.
func CatNote(
	notesDir string,
	configDir string,
	sshKeys []string,
	slug string,
	prompt kvlt.PassphrasePrompt,
) (string, error) {
	note, err := FindNote(notesDir, slug)
	if err != nil {
		return "", err
	}

	if note.Secure {
		store, err := NewSecureStore(configDir, sshKeys, prompt)
		if err != nil {
			return "", fmt.Errorf("open secure store: %w", err)
		}
		return store.ReadNote(context.Background(), slug)
	}

	return note.Body, nil
}

// EncryptNote encrypts an existing plaintext note. The body is stored in
// kvlt and the .md file is rewritten to contain only frontmatter with
// secure: true.
func EncryptNote(
	notesDir string,
	configDir string,
	sshKeys []string,
	slug string,
	prompt kvlt.PassphrasePrompt,
) error {
	note, err := FindNote(notesDir, slug)
	if err != nil {
		return err
	}
	if note.Secure {
		return fmt.Errorf("note %q is already encrypted", slug)
	}

	store, err := NewSecureStore(configDir, sshKeys, prompt)
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
func DecryptNote(
	notesDir string,
	configDir string,
	sshKeys []string,
	slug string,
	prompt kvlt.PassphrasePrompt,
) error {
	note, err := FindNote(notesDir, slug)
	if err != nil {
		return err
	}
	if !note.Secure {
		return fmt.Errorf("note %q is not encrypted", slug)
	}

	store, err := NewSecureStore(configDir, sshKeys, prompt)
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
// heading, and updates all [[wikilinks]] in the notes directory. Returns
// the new slug.
func RenameNote(
	notesDir string,
	oldSlug string,
	newTitle string,
) (string, error) {
	newSlug := NewSlug(newTitle, time.Now())

	oldPath := filepath.Join(notesDir, oldSlug+".md")
	newPath := filepath.Join(notesDir, newSlug+".md")

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

	_ = RewriteWikiLinks(notesDir, oldSlug, newSlug)

	return newSlug, nil
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

// RewriteWikiLinks walks notesDir and replaces [[oldSlug]] with [[newSlug]]
// in every .md file.
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
