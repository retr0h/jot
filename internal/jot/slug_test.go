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
	"time"

	"github.com/retr0h/jot/internal/jot"
)

func TestNewSlug(t *testing.T) {
	date := time.Date(2026, 5, 14, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name  string
		title string
		want  string
	}{
		{
			name:  "simple lowercase title",
			title: "Standup",
			want:  "2026-05-14-standup",
		},
		{
			name:  "multiple words with spaces",
			title: "Meshx PR Review",
			want:  "2026-05-14-meshx-pr-review",
		},
		{
			name:  "special characters and version number",
			title: "Ideas & Notes: v2.0",
			want:  "2026-05-14-ideas-notes-v2-0",
		},
		{
			name:  "leading and trailing spaces with internal runs",
			title: "  lots   of   spaces  ",
			want:  "2026-05-14-lots-of-spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := jot.NewSlug(tt.title, date)
			if got != tt.want {
				t.Errorf("NewSlug(%q) = %q, want %q", tt.title, got, tt.want)
			}
		})
	}
}
