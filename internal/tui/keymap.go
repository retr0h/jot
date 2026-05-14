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

// Package tui implements the full-screen bubbletea TUI for jot.
package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap holds all key bindings used by the TUI.
type KeyMap struct {
	Up, Down, Left, Right key.Binding
	Toggle, Filter        key.Binding
	Enter, Done, New      key.Binding
	Quit                  key.Binding
}

// DefaultKeyMap returns vim-style key bindings for the TUI.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:     key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k", "up")),
		Down:   key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j", "down")),
		Left:   key.NewBinding(key.WithKeys("h", "left"), key.WithHelp("h", "left pane")),
		Right:  key.NewBinding(key.WithKeys("l", "right"), key.WithHelp("l", "right pane")),
		Toggle: key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "toggle notes/tasks")),
		Filter: key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Enter:  key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open")),
		Done:   key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "mark done")),
		New:    key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new note")),
		Quit:   key.NewBinding(key.WithKeys("q", "esc"), key.WithHelp("q", "quit")),
	}
}
