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
	tests := []struct {
		name    string
		visual  string
		editor  string
		want    string
		wantErr bool
	}{
		{
			name:   "VISUAL takes priority over EDITOR",
			visual: "code",
			editor: "nano",
			want:   "code",
		},
		{
			name:   "EDITOR used when VISUAL is empty",
			visual: "",
			editor: "nano",
			want:   "nano",
		},
		{
			name:    "error when neither is set",
			visual:  "",
			editor:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("VISUAL", tt.visual)
			t.Setenv("EDITOR", tt.editor)

			got, err := jot.EditorName()
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf(
					"EditorName() with VISUAL=%q EDITOR=%q = %q, want %q",
					tt.visual,
					tt.editor,
					got,
					tt.want,
				)
			}
		})
	}
}
