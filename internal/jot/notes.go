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
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	headingRe   = regexp.MustCompile(`(?m)^#\s+(.+)$`)
	tagNameRe   = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_/-]*$`)
	secureKeyRe = regexp.MustCompile(`^\s*["']?secure["']?\s*:`)
)

// maxMatch caps a search result's matching line, so one long line does not
// cost an agent more than the summary it came with.
const maxMatch = 200

// Note is a markdown note backed by a .md file on disk.
type Note struct {
	Slug        string   `json:"slug"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags"`
	Secure      bool     `json:"secure"`
	Created     string   `json:"created"`
	Updated     string   `json:"updated"`
	Path        string   `json:"path"`
	Body        string   `json:"body"`
	Tasks       []Task   `json:"tasks"`
	Links       []string `json:"links"`

	raw string // the file as read, so rewrites can keep its front-matter byte for byte
}

// NoteSummary is a note without its body, tasks or links: enough for an
// agent to pick a note before reading it in full.
type NoteSummary struct {
	Slug        string   `json:"slug"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags"`
	Created     string   `json:"created"`
	Updated     string   `json:"updated"`
	Secure      bool     `json:"secure,omitempty"`
	OpenTasks   int      `json:"open_tasks"`
	Match       string   `json:"match,omitempty"`
}

// Task is a checkbox task extracted from a note's markdown body.
type Task struct {
	NoteSlug    string   `json:"note_slug"`
	Description string   `json:"description"`
	DueDate     string   `json:"due_date"`
	DoneDate    string   `json:"done_date"`
	Tags        []string `json:"tags"`
	Done        bool     `json:"done"`
	Line        int      `json:"line"` // 1-based line in the note file
}

// CleanSlug validates a note slug from user or agent input and returns it in
// canonical form: forward slashes, no ".md" suffix, inside the notes dir.
// Slugs may name a subdirectory, such as "done/2026-11-03".
func CleanSlug(slug string) (string, error) {
	clean := filepath.ToSlash(filepath.Clean(strings.TrimSuffix(slug, ".md")))
	if strings.TrimSpace(slug) == "" || filepath.IsAbs(slug) || clean == "." || clean == ".." ||
		strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("invalid note slug %q", slug)
	}
	return clean, nil
}

// slugFor returns the slug of the note file at path: its path relative to
// notesDir without ".md", or just its base name when it lies outside.
func slugFor(notesDir string, path string) string {
	rel, err := filepath.Rel(notesDir, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		rel = filepath.Base(path)
	}
	return filepath.ToSlash(strings.TrimSuffix(rel, ".md"))
}

// vaultDir is where kvlt keeps the "jot" vault's encrypted bodies, relative
// to the notes dir (kvlt's .kvlt/secrets/{type}/{name}/{key}.age layout).
const vaultDir = ".kvlt/secrets/local/jot/"

// vaultKey maps a slug to its kvlt key. kvlt keys cannot contain "/", so a
// folder slug escapes it ("ops%2Frunbook"); slugs without "/" or "%" keep
// the keys they always had.
func vaultKey(slug string) string {
	return strings.NewReplacer("%", "%25", "/", "%2F").Replace(slug)
}

// NoteSlugForPath maps a path in the notes repository to the slug of the
// note it stores: a note's .md file, or a secure note's encrypted body. ok
// is false for any other file.
func NoteSlugForPath(path string) (string, bool) {
	if key, ok := strings.CutPrefix(path, vaultDir); ok {
		key, isBody := strings.CutSuffix(key, ".age")
		slug, err := url.PathUnescape(key)
		return slug, isBody && err == nil
	}
	slug, ok := strings.CutSuffix(path, ".md")
	return slug, ok
}

// Summarize returns n without its body. When query is set, Match holds the
// first body line containing it (case-insensitive), trimmed and capped.
func Summarize(n *Note, query string) NoteSummary {
	s := NoteSummary{
		Slug:        n.Slug,
		Title:       n.Title,
		Description: n.Description,
		Tags:        n.Tags,
		Created:     n.Created,
		Updated:     n.Updated,
		Secure:      n.Secure,
	}
	for _, t := range n.Tasks {
		if !t.Done {
			s.OpenTasks++
		}
	}
	if query == "" {
		return s
	}
	lower := strings.ToLower(query)
	for _, line := range strings.Split(n.Body, "\n") {
		if strings.Contains(strings.ToLower(line), lower) {
			match := []rune(strings.TrimSpace(line))
			if len(match) > maxMatch {
				match = append(match[:maxMatch], '…')
			}
			s.Match = string(match)
			break
		}
	}
	return s
}

