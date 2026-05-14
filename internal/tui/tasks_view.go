// Copyright (c) 2026 John Dewey

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
// DEALINGS IN THE SOFTWARE.

package tui

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"

	"github.com/retr0h/jot/internal/jot"
)

// taskItem wraps a *jot.Task to satisfy the list.Item interface.
type taskItem struct{ task *jot.Task }

// Title returns the task description.
func (t taskItem) Title() string {
	return t.task.Description
}

// Description returns the due date formatted as "due MM-DD" or "no due date".
func (t taskItem) Description() string {
	if t.task.DueDate != nil {
		return "due " + t.task.DueDate.Format("01-02")
	}
	return "no due date"
}

// FilterValue exposes the description for the bubbles/list fuzzy filter.
func (t taskItem) FilterValue() string {
	return t.task.Description
}

// tasksToItems converts a slice of tasks to the list.Item interface slice
// expected by bubbles/list.
func tasksToItems(tasks []*jot.Task) []list.Item {
	items := make([]list.Item, len(tasks))
	for i, t := range tasks {
		items[i] = taskItem{task: t}
	}
	return items
}

// newTasksList constructs a bubbles/list.Model pre-styled for the tasks pane.
// The delegate uses the pink (#ff6ec7) from the maxheadroom palette.
func newTasksList(tasks []*jot.Task, width, height int) list.Model {
	delegate := list.NewDefaultDelegate()
	pink := lipgloss.Color("#ff6ec7")
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(pink).
		BorderLeftForeground(pink)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(pink).
		BorderLeftForeground(pink)

	l := list.New(tasksToItems(tasks), delegate, width, height)
	l.Title = "Tasks"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.Styles.Title = lipgloss.NewStyle().
		Foreground(pink).
		Bold(true).
		Padding(0, 1)
	return l
}
