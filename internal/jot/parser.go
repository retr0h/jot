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
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	// Matches: - [ ] or - [x] at the start (with optional leading whitespace)
	checkboxRe = regexp.MustCompile(`^(\s*-\s+\[)([ xX])(\]\s+)(.+)$`)
	// Matches inline #tags
	hashTagStripRe = regexp.MustCompile(`\s+#[A-Za-z][A-Za-z0-9_/-]*`)
)

// RawTask represents a single task extracted from a markdown checkbox line.
type RawTask struct {
	Description string
	DueDate     string
	DoneDate    string
	Labels      []string
	Line        int
	Done        bool
}

// ParseTasks extracts all task checkboxes from markdown content.
// A task is any line matching `- [ ] ...` or `- [x] ...` that contains
// a pipe-delimited metadata field (due:, done:) or inline #tags.
// Plain checkboxes without metadata are also captured.
//
// Format: - [ ] description | due:friday #tag1 #tag2
//   - [x] description | due:2026-05-16 | done:2026-05-16 #tag1
func ParseTasks(content string) []RawTask {
	lines := strings.Split(content, "\n")
	tasks := make([]RawTask, 0, len(lines))

	for i, line := range lines {
		m := checkboxRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}

		checked := m[2] == "x" || m[2] == "X"
		rest := m[4]

		task := parseTaskContent(rest)
		task.Line = i + 1
		task.Done = checked
		tasks = append(tasks, task)
	}

	return tasks
}

// parseTaskContent parses the content after `- [ ] ` or `- [x] `.
// Splits on `|` for metadata fields, extracts inline #tags.
func parseTaskContent(content string) RawTask {
	tags := ParseLineTags(content)
	stripped := strings.TrimSpace(hashTagStripRe.ReplaceAllString(content, ""))

	task := RawTask{Labels: tags}

	fields := strings.Split(stripped, "|")
	for idx, field := range fields {
		field = strings.TrimSpace(field)
		switch {
		case idx == 0:
			task.Description = field
		case strings.HasPrefix(field, "due:"):
			task.DueDate = strings.TrimPrefix(field, "due:")
		case strings.HasPrefix(field, "done:"):
			task.DoneDate = strings.TrimPrefix(field, "done:")
		}
	}

	return task
}

// FormatTask produces a complete checkbox line from a RawTask:
//
//   - [ ] description | due:friday #tag1 #tag2
//   - [x] description | due:friday | done:2026-05-17 #tag1 #tag2
func FormatTask(task RawTask) string {
	check := " "
	if task.Done {
		check = "x"
	}

	parts := []string{task.Description}
	if task.DueDate != "" {
		parts = append(parts, "due:"+task.DueDate)
	}
	if task.DoneDate != "" {
		parts = append(parts, "done:"+task.DoneDate)
	}

	result := "- [" + check + "] " + strings.Join(parts, " | ")

	if len(task.Labels) > 0 {
		tagStr := make([]string, len(task.Labels))
		for i, t := range task.Labels {
			tagStr[i] = "#" + t
		}
		result += " " + strings.Join(tagStr, " ")
	}

	return result
}

// NewTask builds a task from user input. due and done accept any ParseDate
// form ("friday", "2026-05-20") resolved relative to now; an empty done
// leaves the task open. Tags may carry a leading "#". AddTask validates the
// result before writing it.
func NewTask(
	desc string,
	due string,
	done string,
	tags []string,
	now time.Time,
) (RawTask, error) {
	task := RawTask{Description: strings.TrimSpace(desc)}

	if due != "" {
		d, err := ParseDate(due, now)
		if err != nil {
			return RawTask{}, fmt.Errorf("due: %w", err)
		}
		task.DueDate = d.Format("2006-01-02")
	}

	if done != "" {
		d, err := ParseDate(done, now)
		if err != nil {
			return RawTask{}, fmt.Errorf("done: %w", err)
		}
		task.Done = true
		task.DoneDate = d.Format("2006-01-02")
	}

	labels := make([]string, 0, len(tags))
	for _, t := range tags {
		labels = append(labels, strings.TrimPrefix(strings.TrimSpace(t), "#"))
	}
	if len(labels) > 0 {
		task.Labels = labels
	}

	return task, nil
}

// DiffTasks compares old and new content, returns added/removed/unchanged tasks.
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
// replaces the task line with the resolved string.
func ApplyResolutions(content string, resolutions map[int]string) string {
	lines := strings.Split(content, "\n")

	for lineNum, resolved := range resolutions {
		idx := lineNum - 1
		if idx < 0 || idx >= len(lines) {
			continue
		}
		lines[idx] = resolved
	}

	return strings.Join(lines, "\n")
}
