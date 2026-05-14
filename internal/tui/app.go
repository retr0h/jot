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
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/retr0h/jot/internal/jot"
)

// mode selects whether the list pane shows notes or tasks.
type mode int

const (
	modeNotes mode = iota
	modeTasks
)

// pane selects which panel has keyboard focus.
type pane int

const (
	paneList pane = iota
	panePreview
)

// editorFinishedMsg is sent by tea.ExecProcess once the editor process exits.
type editorFinishedMsg struct{ err error }

// listWidthFraction is the portion of the terminal width given to the list pane.
const listWidthFraction = 0.35

// borderColor is the inactive pane border color from the Tokyo Night palette.
const borderColor = "#3b4261"

// Model is the top-level bubbletea model for the jot TUI.
type Model struct {
	store    *jot.Store
	notesDir string
	keys     KeyMap

	notesList list.Model
	tasksList list.Model
	preview   viewport.Model

	mode  mode
	focus pane

	width, height int
	quitting      bool
}

// New constructs a TUI Model ready for use with tea.NewProgram.
func New(store *jot.Store, notesDir string) Model {
	m := Model{
		store:    store,
		notesDir: notesDir,
		keys:     DefaultKeyMap(),
		mode:     modeNotes,
		focus:    paneList,
	}
	return m
}

// Init satisfies tea.Model. It requests the alternate screen so the TUI takes
// over the full terminal without disturbing the scroll-back buffer.
func (m Model) Init() tea.Cmd {
	return tea.EnterAltScreen
}

// Update satisfies tea.Model and routes all messages to the appropriate handler.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m = m.reload()
		return m, nil

	case editorFinishedMsg:
		// Reload to pick up any changes the editor made to the note.
		m = m.reload()
		return m, tea.EnterAltScreen

	case tea.KeyMsg:
		// When the list pane is in filter mode, let it consume all keystrokes
		// so the user can type freely without triggering TUI bindings.
		if m.focus == paneList && m.isFiltering() {
			return m.delegateToList(msg)
		}

		switch {
		case msg.String() == "q" || msg.String() == "esc":
			m.quitting = true
			return m, tea.Quit

		case msg.String() == "t":
			if m.mode == modeNotes {
				m.mode = modeTasks
			} else {
				m.mode = modeNotes
			}
			m = m.reload()
			return m, nil

		case msg.String() == "h" || msg.String() == "left":
			m.focus = paneList
			return m, nil

		case msg.String() == "l" || msg.String() == "right":
			m.focus = panePreview
			return m, nil

		case msg.String() == "enter":
			if m.focus == paneList && m.mode == modeNotes {
				if item, ok := m.notesList.SelectedItem().(noteItem); ok {
					path := filepath.Join(m.notesDir, item.note.Slug+".md")
					cmd := jot.EditorCmd(path)
					return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
						return editorFinishedMsg{err: err}
					})
				}
			}
			return m, nil

		case msg.String() == "d":
			if m.focus == paneList && m.mode == modeTasks {
				if item, ok := m.tasksList.SelectedItem().(taskItem); ok {
					_ = m.store.CompleteTask(item.task.ID)
					m = m.reload()
				}
			}
			return m, nil

		case msg.String() == "j" || msg.String() == "down":
			return m.handleDown()

		case msg.String() == "k" || msg.String() == "up":
			return m.handleUp()
		}

	}

	// For any unhandled message, delegate to the focused component.
	return m.delegateFocused(msg)
}

// View satisfies tea.Model and renders the split-pane layout.
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	listWidth := int(float64(m.width) * listWidthFraction)
	previewWidth := m.width - listWidth

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(borderColor))

	// Inner dimensions account for the border (1 cell each side).
	listInner := listWidth - 2
	previewInner := previewWidth - 2
	innerHeight := m.height - 2

	// Resize list and preview to their inner dimensions before rendering.
	activeList := m.activeList()
	activeList.SetSize(listInner, innerHeight)

	vp := m.preview
	vp.Width = previewInner
	vp.Height = innerHeight

	listPane := border.Width(listWidth - 2).Height(innerHeight).Render(activeList.View())
	previewPane := border.Width(previewWidth - 2).Height(innerHeight).Render(vp.View())

	return lipgloss.JoinHorizontal(lipgloss.Top, listPane, previewPane)
}

// reload re-fetches data from the store and rebuilds both list and preview
// models using the current terminal dimensions.
func (m Model) reload() Model {
	listWidth := int(float64(m.width) * listWidthFraction)
	previewWidth := m.width - listWidth
	// Subtract 2 for border on each pane side.
	listInner := listWidth - 2
	previewInner := previewWidth - 2
	innerHeight := m.height - 2

	switch m.mode {
	case modeNotes:
		notes, _ := m.store.ListNotes("")
		m.notesList = newNotesList(notes, listInner, innerHeight)
	case modeTasks:
		tasks, _ := m.store.ListTasks("open", "")
		m.tasksList = newTasksList(tasks, listInner, innerHeight)
	}

	m.preview = newPreview(m.previewContent(), previewInner, innerHeight)
	return m
}

// updatePreviewFromSelection reads the currently selected note's markdown file
// and re-renders the preview pane.
func (m *Model) updatePreviewFromSelection() {
	previewWidth := m.width - int(float64(m.width)*listWidthFraction)
	previewInner := previewWidth - 2
	innerHeight := m.height - 2

	content := m.previewContent()
	m.preview = newPreview(content, previewInner, innerHeight)
}

// previewContent returns the raw markdown content for the currently selected
// list item, or an empty string when nothing is selected or the mode is tasks.
func (m Model) previewContent() string {
	if m.mode != modeNotes {
		return ""
	}
	item, ok := m.notesList.SelectedItem().(noteItem)
	if !ok {
		return ""
	}
	path := filepath.Join(m.notesDir, item.note.Slug+".md")
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// activeList returns a pointer to the list model that matches the current mode.
func (m Model) activeList() list.Model {
	if m.mode == modeNotes {
		return m.notesList
	}
	return m.tasksList
}

// isFiltering returns true when the active list is in filter/search mode.
func (m Model) isFiltering() bool {
	return m.activeList().FilterState() == list.Filtering
}

// handleDown moves the selection down in the focused component.
func (m Model) handleDown() (tea.Model, tea.Cmd) {
	if m.focus == panePreview {
		m.preview.LineDown(1)
		return m, nil
	}
	return m.delegateToList(tea.KeyMsg{Type: tea.KeyDown})
}

// handleUp moves the selection up in the focused component.
func (m Model) handleUp() (tea.Model, tea.Cmd) {
	if m.focus == panePreview {
		m.preview.LineUp(1)
		return m, nil
	}
	return m.delegateToList(tea.KeyMsg{Type: tea.KeyUp})
}

// delegateToList forwards a message to the active list model and refreshes the
// preview pane when the selection changes.
func (m Model) delegateToList(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if m.mode == modeNotes {
		m.notesList, cmd = m.notesList.Update(msg)
	} else {
		m.tasksList, cmd = m.tasksList.Update(msg)
	}
	m.updatePreviewFromSelection()
	return m, cmd
}

// delegateFocused forwards an unhandled message to whichever component
// currently has focus.
func (m Model) delegateFocused(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if m.focus == paneList {
		if m.mode == modeNotes {
			m.notesList, cmd = m.notesList.Update(msg)
		} else {
			m.tasksList, cmd = m.tasksList.Update(msg)
		}
		m.updatePreviewFromSelection()
	} else {
		m.preview, cmd = m.preview.Update(msg)
	}
	return m, cmd
}
