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

func TestParseLinks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name:    "single wiki link",
			content: "See [[other-note]] for context.",
			want:    []string{"other-note"},
		},
		{
			name:    "multiple distinct links in order",
			content: "Refs: [[alpha]], [[beta]], [[gamma]].",
			want:    []string{"alpha", "beta", "gamma"},
		},
		{
			name:    "duplicate links are deduplicated keeping first-seen order",
			content: "[[foo]] and [[bar]] and [[foo]] again.",
			want:    []string{"foo", "bar"},
		},
		{
			name:    "no links returns nil slice",
			content: "No links here, just plain text.",
			want:    nil,
		},
		{
			name:    "empty content returns nil slice",
			content: "",
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := jot.ParseLinks(tt.content)

			if len(got) != len(tt.want) {
				t.Errorf("ParseLinks(%q) = %v, want %v", tt.content, got, tt.want)
				return
			}
			for i, link := range tt.want {
				if got[i] != link {
					t.Errorf("ParseLinks[%d] = %q, want %q", i, got[i], link)
				}
			}
		})
	}
}

func TestParseInlineTags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name:    "single tag",
			content: "This is #go related.",
			want:    []string{"go"},
		},
		{
			name:    "multiple distinct tags in order",
			content: "Topics: #go #testing #ci.",
			want:    []string{"go", "testing", "ci"},
		},
		{
			name:    "duplicate tags are deduplicated keeping first-seen order",
			content: "#go code and more #testing and #go again.",
			want:    []string{"go", "testing"},
		},
		{
			name:    "no tags returns nil slice",
			content: "No hash tokens here.",
			want:    nil,
		},
		{
			name:    "empty content returns nil slice",
			content: "",
			want:    nil,
		},
		{
			name:    "tag with hyphen and slash",
			content: "Work on #code-review and #proj/alpha today.",
			want:    []string{"code-review", "proj/alpha"},
		},
		{
			name:    "hash inside a word without preceding space is ignored",
			content: "color#ff0000 is not a tag but #real is.",
			want:    []string{"real"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := jot.ParseInlineTags(tt.content)

			if len(got) != len(tt.want) {
				t.Errorf("ParseInlineTags(%q) = %v, want %v", tt.content, got, tt.want)
				return
			}
			for i, tag := range tt.want {
				if got[i] != tag {
					t.Errorf("ParseInlineTags[%d] = %q, want %q", i, got[i], tag)
				}
			}
		})
	}
}

func TestParseLineTags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		line string
		want []string
	}{
		{
			name: "single tag on line",
			line: "@task do something #work",
			want: []string{"work"},
		},
		{
			name: "multiple tags on single line",
			line: "@task do something #go #testing #ci",
			want: []string{"go", "testing", "ci"},
		},
		{
			name: "no deduplication — returns all occurrences",
			line: "#go stuff and #go again",
			want: []string{"go", "go"},
		},
		{
			name: "no tags on line returns nil slice",
			line: "just plain text",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := jot.ParseLineTags(tt.line)

			if len(got) != len(tt.want) {
				t.Errorf("ParseLineTags(%q) = %v, want %v", tt.line, got, tt.want)
				return
			}
			for i, tag := range tt.want {
				if got[i] != tag {
					t.Errorf("ParseLineTags[%d] = %q, want %q", i, got[i], tag)
				}
			}
		})
	}
}
