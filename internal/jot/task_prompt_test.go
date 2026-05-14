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

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/retr0h/jot/internal/jot"
)

func TestTaskPromptInitialState(t *testing.T) {
	m := jot.NewTaskPrompt("review PR", nil)

	if got := m.Description(); got != "review PR" {
		t.Errorf("Description() = %q, want %q", got, "review PR")
	}
	if m.Done() {
		t.Error("Done() = true, want false")
	}
}

func TestTaskPromptSkip(t *testing.T) {
	m := jot.NewTaskPrompt("review PR", nil)

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	tp := result.(jot.TaskPrompt)

	if !tp.Done() {
		t.Error("Done() = false, want true after Escape")
	}
	if !tp.Skipped() {
		t.Error("Skipped() = false, want true after Escape")
	}
}

func TestTaskPromptSave(t *testing.T) {
	m := jot.NewTaskPrompt("review PR", nil)

	// First Enter moves focus from due to labels.
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	tp := result.(jot.TaskPrompt)

	if tp.Done() {
		t.Error("Done() = true after first Enter, want false")
	}

	// Second Enter confirms and saves.
	result2, _ := tp.Update(tea.KeyMsg{Type: tea.KeyEnter})
	tp2 := result2.(jot.TaskPrompt)

	if !tp2.Done() {
		t.Error("Done() = false after second Enter, want true")
	}
	if tp2.Skipped() {
		t.Error("Skipped() = true, want false after save")
	}
}

// Ensure textinput.Blink satisfies tea.Cmd so the import is used.
var _ tea.Cmd = textinput.Blink
