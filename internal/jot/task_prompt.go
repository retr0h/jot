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
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type TaskPrompt struct {
	description string
	dueInput    textinput.Model
	labelInput  textinput.Model
	focusDue    bool
	done        bool
	skipped     bool
}

func NewTaskPrompt(description string, existingLabels []string) TaskPrompt {
	due := textinput.New()
	due.Placeholder = "due date (friday, 2026-05-20, ...)"
	due.Focus()
	due.CharLimit = 64

	lbl := textinput.New()
	lbl.Placeholder = "labels (space-separated)"
	lbl.CharLimit = 128

	return TaskPrompt{
		description: description,
		dueInput:    due,
		labelInput:  lbl,
		focusDue:    true,
	}
}

func (m TaskPrompt) Description() string { return m.description }
func (m TaskPrompt) Done() bool          { return m.done }
func (m TaskPrompt) Skipped() bool       { return m.skipped }
func (m TaskPrompt) DueValue() string    { return strings.TrimSpace(m.dueInput.Value()) }

func (m TaskPrompt) LabelValues() []string {
	raw := strings.TrimSpace(m.labelInput.Value())
	if raw == "" {
		return nil
	}
	var out []string
	for _, l := range strings.Fields(raw) {
		if l != "" {
			out = append(out, l)
		}
	}
	return out
}

func (m TaskPrompt) Init() tea.Cmd {
	return textinput.Blink
}

func (m TaskPrompt) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEscape:
			m.done = true
			m.skipped = true
			return m, tea.Quit
		case tea.KeyEnter:
			if m.focusDue {
				m.focusDue = false
				m.dueInput.Blur()
				m.labelInput.Focus()
				return m, textinput.Blink
			}
			m.done = true
			return m, tea.Quit
		case tea.KeyTab, tea.KeyShiftTab:
			m.focusDue = !m.focusDue
			if m.focusDue {
				m.labelInput.Blur()
				m.dueInput.Focus()
			} else {
				m.dueInput.Blur()
				m.labelInput.Focus()
			}
			return m, textinput.Blink
		}
	}

	var cmd tea.Cmd
	if m.focusDue {
		m.dueInput, cmd = m.dueInput.Update(msg)
	} else {
		m.labelInput, cmd = m.labelInput.Update(msg)
	}
	return m, cmd
}

func (m TaskPrompt) View() string {
	accent := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffb86c"))
	muted := lipgloss.NewStyle().Faint(true)

	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", accent.Render("┌─ New Task ─────────────────────────────┐"))
	fmt.Fprintf(&b, "│ %s\n", m.description)
	fmt.Fprintf(&b, "│\n")
	fmt.Fprintf(&b, "│ Due:    %s\n", m.dueInput.View())
	fmt.Fprintf(&b, "│ Labels: %s\n", m.labelInput.View())
	fmt.Fprintf(&b, "│\n")
	fmt.Fprintf(&b, "│ %s\n", muted.Render("Enter=save  Esc=skip  Tab=next field"))
	fmt.Fprintf(&b, "%s\n", accent.Render("└────────────────────────────────────────┘"))
	return b.String()
}
