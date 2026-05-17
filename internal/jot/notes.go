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

// Package jot holds the core logic — filesystem-backed notes, task parser,
// editor integration, and kvlt-backed encryption.
package jot

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
)

var headingRe = regexp.MustCompile(`(?m)^#\s+(.+)$`)

// Note is a markdown note backed by a .md file on disk. SQLite is not
// involved; the file is the single source of truth.
type Note struct {
	Slug    string   `json:"slug"`
	Title   string   `json:"title"`
	Tags    []string `json:"tags"`
	Secure  bool     `json:"secure"`
	Created string   `json:"created"`
	Path    string   `json:"path"`
	Body    string   `json:"body"`
	Tasks   []Task   `json:"tasks"`
	Links   []string `json:"links"`
}

// Task is a @task item extracted from a note's markdown body.
type Task struct {
	NoteSlug    string   `json:"note_slug"`
	Description string   `json:"description"`
	DueDate     string   `json:"due_date"`
	DoneDate    string   `json:"done_date"`
	Tags        []string `json:"tags"`
	Done        string   `json:"done"`
	Line        int      `json:"line"`
}

// ReadNote reads the .md file at path, parses its front-matter, @task markers,
// and [[wiki-links]], then returns a fully-populated Note. Title falls back to
// the first level-1 heading when the front-matter title is empty.
func ReadNote(path string) (*Note, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("jot: read note %q: %w", path, err)
	}
	content := string(data)

	fm, body := ParseFrontmatter(content)

	slug := strings.TrimSuffix(filepath.Base(path), ".md")

	title := fm.Title
	if title == "" {
		title = titleFromBody(body)
	}
	if title == "" {
		title = slug
	}

	raw := ParseTasks(body)
	tasks := make([]Task, 0, len(raw))
	for _, r := range raw {
		done := ""
		if r.Done {
			done = "done"
		}
		t := Task{
			NoteSlug:    slug,
			Description: r.Description,
			DueDate:     r.DueDate,
			DoneDate:    r.DoneDate,
			Done:        done,
			Line:        r.Line,
		}
		t.Tags = r.Labels
		tasks = append(tasks, t)
	}

	links := ParseLinks(body)

	return &Note{
		Slug:    slug,
		Title:   title,
		Tags:    fm.Tags,
		Secure:  fm.Secure,
		Created: fm.Created,
		Path:    path,
		Body:    body,
		Tasks:   tasks,
		Links:   links,
	}, nil
}

// ReadAllNotes walks notesDir (skipping .git) and reads every .md file
// concurrently. Errors on individual files are collected and returned as a
// combined error so that a single bad file does not abort the entire scan.
func ReadAllNotes(notesDir string) ([]*Note, error) {
	var paths []string
	if err := filepath.WalkDir(notesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".md") {
			paths = append(paths, path)
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("jot: walk %q: %w", notesDir, err)
	}

	type result struct {
		note *Note
		err  error
	}

	results := make([]result, len(paths))
	var wg sync.WaitGroup
	wg.Add(len(paths))
	for i, p := range paths {
		go func() {
			defer wg.Done()
			n, err := ReadNote(p)
			results[i] = result{note: n, err: err}
		}()
	}
	wg.Wait()

	notes := make([]*Note, 0, len(paths))
	var errs []string
	for _, r := range results {
		if r.err != nil {
			errs = append(errs, r.err.Error())
			continue
		}
		notes = append(notes, r.note)
	}
	if len(errs) > 0 {
		return notes, fmt.Errorf("jot: read notes errors: %s", strings.Join(errs, "; "))
	}
	return notes, nil
}

// ListNotes returns all notes in notesDir, optionally filtered to those whose
// frontmatter tags or inline tags include tagFilter. Pass an empty string to
// return all notes.
func ListNotes(notesDir string, tagFilter string) ([]*Note, error) {
	notes, err := ReadAllNotes(notesDir)
	if err != nil {
		return nil, err
	}
	if tagFilter == "" {
		return notes, nil
	}
	var filtered []*Note
	for _, n := range notes {
		if hasLabel(n.Tags, tagFilter) {
			filtered = append(filtered, n)
		}
	}
	return filtered, nil
}

