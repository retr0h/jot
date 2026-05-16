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

package jot_test

import (
	"testing"

	"github.com/retr0h/jot/internal/jot"
)

func TestParseTasks(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantCount int
		wantDesc  string // first task description, if any
	}{
		{
			name:      "bare @task",
			content:   "- @task review meshx PR",
			wantCount: 1,
			wantDesc:  "review meshx PR",
		},
		{
			name:      "resolved @task with inline tags",
			content:   "- @task(review meshx PR | due:2026-05-16) #meshx #pr",
			wantCount: 1,
			wantDesc:  "review meshx PR",
		},
		{
			name:      "multiple bare tasks",
			content:   "- @task first\n- @task second",
			wantCount: 2,
			wantDesc:  "first",
		},
		{
			name:      "no tasks",
			content:   "just text",
			wantCount: 0,
		},
		{
			name:      "task in middle of document",
			content:   "# Title\n\ntext\n\n- @task fix bug\n\nDone.",
			wantCount: 1,
			wantDesc:  "fix bug",
		},
		{
			name:      "done task",
			content:   "- @task(reviewed PR | done:2026-05-14)",
			wantCount: 1,
			wantDesc:  "reviewed PR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := jot.ParseTasks(tt.content)
			if len(got) != tt.wantCount {
				t.Errorf(
					"ParseTasks(%q): got %d tasks, want %d",
					tt.content,
					len(got),
					tt.wantCount,
				)
			}
			if tt.wantCount > 0 && tt.wantDesc != "" {
				if got[0].Description != tt.wantDesc {
					t.Errorf(
						"ParseTasks(%q): first desc = %q, want %q",
						tt.content,
						got[0].Description,
						tt.wantDesc,
					)
				}
			}
		})
	}
}

// TestParseTaskResolved verifies that tags are read from inline #tags OUTSIDE
// the @task(...) parens, not from a label: field inside them.
func TestParseTaskResolved(t *testing.T) {
	// New format: tags are outside the parens as #tags.
	content := "- @task(review meshx PR | due:2026-05-16) #meshx #pr"
	tasks := jot.ParseTasks(content)

	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}

	task := tasks[0]

	if task.DueDate != "2026-05-16" {
		t.Errorf("DueDate = %q, want %q", task.DueDate, "2026-05-16")
	}

	wantLabels := []string{"meshx", "pr"}
	if len(task.Labels) != len(wantLabels) {
		t.Fatalf("Labels = %v, want %v", task.Labels, wantLabels)
	}
	for i, l := range wantLabels {
		if task.Labels[i] != l {
			t.Errorf("Labels[%d] = %q, want %q", i, task.Labels[i], l)
		}
	}

	if !task.Resolved {
		t.Error("expected Resolved = true")
	}
}

func TestParseTaskDone(t *testing.T) {
	content := "- @task(reviewed PR | done:2026-05-14)"
	tasks := jot.ParseTasks(content)

	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}

	if tasks[0].DoneDate != "2026-05-14" {
		t.Errorf("DoneDate = %q, want %q", tasks[0].DoneDate, "2026-05-14")
	}
}

// TestParseTaskBareTags verifies that inline #tags on a bare @task line are
// stripped from the description and collected into Labels.
func TestParseTaskBareTags(t *testing.T) {
	content := "- @task review meshx PR #meshx #pr"
	tasks := jot.ParseTasks(content)

	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}

	task := tasks[0]
	if task.Description != "review meshx PR" {
		t.Errorf("Description = %q, want %q", task.Description, "review meshx PR")
	}

	wantLabels := []string{"meshx", "pr"}
	if len(task.Labels) != len(wantLabels) {
		t.Fatalf("Labels = %v, want %v", task.Labels, wantLabels)
	}
	for i, l := range wantLabels {
		if task.Labels[i] != l {
			t.Errorf("Labels[%d] = %q, want %q", i, task.Labels[i], l)
		}
	}
}

func TestResolveLine(t *testing.T) {
	tests := []struct {
		name   string
		task   jot.RawTask
		due    string
		labels []string
		want   string
	}{
		{
			name:   "with due and labels",
			task:   jot.RawTask{Description: "review PR"},
			due:    "2026-05-16",
			labels: []string{"meshx", "pr"},
			want:   "@task(review PR | due:2026-05-16) #meshx #pr",
		},
		{
			name:   "no due",
			task:   jot.RawTask{Description: "review PR"},
			due:    "",
			labels: []string{"meshx"},
			want:   "@task(review PR) #meshx",
		},
		{
			name:   "no labels",
			task:   jot.RawTask{Description: "review PR"},
			due:    "2026-05-16",
			labels: nil,
			want:   "@task(review PR | due:2026-05-16)",
		},
		{
			name:   "done preserved from task",
			task:   jot.RawTask{Description: "review PR", DoneDate: "2026-05-14"},
			due:    "",
			labels: nil,
			want:   "@task(review PR | done:2026-05-14)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := jot.ResolveLine(tt.task, tt.due, tt.labels)
			if got != tt.want {
				t.Errorf("ResolveLine() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDiffTasks(t *testing.T) {
	// New format: tags outside parens.
	oldContent := "- @task(review meshx PR | due:2026-05-16) #meshx #pr"
	newContent := "- @task(review meshx PR | due:2026-05-16) #meshx #pr\n- @task fix bug"

	added, removed, unchanged := jot.DiffTasks(oldContent, newContent)

	if len(added) != 1 {
		t.Errorf("added: got %d, want 1", len(added))
	}
	if added[0].Description != "fix bug" {
		t.Errorf("added[0].Description = %q, want %q", added[0].Description, "fix bug")
	}

	if len(removed) != 0 {
		t.Errorf("removed: got %d, want 0", len(removed))
	}

	if len(unchanged) != 1 {
		t.Errorf("unchanged: got %d, want 1", len(unchanged))
	}
}

func TestApplyResolutions(t *testing.T) {
	content := "# Notes\n\n- @task review meshx PR\n- @task fix bug\n\nDone."

	// New format: tags outside parens.
	resolutions := map[int]string{
		3: "@task(review meshx PR | due:2026-05-16) #meshx #pr",
		4: "@task(fix bug | due:2026-05-17)",
	}

	got := jot.ApplyResolutions(content, resolutions)

	wantLine3 := "- @task(review meshx PR | due:2026-05-16) #meshx #pr"
	wantLine4 := "- @task(fix bug | due:2026-05-17)"

	lines := splitLines(got)
	if len(lines) < 4 {
		t.Fatalf("expected at least 4 lines, got %d", len(lines))
	}

	if lines[2] != wantLine3 {
		t.Errorf("line 3 = %q, want %q", lines[2], wantLine3)
	}
	if lines[3] != wantLine4 {
		t.Errorf("line 4 = %q, want %q", lines[3], wantLine4)
	}
}

// splitLines is a local helper so the test has no external dependency for simple splitting.
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	lines = append(lines, s[start:])
	return lines
}