// DoneSince keeps the done tasks finished on or after since (YYYY-MM-DD).
func DoneSince(tasks []Task, since string) []Task {
	kept := make([]Task, 0, len(tasks))
	for _, t := range tasks {
		if t.Done && t.DoneDate >= since {
			kept = append(kept, t)
		}
	}
	return kept
}

// readNote reads the .md file at path, parses its front-matter, tasks and
// [[wiki-links]], then returns a fully-populated Note. Title falls back to
// the first level-1 heading when the front-matter title is empty.
func readNote(notesDir string, path string) (*Note, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("jot: read note %q: %w", path, err)
	}
	content := string(data)

	fm, body := ParseFrontmatter(content)

	slug := slugFor(notesDir, path)

	title := fm.Title
	if title == "" {
		title = titleFromBody(body)
	}
	if title == "" {
		title = slug
	}

	updated := ""
	if info, err := os.Stat(path); err == nil {
		updated = info.ModTime().Format(time.RFC3339)
	}

	// Task lines count from the top of the file, not the end of the
	// front-matter, so they match what an editor shows.
	offset := 0
	if strings.HasSuffix(content, body) {
		offset = strings.Count(content[:len(content)-len(body)], "\n")
	}

	raw := ParseTasks(body)
	tasks := make([]Task, 0, len(raw))
	for _, r := range raw {
		t := Task{
			NoteSlug:    slug,
			Description: r.Description,
			DueDate:     r.DueDate,
			DoneDate:    r.DoneDate,
			Done:        r.Done,
			Line:        r.Line + offset,
		}
		t.Tags = r.Labels
		tasks = append(tasks, t)
	}

	links := ParseLinks(body)

	return &Note{
		Slug:        slug,
		Title:       title,
		Description: fm.Description,
		Tags:        fm.Tags,
		Secure:      fm.Secure,
		Created:     fm.Created,
		Updated:     updated,
		Path:        path,
		Body:        body,
		Tasks:       tasks,
		Links:       links,
		raw:         content,
	}, nil
}

