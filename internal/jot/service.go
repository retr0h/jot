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
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/retr0h/kvlt/pkg/kvlt"

	"github.com/retr0h/jot/internal/lock"
)

// Service holds configuration state and provides all jot domain operations
// as methods. Consumers define narrow interfaces against this type.
type Service struct {
	NotesDir string
	SSHKeys  []string
	Prompt   kvlt.PassphrasePrompt
	// Committer, when set, commits each write while the write lock is still
	// held, so concurrent writers never fold one another's changes together.
	Committer Committer
}

// Committer commits the notes repository for a caller that holds its write
// lock; *gitops.Repo satisfies it.
type Committer interface {
	CommitHeld(subject string) error
}

// write runs fn under the notes dir's write lock and, when fn succeeds,
// commits with the subject fn returns before releasing the lock.
func (s *Service) write(fn func() (string, error)) error {
	return lock.With(s.NotesDir, func() error {
		subject, err := fn()
		if err != nil || s.Committer == nil {
			return err
		}
		if err := s.Committer.CommitHeld(subject); err != nil {
			return fmt.Errorf("saved, but the commit failed: %w", err)
		}
		return nil
	})
}

// cleanOr returns slug in canonical form for a commit subject, so
// "/work/todo" and "work/todo" commit alike; a slug CleanSlug refuses is
// kept as given, since the write it names fails anyway.
func cleanOr(slug string) string {
	if clean, err := CleanSlug(slug); err == nil {
		return clean
	}
	return slug
}

// CatNote returns the body of a note by slug, transparently decrypting
// from kvlt if the note is marked secure.
func (s *Service) CatNote(slug string) (string, error) {
	note, err := findNote(s.NotesDir, slug)
	if err != nil {
		return "", err
	}

	if note.Secure {
		store, err := NewSecureStore(s.NotesDir, s.SSHKeys, s.Prompt)
		if err != nil {
			return "", fmt.Errorf("open secure store: %w", err)
		}
		return store.ReadNote(context.Background(), vaultKey(note.Slug))
	}

	return note.Body, nil
}

// EncryptNote encrypts an existing plaintext note. The body is stored in
// kvlt and the .md file keeps its front-matter, every field as written,
// plus secure: true.
func (s *Service) EncryptNote(slug string) error {
	return s.write(func() (string, error) {
		return "note(secure): encrypt " + cleanOr(slug), s.encryptNote(slug)
	})
}

func (s *Service) encryptNote(slug string) error {
	note, err := findNote(s.NotesDir, slug)
	if err != nil {
		return err
	}
	if note.Secure {
		return fmt.Errorf("note %q is already encrypted", slug)
	}

	store, err := NewSecureStore(s.NotesDir, s.SSHKeys, s.Prompt)
	if err != nil {
		return fmt.Errorf("open secure store: %w", err)
	}

	_, body, _ := splitFrontmatter(note.raw)
	if err := store.WriteNote(context.Background(), vaultKey(note.Slug), body); err != nil {
		return err
	}

	if err := os.WriteFile(note.Path, []byte(setSecure(noteHead(note), true)), 0o600); err != nil {
		return fmt.Errorf("rewrite note file: %w", err)
	}

	return nil
}

// DecryptNote restores a secure note to plaintext. The body is read from
// kvlt, written back under the note's front-matter, and removed from the
// vault.
func (s *Service) DecryptNote(slug string) error {
	return s.write(func() (string, error) {
		return "note(secure): decrypt " + cleanOr(slug), s.decryptNote(slug)
	})
}

func (s *Service) decryptNote(slug string) error {
	note, err := findNote(s.NotesDir, slug)
	if err != nil {
		return err
	}
	if !note.Secure {
		return fmt.Errorf("note %q is not encrypted", slug)
	}

	store, err := NewSecureStore(s.NotesDir, s.SSHKeys, s.Prompt)
	if err != nil {
		return fmt.Errorf("open secure store: %w", err)
	}

	body, err := store.ReadNote(context.Background(), vaultKey(note.Slug))
	if err != nil {
		return err
	}

	if !strings.HasPrefix(body, "\n") && !strings.HasPrefix(body, "\r\n") {
		body = "\n" + body
	}
	content := setSecure(noteHead(note), false) + body
	if err := os.WriteFile(note.Path, []byte(content), 0o600); err != nil {
		return fmt.Errorf("rewrite note file: %w", err)
	}

	if err := store.DeleteNote(context.Background(), vaultKey(note.Slug)); err != nil {
		return fmt.Errorf("remove from secure store: %w", err)
	}

	return nil
}

