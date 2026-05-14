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
	"regexp"
	"strings"
)

var (
	bareTaskRe     = regexp.MustCompile(`@task\s+(.+)$`)
	resolvedTaskRe = regexp.MustCompile(`@task\(([^)]+)\)`)
)

// RawTask represents a single @task marker extracted from a note.
type RawTask struct {
	Description string
	DueDate     string   // "2026-05-16" or ""
	DoneDate    string   // "2026-05-14" or ""
	Labels      []string
	Line        int  // 1-indexed line number in the note
	Resolved    bool // true if parsed from @task(...) form
}

// ParseTasks extracts all @task markers from markdown content.
// Returns them in document order. Handles both bare and resolved forms.
func ParseTasks(content string) []RawTask {
	lines := strings.Split(content, "\n")
	tasks := make([]RawTask, 0, len(lines))

	for i, line := range lines {
		lineNum := i + 1

		// Try resolved form first: @task(...)
		if m := resolvedTaskRe.FindStringSubmatch(line); m != nil {
			task := parseResolvedInner(m[1])
			task.Line = lineNum
			task.Resolved = true
			tasks = append(tasks, task)
			continue
		}

		// Try bare form: @task <description>
		if m := bareTaskRe.FindStringSubmatch(line); m != nil {
			tasks = append(tasks, RawTask{
				Description: strings.TrimSpace(m[1]),
				Line:        lineNum,
				Resolved:    false,
			})
		}
	}

	return tasks
}

// parseResolvedInner parses the inner content of @task(...), splitting on "|".
func parseResolvedInner(inner string) RawTask {
	fields := strings.Split(inner, "|")
	task := RawTask{}

	for idx, field := range fields {
		field = strings.TrimSpace(field)
		switch {
		case idx == 0:
			task.Description = field
		case strings.HasPrefix(field, "due:"):
			task.DueDate = strings.TrimPrefix(field, "due:")
		case strings.HasPrefix(field, "done:"):
			task.DoneDate = strings.TrimPrefix(field, "done:")
		case strings.HasPrefix(field, "label:"):
			raw := strings.TrimPrefix(field, "label:")
			task.Labels = strings.Split(raw, ",")
		}
	}

	return task
}

// ResolveLine produces the resolved form: @task(desc | due:X | label:Y | done:Z).
// Only includes fields that are non-empty. DoneDate from the RawTask is preserved.
func ResolveLine(task RawTask, due string, labels []string) string {
	parts := []string{task.Description}

	if due != "" {
		parts = append(parts, "due:"+due)
	}

	if len(labels) > 0 {
		parts = append(parts, "label:"+strings.Join(labels, ","))
	}

	if task.DoneDate != "" {
		parts = append(parts, "done:"+task.DoneDate)
	}

	return "@task(" + strings.Join(parts, " | ") + ")"
}

// DiffTasks compares old and new content, returns added/removed/unchanged tasks.
// Matching is by description (string equality).
func DiffTasks(oldContent, newContent string) (added, removed, unchanged []RawTask) {
	oldTasks := ParseTasks(oldContent)
	newTasks := ParseTasks(newContent)

	oldByDesc := make(map[string]RawTask, len(oldTasks))
	for _, t := range oldTasks {
		oldByDesc[t.Description] = t
	}

	newByDesc := make(map[string]RawTask, len(newTasks))
	for _, t := range newTasks {
		newByDesc[t.Description] = t
	}

	for _, t := range newTasks {
		if _, exists := oldByDesc[t.Description]; exists {
			unchanged = append(unchanged, t)
		} else {
			added = append(added, t)
		}
	}

	for _, t := range oldTasks {
		if _, exists := newByDesc[t.Description]; !exists {
			removed = append(removed, t)
		}
	}

	return added, removed, unchanged
}

// ApplyResolutions takes note content and a map of lineNumber→resolvedString,
// replaces the @task marker on each line with the resolved string.
func ApplyResolutions(content string, resolutions map[int]string) string {
	lines := strings.Split(content, "\n")

	for lineNum, resolved := range resolutions {
		idx := lineNum - 1
		if idx < 0 || idx >= len(lines) {
			continue
		}
		line := lines[idx]

		// Replace the resolved form first if present, then bare form.
		if resolvedTaskRe.MatchString(line) {
			lines[idx] = resolvedTaskRe.ReplaceAllLiteralString(line, resolved)
		} else if bareTaskRe.MatchString(line) {
			lines[idx] = bareTaskRe.ReplaceAllLiteralString(line, resolved)
		}
	}

	return strings.Join(lines, "\n")
}
