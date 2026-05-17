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

// MarkTaskDone finds the first checkbox task in notePath whose description
// contains desc (case-insensitive substring), flips [ ] → [x], and writes
// the file back. The git commit timestamp records when it was completed.
func MarkTaskDone(
	notePath string,
	desc string,
) error {
	content, err := os.ReadFile(notePath)
	if err != nil {
		return fmt.Errorf("read note %q: %w", notePath, err)
	}

	lower := strings.ToLower(desc)
	lines := strings.Split(string(content), "\n")
	found := false

	for i, line := range lines {
		tasks := ParseTasks(line)
		for _, t := range tasks {
			if t.Done {
				continue
			}
			if !strings.Contains(strings.ToLower(t.Description), lower) {
				continue
			}
			t.Done = true
			t.DoneDate = time.Now().Format("2006-01-02")
			lines[i] = FormatTask(t)
			found = true
			break
		}
		if found {
			break
		}
	}

	if !found {
		return fmt.Errorf("no open task matching %q found in %s", desc, notePath)
	}

	return os.WriteFile(notePath, []byte(strings.Join(lines, "\n")), 0o600)
}
