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
	"testing"
)

func TestEditArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		path    string
		lineNum string
		want    []string
	}{
		{
			name:    "no line number",
			path:    "/notes/test.md",
			lineNum: "",
			want:    []string{"/notes/test.md"},
		},
		{
			name:    "with line number",
			path:    "/notes/test.md",
			lineNum: "42",
			want:    []string{"+42", "/notes/test.md"},
		},
		{
			name:    "line number 1",
			path:    "/notes/deep/nested.md",
			lineNum: "1",
			want:    []string{"+1", "/notes/deep/nested.md"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := editArgs(tt.path, tt.lineNum)
			if len(got) != len(tt.want) {
				t.Fatalf("editArgs() = %v, want %v", got, tt.want)
			}
			for i, v := range tt.want {
				if got[i] != v {
					t.Errorf("editArgs()[%d] = %q, want %q", i, got[i], v)
				}
			}
		})
	}
}