// noteHead returns note's front-matter block as written, or one rebuilt from
// its parsed fields when the file has no "---" block to keep.
func noteHead(n *Note) string {
	if head, _, ok := splitFrontmatter(n.raw); ok {
		return head
	}
	return fmt.Sprintf(
		"---\ntitle: %q\ntags: [%s]\ncreated: %s\n---\n",
		n.Title,
		FormatTags(n.Tags),
		n.Created,
	)
}

// AppendNote adds text to the end of an existing note's body. A secure
// note's body is updated in the vault.
func (s *Service) AppendNote(
	slug string,
	text string,
) error {
	text = strings.TrimRight(text, "\n")
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("text to append is required")
	}
	return s.write(func() (string, error) {
		return "note: append to " + cleanOr(slug), s.rewriteBody(slug, func(body string) string {
			if body != "" && !strings.HasSuffix(body, "\n") {
				body += "\n"
			}
			return body + text + "\n"
		})
	})
}

// UpdateNote replaces an existing note's body and keeps its front-matter.
// Content that opens with its own "---" front-matter block replaces the
// whole file instead. Neither form can change whether a note is encrypted;
// encrypt_note and decrypt_note do that.
func (s *Service) UpdateNote(
	slug string,
	content string,
) error {
	return s.write(func() (string, error) {
		return "note: update " + cleanOr(slug), s.updateNote(slug, content)
	})
}

func (s *Service) updateNote(
	slug string,
	content string,
) error {
	if _, _, whole := splitFrontmatter(content); !whole {
		return s.rewriteBody(slug, func(string) string { return content })
	}
	if fm, _ := ParseFrontmatter(content); fm.Secure {
		return fmt.Errorf("content sets secure: true; encrypt the note instead")
	}
	note, err := findNote(s.NotesDir, slug)
	if err != nil {
		return err
	}
	if note.Secure {
		return fmt.Errorf("note %q is encrypted; update its body without front-matter", slug)
	}
	return os.WriteFile(note.Path, []byte(content), 0o600)
}

// rewriteBody replaces a note's body with edit(body), keeping its
// front-matter byte for byte. A secure note's body lives in the vault.
func (s *Service) rewriteBody(
	slug string,
	edit func(body string) string,
) error {
	note, err := findNote(s.NotesDir, slug)
	if err != nil {
		return err
	}

	if note.Secure {
		store, err := NewSecureStore(s.NotesDir, s.SSHKeys, s.Prompt)
		if err != nil {
			return fmt.Errorf("open secure store: %w", err)
		}
		body, err := store.ReadNote(context.Background(), vaultKey(note.Slug))
		if err != nil {
			return err
		}
		return store.WriteNote(context.Background(), vaultKey(note.Slug), edit(body))
	}

	head, body, hasHead := splitFrontmatter(note.raw)
	body = edit(body)
	if hasHead && !strings.HasPrefix(body, "\n") {
		body = "\n" + body
	}
	if !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	return os.WriteFile(note.Path, []byte(head+body), 0o600)
}

// RenameNote renames a note file within its folder, rewrites its
// frontmatter title and first heading, and updates all [[wikilinks]].
// Returns the new slug.
func (s *Service) RenameNote(
	oldSlug string,
	newTitle string,
) (string, error) {
	var newSlug string
	err := s.write(func() (string, error) {
		var err error
		newSlug, err = s.renameNote(oldSlug, newTitle)
		return fmt.Sprintf("note: rename %s -> %s", cleanOr(oldSlug), newSlug), err
	})
	return newSlug, err
}

func (s *Service) renameNote(
	oldSlug string,
	newTitle string,
) (string, error) {
	oldSlug, oldPath, err := slugPath(s.NotesDir, oldSlug)
	if err != nil {
		return "", err
	}
	newSlug, err := noteName(newTitle)
	if err != nil {
		return "", err
	}
	if dir := path.Dir(oldSlug); dir != "." {
		newSlug = dir + "/" + newSlug
	}
	newPath := filepath.Join(s.NotesDir, filepath.FromSlash(newSlug)+".md")

	content, err := os.ReadFile(oldPath)
	if err != nil {
		return "", fmt.Errorf("read note %q: %w", oldSlug, err)
	}

	updated := RewriteNoteTitle(string(content), newTitle)

	// A secure note's body lives in the vault under its slug, so it is copied
	// to the new slug first and the old entry removed only after the rename.
	fm, _ := ParseFrontmatter(string(content))
	if fm.Secure {
		if err := s.copyVaultEntry(oldSlug, newSlug); err != nil {
			return "", err
		}
	}

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
	// Links can name a note in a folder by its file name alone, unless
	// another note still has that name and the link may mean it.
	if base := path.Base(oldSlug); base != oldSlug && !nameTaken(s.NotesDir, base) {
		_ = RewriteWikiLinks(s.NotesDir, base, path.Base(newSlug))
	}
	if fm.Secure {
		// The copy already succeeded; a failed delete only leaves an orphaned blob.
		if store, err := NewSecureStore(s.NotesDir, s.SSHKeys, s.Prompt); err == nil {
			_ = store.DeleteNote(context.Background(), vaultKey(oldSlug))
		}
	}

	return newSlug, nil
}

