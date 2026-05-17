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
	"strings"
	"testing"

	"github.com/retr0h/jot/internal/jot"
)

func TestParseTasks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		content    string
		wantCount  int
		wantDesc   string
		wantDue    string
		wantDone   bool
		wantDoneAt string
		wantLabels []string
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
			wantDone:  true,
		},
		{
			name:       "task with due date and tags",
			content:    "- [ ] review meshx PR | due:2026-05-16 #meshx #pr",
			wantCount:  1,
			wantDesc:   "review meshx PR",
			wantDue:    "2026-05-16",
			wantLabels: []string{"meshx", "pr"},
		},
		{
			name:       "done task with due and done dates",
			content:    "- [x] reviewed PR | due:2026-05-16 | done:2026-05-17 #work",
			wantCount:  1,
			wantDesc:   "reviewed PR",
			wantDue:    "2026-05-16",
			wantDone:   true,
			wantDoneAt: "2026-05-17",
			wantLabels: []string{"work"},
		},
		{
			name:       "task with tags but no due date",
			content:    "- [ ] review meshx PR #meshx #pr",
			wantCount:  1,
			wantDesc:   "review meshx PR",
			wantLabels: []string{"meshx", "pr"},
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
			name:      "empty content",
			content:   "",
			wantCount: 0,
		},
		{
			name:      "task in middle of document",
			content:   "# Title\n\ntext\n\n- [ ] fix bug\n\nDone.",
			wantCount: 1,
			wantDesc:  "fix bug",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := jot.ParseTasks(tt.content)
			if len(got) != tt.wantCount {
				t.Fatalf(
					"ParseTasks(%q): got %d tasks, want %d",
					tt.content,
					len(got),
					tt.wantCount,
				)
			}
			if tt.wantCount == 0 {
				return
			}

			task := got[0]
			if tt.wantDesc != "" && task.Description != tt.wantDesc {
				t.Errorf("Description = %q, want %q", task.Description, tt.wantDesc)
			}
			if task.Done != tt.wantDone {
				t.Errorf("Done = %v, want %v", task.Done, tt.wantDone)
			}
			if tt.wantDue != "" && task.DueDate != tt.wantDue {
				t.Errorf("DueDate = %q, want %q", task.DueDate, tt.wantDue)
			}
			if tt.wantDoneAt != "" && task.DoneDate != tt.wantDoneAt {
				t.Errorf("DoneDate = %q, want %q", task.DoneDate, tt.wantDoneAt)
			}
			if tt.wantLabels != nil {
				if len(task.Labels) != len(tt.wantLabels) {
					t.Fatalf("Labels = %v, want %v", task.Labels, tt.wantLabels)
				}
				for i, l := range tt.wantLabels {
					if task.Labels[i] != l {
						t.Errorf("Labels[%d] = %q, want %q", i, task.Labels[i], l)
					}
				}
			}
		})
	}
}

func TestFormatTask(t *testing.T) {
	t.Parallel()

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
			name: "done with due and labels",
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
		{
			name: "bare description only",
			task: jot.RawTask{Description: "simple task"},
			want: "- [ ] simple task",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := jot.FormatTask(tt.task)
			if got != tt.want {
				t.Errorf("FormatTask() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDiffTasks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		oldContent    string
		newContent    string
		wantAdded     int
		wantRemoved   int
		wantUnchanged int
		wantAddedDesc string
	}{
		{
			name:          "task added",
			oldContent:    "- [ ] review meshx PR | due:2026-05-16 #meshx #pr",
			newContent:    "- [ ] review meshx PR | due:2026-05-16 #meshx #pr\n- [ ] fix bug",
			wantAdded:     1,
			wantRemoved:   0,
			wantUnchanged: 1,
			wantAddedDesc: "fix bug",
		},
		{
			name:          "task removed",
			oldContent:    "- [ ] first\n- [ ] second",
			newContent:    "- [ ] first",
			wantAdded:     0,
			wantRemoved:   1,
			wantUnchanged: 1,
		},
		{
			name:          "no changes",
			oldContent:    "- [ ] only task",
			newContent:    "- [ ] only task",
			wantAdded:     0,
			wantRemoved:   0,
			wantUnchanged: 1,
		},
		{
			name:          "both empty",
			oldContent:    "no tasks here",
			newContent:    "still no tasks",
			wantAdded:     0,
			wantRemoved:   0,
			wantUnchanged: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			added, removed, unchanged := jot.DiffTasks(tt.oldContent, tt.newContent)

			if len(added) != tt.wantAdded {
				t.Errorf("added: got %d, want %d", len(added), tt.wantAdded)
			}
			if len(removed) != tt.wantRemoved {
				t.Errorf("removed: got %d, want %d", len(removed), tt.wantRemoved)
			}
			if len(unchanged) != tt.wantUnchanged {
				t.Errorf("unchanged: got %d, want %d", len(unchanged), tt.wantUnchanged)
			}
			if tt.wantAddedDesc != "" && len(added) > 0 {
				if added[0].Description != tt.wantAddedDesc {
					t.Errorf(
						"added[0].Description = %q, want %q",
						added[0].Description,
						tt.wantAddedDesc,
					)
				}
			}
		})
	}
}

func TestApplyResolutions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		content     string
		resolutions map[int]string
		wantLines   map[int]string
	}{
		{
			name:    "replace multiple lines",
			content: "# Notes\n\n- [ ] review meshx PR\n- [ ] fix bug\n\nDone.",
			resolutions: map[int]string{
				3: "- [x] review meshx PR | due:2026-05-16 #meshx #pr",
				4: "- [x] fix bug",
			},
			wantLines: map[int]string{
				3: "- [x] review meshx PR | due:2026-05-16 #meshx #pr",
				4: "- [x] fix bug",
			},
		},
		{
			name:        "out of bounds line numbers are ignored",
			content:     "line one\nline two",
			resolutions: map[int]string{99: "nope", -1: "nope"},
			wantLines: map[int]string{
				1: "line one",
				2: "line two",
			},
		},
		{
			name:        "empty resolutions returns content unchanged",
			content:     "# Title\n\nbody",
			resolutions: map[int]string{},
			wantLines: map[int]string{
				1: "# Title",
				3: "body",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := jot.ApplyResolutions(tt.content, tt.resolutions)
			lines := strings.Split(got, "\n")

			for lineNum, want := range tt.wantLines {
				idx := lineNum - 1
				if idx < 0 || idx >= len(lines) {
					t.Errorf("line %d: out of bounds (got %d lines)", lineNum, len(lines))
					continue
				}
				if lines[idx] != want {
					t.Errorf("line %d = %q, want %q", lineNum, lines[idx], want)
				}
			}
		})
	}
}
