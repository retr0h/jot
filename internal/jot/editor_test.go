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

func TestEditorName(t *testing.T) {
	// Not parallel at suite level: subtests manipulate env vars via t.Setenv,
	// which are automatically restored but must not race with each other.

	tests := []struct {
		name         string
		configEditor string
		visual       string
		editor       string
		want         string
	}{
		{
			name:         "config value takes highest priority",
			configEditor: "vim",
			visual:       "code",
			editor:       "nano",
			want:         "vim",
		},
		{
			name:         "VISUAL used when config is empty",
			configEditor: "",
			visual:       "code",
			editor:       "nano",
			want:         "code",
		},
		{
			name:         "EDITOR used when config and VISUAL are empty",
			configEditor: "",
			visual:       "",
			editor:       "nano",
			want:         "nano",
		},
		{
			name:         "defaults to nvim when all sources are empty",
			configEditor: "",
			visual:       "",
			editor:       "",
			want:         "nvim",
		},
		{
			name:         "config overrides even when env vars are set",
			configEditor: "emacs",
			visual:       "",
			editor:       "",
			want:         "emacs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// t.Setenv restores the original value after the test.
			t.Setenv("VISUAL", tt.visual)
			t.Setenv("EDITOR", tt.editor)

			got := jot.EditorName(tt.configEditor)
			if got != tt.want {
				t.Errorf(
					"EditorName(%q) with VISUAL=%q EDITOR=%q = %q, want %q",
					tt.configEditor,
					tt.visual,
					tt.editor,
					got,
					tt.want,
				)
			}
		})
	}
}