// copyVaultEntry copies a secure note's body from one slug's vault key to
// another's.
func (s *Service) copyVaultEntry(
	from string,
	to string,
) error {
	store, err := NewSecureStore(s.NotesDir, s.SSHKeys, s.Prompt)
	if err != nil {
		return fmt.Errorf("open secure store: %w", err)
	}
	body, err := store.ReadNote(context.Background(), vaultKey(from))
	if err == nil {
		err = store.WriteNote(context.Background(), vaultKey(to), body)
	}
	return err
}

// nameTaken reports whether any note in notesDir is named base.md.
func nameTaken(
	notesDir string,
	base string,
) bool {
	taken := false
	_ = filepath.WalkDir(notesDir, func(_ string, d fs.DirEntry, _ error) error {
		if d != nil && !d.IsDir() && d.Name() == base+".md" {
			taken = true
			return filepath.SkipAll
		}
		return nil
	})
	return taken
}

// ListNotes returns all notes, optionally filtered by tag and to those under
// dir (see notesUnder).
func (s *Service) ListNotes(tagFilter string, dir string) ([]*Note, error) {
	return listNotes(s.NotesDir, tagFilter, dir)
}

// SearchNotes performs a case-insensitive substring search over the notes
// under dir (see notesUnder).
func (s *Service) SearchNotes(query string, dir string) ([]*Note, error) {
	return searchNotes(s.NotesDir, query, dir)
}

// FindNoteBySlug reads and returns the note at notesDir/slug.md.
func (s *Service) FindNoteBySlug(slug string) (*Note, error) {
	return findNote(s.NotesDir, slug)
}

// AllTasks aggregates all tasks, filtered by status, tag, and the notes under
// dir (see notesUnder).
func (s *Service) AllTasks(
	status string,
	tagFilter string,
	dir string,
) ([]Task, error) {
	return allTasks(s.NotesDir, status, tagFilter, dir)
}

// TasksDue returns open tasks due within [from, to].
func (s *Service) TasksDue(
	from time.Time,
	to time.Time,
) ([]Task, error) {
	return tasksDue(s.NotesDir, from, to)
}

// AllTags returns deduplicated tags across all notes.
func (s *Service) AllTags() ([]string, error) {
	return allTags(s.NotesDir)
}

// MarkTaskDone marks done the open task in the given note that desc names:
// an exact description match, else the only one containing desc, recording
// why it closed (see checkOff). It refuses a task with an open blocker
// unless force is set.
func (s *Service) MarkTaskDone(
	slug string,
	desc string,
	reason string,
	force bool,
) error {
	_, notePath, err := slugPath(s.NotesDir, slug)
	if err != nil {
		return err
	}
	return s.write(func() (string, error) {
		subject := fmt.Sprintf("task: done %q in %s", desc, cleanOr(slug))
		byID, err := s.blockerIndex(force)
		if err != nil {
			return subject, err
		}
		return subject, markTaskDone(notePath, desc, reason, byID)
	})
}

// MarkTaskDoneAt marks done the open task on line (1-based, as Task.Line
// reports it) of the given note, recording why it closed (see checkOff). It
// refuses a task with an open blocker unless force is set.
func (s *Service) MarkTaskDoneAt(
	slug string,
	line int,
	reason string,
	force bool,
) error {
	_, notePath, err := slugPath(s.NotesDir, slug)
	if err != nil {
		return err
	}
	return s.write(func() (string, error) {
		subject := fmt.Sprintf("task: done line %d in %s", line, cleanOr(slug))
		byID, err := s.blockerIndex(force)
		if err != nil {
			return subject, err
		}
		return subject, markTaskDoneAt(notePath, line, reason, byID)
	})
}

