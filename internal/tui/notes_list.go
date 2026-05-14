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

// noteItem wraps a *jot.Note to satisfy the list.Item interface.
type noteItem struct{ note *jot.Note }

// Title returns the note title with a lock icon appended when the note is secure.
func (n noteItem) Title() string {
	if n.note.Secure {
		return n.note.Title + " \U0001F512"
	}
	return n.note.Title
}

// Description returns the note's updated-at date formatted as YYYY-MM-DD.
func (n noteItem) Description() string {
	return n.note.UpdatedAt.Format("2006-01-02")
}

// FilterValue exposes the title and slug for the bubbles/list fuzzy filter.
func (n noteItem) FilterValue() string {
	return n.note.Title + " " + n.note.Slug
}

// notesToItems converts a slice of notes to the list.Item interface slice
// expected by bubbles/list.
func notesToItems(notes []*jot.Note) []list.Item {
	items := make([]list.Item, len(notes))
	for i, n := range notes {
		items[i] = noteItem{note: n}
	}
	return items
}

// newNotesList constructs a bubbles/list.Model pre-styled for the notes pane.
// The delegate uses the accent orange (#ffb86c) from the maxheadroom palette.
func newNotesList(notes []*jot.Note, width, height int) list.Model {
	delegate := list.NewDefaultDelegate()
	accent := lipgloss.Color("#ffb86c")
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(accent).
		BorderLeftForeground(accent)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(accent).
		BorderLeftForeground(accent)

	l := list.New(notesToItems(notes), delegate, width, height)
	l.Title = "Notes"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.Styles.Title = lipgloss.NewStyle().
		Foreground(accent).
		Bold(true).
		Padding(0, 1)
	return l
}