// readAllNotes walks notesDir (skipping .git) and reads every .md file
// concurrently. Errors on individual files are collected and returned as a
// combined error so that a single bad file does not abort the entire scan.
func readAllNotes(notesDir string) ([]*Note, error) {
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
			n, err := readNote(notesDir, p)
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

// listNotes returns all notes in notesDir, optionally filtered to those whose
// frontmatter tags or inline tags include tagFilter. Pass an empty string to
// return all notes.
func listNotes(notesDir string, tagFilter string) ([]*Note, error) {
	notes, err := readAllNotes(notesDir)
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

// searchNotes performs a case-insensitive substring search across note titles
// and bodies. All notes that contain query in either field are returned.
func searchNotes(notesDir string, query string) ([]*Note, error) {
	notes, err := readAllNotes(notesDir)
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

// slugPath validates slug and returns its canonical form and file path,
// refusing a path that symlinks lead outside the notes dir.
func slugPath(notesDir string, slug string) (string, string, error) {
	clean, err := CleanSlug(slug)
	if err != nil {
		return "", "", err
	}
	path := filepath.Join(notesDir, filepath.FromSlash(clean)+".md")
	if err := contained(notesDir, path); err != nil {
		return "", "", err
	}
	return clean, path, nil
}

// contained refuses path when the part of it that exists resolves, through
// symlinks, outside notesDir, or through a broken link a write would follow.
// ponytail: checked before use, not held open; os.Root if racing links matter.
func contained(
	notesDir string,
	path string,
) error {
	root, err := filepath.EvalSymlinks(notesDir)
	if err != nil {
		return nil // no notes dir yet, so nothing in it can link out
	}
	existing := path
	for {
		if _, err := os.Lstat(existing); err == nil {
			break
		}
		existing = filepath.Dir(existing)
	}
	resolved, err := filepath.EvalSymlinks(existing)
	if err != nil {
		return fmt.Errorf("note path %q goes through a broken symlink", path)
	}
	if rel, _ := filepath.Rel(root, resolved); rel == ".." ||
		strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("note path %q leaves the notes dir through a symlink", path)
	}
	return nil
}

// findNote reads and returns the note at notesDir/slug.md.
func findNote(notesDir string, slug string) (*Note, error) {
	_, path, err := slugPath(notesDir, slug)
	if err != nil {
		return nil, err
	}
	n, err := readNote(notesDir, path)
	if err != nil {
		return nil, fmt.Errorf("jot: find note %q: %w", slug, err)
	}
	return n, nil
}

// allTasks aggregates all tasks from every note in notesDir. status filters by
// "open" (Done == false), "done" (Done == true), or "all". tagFilter restricts
// to tasks whose Tags contain the given value; pass "" to skip tag filtering.
func allTasks(
	notesDir string,
	status string,
	tagFilter string,
) ([]Task, error) {
	notes, err := readAllNotes(notesDir)
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

// tasksDue returns open tasks whose DueDate falls within [from, to] inclusive.
// Dates are compared as YYYY-MM-DD strings (lexicographic order is correct for
// ISO-8601 dates).
func tasksDue(
	notesDir string,
	from time.Time,
	to time.Time,
) ([]Task, error) {
	notes, err := readAllNotes(notesDir)
	if err != nil {
		return nil, err
	}

	const dateFmt = "2006-01-02"
	fromStr := from.Format(dateFmt)
	toStr := to.Format(dateFmt)

	var tasks []Task
	for _, n := range notes {
		for _, t := range n.Tasks {
			if t.Done {
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

// allTags returns the deduplicated union of all tags across the notes in
// notesDir. Sources: frontmatter tags, inline #tags in the body, and task
// tags.
func allTags(notesDir string) ([]string, error) {
	notes, err := readAllNotes(notesDir)
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
// "open" requires Done == false, "done" requires Done == true, "all" always matches.
func matchStatus(t Task, status string) bool {
	switch status {
	case "done":
		return t.Done
	case "open":
		return !t.Done
	default: // "all" or ""
		return true
	}
}

// markTaskDone marks done the open task in notePath that desc names: the
// first whose description equals desc (case-insensitive), else the only one
// containing it. When several contain it, the error lists their lines so the
// caller can pick one with markTaskDoneAt.
func markTaskDone(
	notePath string,
	desc string,
) error {
	lower := strings.ToLower(strings.TrimSpace(desc))
	if lower == "" {
		return fmt.Errorf("task description is required")
	}
	lines, err := readLines(notePath)
	if err != nil {
		return err
	}

	var exact, partial []int
	for i, line := range lines {
		tasks := ParseTasks(line)
		if len(tasks) == 0 || tasks[0].Done {
			continue
		}
		d := strings.ToLower(tasks[0].Description)
		switch {
		case d == lower:
			exact = append(exact, i)
		case strings.Contains(d, lower):
			partial = append(partial, i)
		}
	}

	candidates := partial
	if len(exact) > 0 {
		candidates = exact
	}
	switch len(candidates) {
	case 0:
		return fmt.Errorf("no open task matching %q found in %s", desc, notePath)
	case 1:
		return checkOff(notePath, lines, candidates[0])
	}
	nums := make([]string, len(candidates))
	for i, idx := range candidates {
		nums[i] = strconv.Itoa(idx + 1)
	}
	return fmt.Errorf(
		"%d open tasks match %q in %s (lines %s); pick one by line",
		len(candidates),
		desc,
		notePath,
		strings.Join(nums, ", "),
	)
}

// markTaskDoneAt marks done the open task on line (1-based, as Task.Line
// reports it) of notePath.
func markTaskDoneAt(
	notePath string,
	line int,
) error {
	lines, err := readLines(notePath)
	if err != nil {
		return err
	}
	if line < 1 || line > len(lines) {
		return fmt.Errorf("line %d is outside %s", line, notePath)
	}
	if tasks := ParseTasks(lines[line-1]); len(tasks) == 0 || tasks[0].Done {
		return fmt.Errorf("line %d of %s is not an open task", line, notePath)
	}
	return checkOff(notePath, lines, line-1)
}

// readLines reads notePath split into lines.
func readLines(notePath string) ([]string, error) {
	content, err := os.ReadFile(notePath)
	if err != nil {
		return nil, fmt.Errorf("read note %q: %w", notePath, err)
	}
	return strings.Split(string(content), "\n"), nil
}

// checkOff marks the task on lines[i] done today, keeping its indentation,
// and writes the file back.
func checkOff(
	notePath string,
	lines []string,
	i int,
) error {
	t := ParseTasks(lines[i])[0]
	t.Done = true
	t.DoneDate = time.Now().Format("2006-01-02")
	indent := lines[i][:len(lines[i])-len(strings.TrimLeft(lines[i], " \t"))]
	lines[i] = indent + FormatTask(t)
	return os.WriteFile(notePath, []byte(strings.Join(lines, "\n")), 0o600)
}

// addTask appends task as a checkbox line to notesDir/slug.md, creating the
// note (and any parent directories) when it does not exist. It refuses
// anything that would not parse back to the same task: a multi-line or
// piped description, an inline #tag in the description, a malformed tag or
// date. Secure notes are refused because their body lives in kvlt.
func addTask(
	notesDir string,
	slug string,
	title string,
	task RawTask,
) error {
	clean, notePath, err := slugPath(notesDir, slug)
	if err != nil {
		return err
	}

	task.Description = strings.TrimSpace(task.Description)
	switch {
	case task.Description == "":
		return fmt.Errorf("task description is required")
	case strings.ContainsAny(task.Description, "|\r\n"):
		return fmt.Errorf("task description must be one line without \"|\": %q", task.Description)
	case len(ParseLineTags(task.Description)) > 0:
		return fmt.Errorf(
			"task description contains a #tag; pass tags separately: %q",
			task.Description,
		)
	}
	for _, t := range task.Labels {
		if !tagNameRe.MatchString(t) {
			return fmt.Errorf("invalid tag %q", t)
		}
	}
	for _, d := range []string{task.DueDate, task.DoneDate} {
		if _, err := time.Parse("2006-01-02", d); d != "" && err != nil {
			return fmt.Errorf("invalid date %q: want a real YYYY-MM-DD date", d)
		}
	}
	if task.Done && task.DoneDate == "" {
		task.DoneDate = time.Now().Format("2006-01-02")
	}

	content, err := os.ReadFile(notePath)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		if title == "" {
			title = filepath.Base(clean)
		}
		if err := os.MkdirAll(filepath.Dir(notePath), 0o700); err != nil {
			return fmt.Errorf("create dir for %q: %w", slug, err)
		}
		content = []byte(ScaffoldFrontmatter(title, time.Now().Format("2006-01-02")))
	case err != nil:
		return fmt.Errorf("read note %q: %w", notePath, err)
	default:
		if fm, _ := ParseFrontmatter(string(content)); fm.Secure {
			return fmt.Errorf("note %q is encrypted; decrypt it before adding tasks", slug)
		}
	}

	text := string(content)
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	return os.WriteFile(notePath, []byte(text+FormatTask(task)+"\n"), 0o600)
}

// splitFrontmatter returns content's YAML front-matter block byte for byte,
// "---" delimiters included, and the body after it. ok is false when content
// does not open with a closed "---" block; then body is all of content.
func splitFrontmatter(content string) (string, string, bool) {
	lines := strings.SplitAfter(content, "\n")
	if strings.TrimRight(lines[0], "\r\n") != "---" {
		return "", content, false
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], "\r\n") == "---" {
			head := strings.Join(lines[:i+1], "")
			return head, content[len(head):], true
		}
	}
	return "", content, false
}

// setSecure returns a front-matter block with its secure key removed, in
// any quoting, and, when secure is true, "secure: true" added before the
// closing delimiter. Every other line, line endings included, stays as
// written.
func setSecure(head string, secure bool) string {
	lines := strings.SplitAfter(head, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	eol := strings.TrimPrefix(lines[0], "---") // "\n" or "\r\n"
	closing := lines[len(lines)-1]

	kept := make([]string, 0, len(lines)+1)
	for _, line := range lines[:len(lines)-1] {
		if !secureKeyRe.MatchString(line) {
			kept = append(kept, line)
		}
	}
	if secure {
		kept = append(kept, "secure: true"+eol)
	}
	return strings.Join(append(kept, closing), "")
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