// MarkTaskDoneByID marks done the open task carrying id, in whichever note
// holds it, recording why it closed (see checkOff). It refuses a task with
// an open blocker unless force is set.
func (s *Service) MarkTaskDoneByID(
	id string,
	reason string,
	force bool,
) error {
	id = strings.ToLower(strings.TrimSpace(id))
	return s.write(func() (string, error) {
		subject := "task: done " + id
		byID, err := tasksByID(s.NotesDir)
		if err != nil {
			return subject, err
		}
		found := byID[id]
		switch {
		case len(found) == 0:
			return subject, fmt.Errorf("no task with id %s", id)
		case len(found) > 1:
			return subject, fmt.Errorf(
				"id %s is on %d tasks; pick one by note and line",
				id,
				len(found),
			)
		case found[0].Done:
			return subject, fmt.Errorf("task %s is already done", id)
		}
		// The slug came from a file under NotesDir, so it maps straight back to it.
		notePath := filepath.Join(s.NotesDir, filepath.FromSlash(found[0].NoteSlug)+".md")
		if force {
			byID = nil
		}
		return subject + " in " + found[0].NoteSlug, markTaskDoneAt(
			notePath,
			found[0].Line,
			reason,
			byID,
		)
	})
}

// blockerIndex returns the task index the blocker check reads, or nil when
// force skips the check.
func (s *Service) blockerIndex(force bool) (map[string][]Task, error) {
	if force {
		return nil, nil
	}
	return tasksByID(s.NotesDir)
}

// AddTask appends a checkbox task to the note at slug, creating the note
// with title (the slug's last segment when empty) if it does not exist.
// Slugs may name a subdirectory, such as "done/2026-11-03". It returns the
// task as written, with the ID it was given.
func (s *Service) AddTask(
	slug string,
	title string,
	task RawTask,
) (RawTask, error) {
	var added RawTask
	err := s.write(func() (string, error) {
		subject := fmt.Sprintf(
			"task: add %q to %s",
			strings.TrimSpace(task.Description),
			cleanOr(slug),
		)
		var err error
		added, err = addTask(s.NotesDir, slug, title, task)
		return subject, err
	})
	return added, err
}

// ReadNoteByPath reads and returns the note at the given absolute path.
func (s *Service) ReadNoteByPath(path string) (*Note, error) {
	return readNote(s.NotesDir, path)
}

// ReadAllNotes walks the notes directory and returns all notes.
func (s *Service) ReadAllNotes() ([]*Note, error) {
	return readAllNotes(s.NotesDir)
}

// CreateNote creates a new note with the given title and content. A title
// of "dir/Title" creates the note in that folder, named from the title
// alone. Content gets jot's title, tags and created fields where its
// front-matter lacks them. If secure is true, the body is stored in kvlt
// and the file contains only frontmatter. It refuses to overwrite an
// existing note. Returns the slug and file path.
func (s *Service) CreateNote(
	title string,
	content string,
	secure bool,
) (string, string, error) {
	var slug, notePath string
	err := s.write(func() (string, error) {
		var err error
		slug, notePath, err = s.createNote(title, content, secure)
		scope := "note"
		if secure {
			scope = "note(secure)"
		}
		return scope + ": add " + slug, err
	})
	return slug, notePath, err
}

func (s *Service) createNote(
	title string,
	content string,
	secure bool,
) (string, string, error) {
	dir := ""
	if i := strings.LastIndex(title, "/"); i >= 0 {
		dir, title = title[:i], title[i+1:]
	}
	name, err := noteName(title)
	if err != nil {
		return "", "", err
	}
	slug, notePath, err := slugPath(s.NotesDir, path.Join(dir, name))
	if err != nil {
		return "", "", err
	}

	if err := os.MkdirAll(filepath.Dir(notePath), 0o700); err != nil {
		return "", "", fmt.Errorf("create notes dir: %w", err)
	}
	if _, err := os.Stat(notePath); err == nil {
		return "", "", fmt.Errorf("note %q already exists; update or append to it instead", slug)
	}

	if secure {
		store, err := NewSecureStore(s.NotesDir, s.SSHKeys, s.Prompt)
		if err != nil {
			return "", "", fmt.Errorf("open secure store: %w", err)
		}
		if err := store.WriteNote(context.Background(), vaultKey(slug), content); err != nil {
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
		body := ScaffoldFrontmatter(title, time.Now().Format("2006-01-02"))
		if content != "" {
			body = withDefaults(content, title, time.Now().Format("2006-01-02"))
		}
		if err := os.WriteFile(notePath, []byte(body), 0o600); err != nil {
			return "", "", fmt.Errorf("write note file: %w", err)
		}
	}

	return slug, notePath, nil
}

// DeleteNote removes the note file for the given slug.
func (s *Service) DeleteNote(slug string) error {
	return s.write(
		func() (string, error) { return "note: delete " + cleanOr(slug), s.deleteNote(slug) },
	)
}

func (s *Service) deleteNote(slug string) error {
	_, notePath, err := slugPath(s.NotesDir, slug)
	if err != nil {
		return err
	}
	if err := os.Remove(notePath); err != nil && !errors.Is(err, fs.ErrNotExist) {
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
