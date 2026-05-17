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
		wantDesc  string
	}{
		{
			name:      "open checkbox task",
			content:   "- [ ] review meshx PR",
			wantCount: 1,
			wantDesc:  "review meshx PR",
		},
		{
			name:      "done checkbox task",
			content:   "- [x] review meshx PR",
			wantCount: 1,
			wantDesc:  "review meshx PR",
		},
		{
			name:      "task with due date and tags",
			content:   "- [ ] review meshx PR | due:2026-05-16 #meshx #pr",
			wantCount: 1,
			wantDesc:  "review meshx PR",
		},
		{
			name:      "multiple tasks",
			content:   "- [ ] first\n- [ ] second",
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
			content:   "# Title\n\ntext\n\n- [ ] fix bug\n\nDone.",
			wantCount: 1,
			wantDesc:  "fix bug",
		},
		{
			name:      "done task",
			content:   "- [x] reviewed PR",
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

func TestParseTaskWithDueAndTags(t *testing.T) {
	content := "- [ ] review meshx PR | due:2026-05-16 #meshx #pr"
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

	if task.Done {
		t.Error("expected Done = false")
	}
}

func TestParseTaskDone(t *testing.T) {
	content := "- [x] reviewed PR"
	tasks := jot.ParseTasks(content)

	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}

	if !tasks[0].Done {
		t.Error("expected Done = true")
	}
}

func TestParseTaskDoneDate(t *testing.T) {
	content := "- [x] reviewed PR | due:2026-05-16 | done:2026-05-17 #work"
	tasks := jot.ParseTasks(content)

	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}

	task := tasks[0]
	if !task.Done {
		t.Error("expected Done = true")
	}
	if task.DoneDate != "2026-05-17" {
		t.Errorf("DoneDate = %q, want %q", task.DoneDate, "2026-05-17")
	}
	if task.DueDate != "2026-05-16" {
		t.Errorf("DueDate = %q, want %q", task.DueDate, "2026-05-16")
	}
}

func TestParseTaskTags(t *testing.T) {
	content := "- [ ] review meshx PR #meshx #pr"
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

func TestFormatTask(t *testing.T) {
	tests := []struct {
		name string
		task jot.RawTask
		want string
	}{
		{
			name: "with due and labels",
			task: jot.RawTask{
				Description: "review PR",
				DueDate:     "2026-05-16",
				Labels:      []string{"meshx", "pr"},
			},
			want: "- [ ] review PR | due:2026-05-16 #meshx #pr",
		},
		{
			name: "no due",
			task: jot.RawTask{Description: "review PR", Labels: []string{"meshx"}},
			want: "- [ ] review PR #meshx",
		},
		{
			name: "no labels",
			task: jot.RawTask{Description: "review PR", DueDate: "2026-05-16"},
			want: "- [ ] review PR | due:2026-05-16",
		},
		{
			name: "done task",
			task: jot.RawTask{Description: "review PR", Done: true},
			want: "- [x] review PR",
		},
		{
			name: "done with due",
			task: jot.RawTask{
				Description: "review PR",
				DueDate:     "2026-05-16",
				Done:        true,
				Labels:      []string{"work"},
			},
			want: "- [x] review PR | due:2026-05-16 #work",
		},
		{
			name: "done with due and done date",
			task: jot.RawTask{
				Description: "review PR",
				DueDate:     "2026-05-16",
				DoneDate:    "2026-05-17",
				Done:        true,
				Labels:      []string{"work"},
			},
			want: "- [x] review PR | due:2026-05-16 | done:2026-05-17 #work",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := jot.FormatTask(tt.task)
			if got != tt.want {
				t.Errorf("FormatTask() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDiffTasks(t *testing.T) {
	oldContent := "- [ ] review meshx PR | due:2026-05-16 #meshx #pr"
	newContent := "- [ ] review meshx PR | due:2026-05-16 #meshx #pr\n- [ ] fix bug"

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
	content := "# Notes\n\n- [ ] review meshx PR\n- [ ] fix bug\n\nDone."

	resolutions := map[int]string{
		3: "- [x] review meshx PR | due:2026-05-16 #meshx #pr",
		4: "- [x] fix bug",
	}

	got := jot.ApplyResolutions(content, resolutions)

	wantLine3 := "- [x] review meshx PR | due:2026-05-16 #meshx #pr"
	wantLine4 := "- [x] fix bug"

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