// SearchNotes performs a case-insensitive substring search across note titles
// and bodies. All notes that contain query in either field are returned.
func SearchNotes(notesDir string, query string) ([]*Note, error) {
	notes, err := ReadAllNotes(notesDir)
	if err != nil {
		return nil, err
	}
	lower := strings.ToLower(query)
	var matched []*Note
	for _, n := range notes {
		if strings.Contains(strings.ToLower(n.Title), lower) ||
			strings.Contains(strings.ToLower(n.Body), lower) {
			matched = append(matched, n)
		}
	}
	return matched, nil
}

// FindNote reads and returns the note at notesDir/slug.md.
func FindNote(notesDir string, slug string) (*Note, error) {
	path := filepath.Join(notesDir, slug+".md")
	n, err := ReadNote(path)
	if err != nil {
		return nil, fmt.Errorf("jot: find note %q: %w", slug, err)
	}
	return n, nil
}

// AllTasks aggregates all tasks from every note in notesDir. status filters by
// "open" (Done == ""), "done" (Done != ""), or "all". tagFilter restricts to
// tasks whose Tags contain the given value; pass "" to skip tag filtering.
func AllTasks(
	notesDir string,
	status string,
	tagFilter string,
) ([]Task, error) {
	notes, err := ReadAllNotes(notesDir)
	if err != nil {
		return nil, err
	}

	var tasks []Task
	for _, n := range notes {
		for _, t := range n.Tasks {
			if !matchStatus(t, status) {
				continue
			}
			if tagFilter != "" && !hasLabel(t.Tags, tagFilter) {
				continue
			}
			tasks = append(tasks, t)
		}
	}
	return tasks, nil
}

// TasksDue returns open tasks whose DueDate falls within [from, to] inclusive.
// Dates are compared as YYYY-MM-DD strings (lexicographic order is correct for
// ISO-8601 dates).
func TasksDue(
	notesDir string,
	from time.Time,
	to time.Time,
) ([]Task, error) {
	notes, err := ReadAllNotes(notesDir)
	if err != nil {
		return nil, err
	}

	const dateFmt = "2006-01-02"
	fromStr := from.Format(dateFmt)
	toStr := to.Format(dateFmt)

	var tasks []Task
	for _, n := range notes {
		for _, t := range n.Tasks {
			if t.Done != "" {
				continue
			}
			if t.DueDate == "" {
				continue
			}
			if t.DueDate >= fromStr && t.DueDate <= toStr {
				tasks = append(tasks, t)
			}
		}
	}
	return tasks, nil
}

// AllTags returns the deduplicated union of all tags across the notes in
// notesDir. Sources: frontmatter tags, inline #tags in the body, and task
// tags.
func AllTags(notesDir string) ([]string, error) {
	notes, err := ReadAllNotes(notesDir)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	for _, n := range notes {
		for _, tag := range n.Tags {
			seen[tag] = struct{}{}
		}
		for _, tag := range ParseInlineTags(n.Body) {
			seen[tag] = struct{}{}
		}
		for _, t := range n.Tasks {
			for _, tag := range t.Tags {
				seen[tag] = struct{}{}
			}
		}
	}

	tags := make([]string, 0, len(seen))
	for tag := range seen {
		tags = append(tags, tag)
	}
	return tags, nil
}

// ─── helpers ─────────────────────────────────────────────────────────────────

// hasLabel reports whether labels contains the given value.
func hasLabel(labels []string, value string) bool {
	return slices.Contains(labels, value)
}

// matchStatus reports whether task t matches the requested status filter.
// "open" requires Done == "", "done" requires Done != "", "all" always matches.
func matchStatus(t Task, status string) bool {
	switch status {
	case "done":
		return t.Done != ""
	case "open":
		return t.Done == ""
	default: // "all" or ""
		return true
	}
}

// titleFromBody extracts the text of the first level-1 heading from body.
// Returns an empty string when no heading is found.
func titleFromBody(body string) string {
	m := headingRe.FindStringSubmatch(body)
	if m == nil {
		return ""
	}
	return strings.TrimSpace(m[1])
}
