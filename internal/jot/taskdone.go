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
	"os"
	"strings"
	"time"
)

// MarkTaskDone finds the first @task line in notePath whose description
// contains desc (case-insensitive substring), appends done:YYYY-MM-DD, and
// writes the file back. Returns an error when no matching task is found.
func MarkTaskDone(
	notePath string,
	desc string,
) error {
	content, err := os.ReadFile(notePath)
	if err != nil {
		return fmt.Errorf("read note %q: %w", notePath, err)
	}

	today := time.Now().Format("2006-01-02")
	lower := strings.ToLower(desc)
	lines := strings.Split(string(content), "\n")
	found := false

	for i, line := range lines {
		tasks := ParseTasks(line)
		for _, t := range tasks {
			if !strings.Contains(strings.ToLower(t.Description), lower) {
				continue
			}
			lines[i] = rewriteTaskDone(line, t, today)
			found = true
			break
		}
		if found {
			break
		}
	}

	if !found {
		return fmt.Errorf("no @task matching %q found in %s", desc, notePath)
	}

	return os.WriteFile(notePath, []byte(strings.Join(lines, "\n")), 0o600)
}

// rewriteTaskDone rewrites a single @task line to include done:YYYY-MM-DD.
// Resolved form is updated in-place; bare form is promoted to resolved.
func rewriteTaskDone(
	line string,
	t RawTask,
	today string,
) string {
	t.DoneDate = today
	if t.Resolved {
		// Rebuild the resolved @task(...) with done appended.
		parts := []string{t.Description}
		if t.DueDate != "" {
			parts = append(parts, "due:"+t.DueDate)
		}
		if t.DoneDate != "" {
			parts = append(parts, "done:"+t.DoneDate)
		}
		newMarker := "@task(" + strings.Join(parts, " | ") + ")"
		// Preserve any inline #tags that were after the closing paren.
		return resolvedTaskRe.ReplaceAllLiteralString(line, newMarker)
	}
	// Bare @task — convert to resolved form with done.
	resolved := ResolveLine(t, t.DueDate, t.Labels)
	return bareTaskRe.ReplaceAllLiteralString(line, resolved)
}
