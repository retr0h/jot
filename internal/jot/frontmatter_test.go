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
	"strings"
	"testing"

	"github.com/retr0h/jot/internal/jot"
)

func TestParseFrontmatter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		content     string
		wantTitle   string
		wantTags    []string
		wantCreated string
		wantSecure  bool
		wantBody    string // substring that must appear in body
	}{
		{
			name: "full frontmatter with title tags and created",
			content: `---
title: "My Note"
tags: [go, testing]
created: 2026-05-14
---

# My Note

Body text here.
`,
			wantTitle:   "My Note",
			wantTags:    []string{"go", "testing"},
			wantCreated: "2026-05-14",
			wantBody:    "Body text here.",
		},
		{
			name: "secure flag is parsed",
			content: `---
title: "Secret"
tags: []
created: 2026-05-14
secure: true
---

private content
`,
			wantTitle:   "Secret",
			wantTags:    []string{},
			wantCreated: "2026-05-14",
			wantSecure:  true,
			wantBody:    "private content",
		},
		{
			name:      "no frontmatter returns original content as body",
			content:   "# Just a Heading\n\nPlain content.",
			wantTitle: "",
			wantTags:  nil,
			wantBody:  "# Just a Heading",
		},
		{
			name:      "empty content returns empty frontmatter and body",
			content:   "",
			wantTitle: "",
			wantTags:  nil,
			wantBody:  "",
		},
		{
			name: "unclosed delimiter treated as no frontmatter",
			content: `---
title: "Unclosed"
`,
			wantTitle: "",
			wantBody:  "---",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fm, body := jot.ParseFrontmatter(tt.content)

			if fm.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", fm.Title, tt.wantTitle)
			}

			if fm.Secure != tt.wantSecure {
				t.Errorf("Secure = %v, want %v", fm.Secure, tt.wantSecure)
			}

			if fm.Created != tt.wantCreated {
				t.Errorf("Created = %q, want %q", fm.Created, tt.wantCreated)
			}

			if len(fm.Tags) != len(tt.wantTags) {
				t.Errorf("Tags = %v, want %v", fm.Tags, tt.wantTags)
			} else {
				for i, tag := range tt.wantTags {
					if fm.Tags[i] != tag {
						t.Errorf("Tags[%d] = %q, want %q", i, fm.Tags[i], tag)
					}
				}
			}

			if tt.wantBody != "" && !strings.Contains(body, tt.wantBody) {
				t.Errorf("body does not contain %q; got: %q", tt.wantBody, body)
			}
		})
	}
}

func TestScaffoldFrontmatter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		title   string
		created string
	}{
		{
			name:    "normal title and date roundtrip",
			title:   "My Sprint Note",
			created: "2026-05-14",
		},
		{
			name:    "title with special characters",
			title:   `Ideas & Plans: v2.0`,
			created: "2026-01-01",
		},
		{
			name:    "empty title produces parseable scaffold",
			title:   "",
			created: "2026-05-14",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			scaffold := jot.ScaffoldFrontmatter(tt.title, tt.created)

			// The scaffold must be parseable back into frontmatter.
			fm, body := jot.ParseFrontmatter(scaffold)

			if fm.Title != tt.title {
				t.Errorf("roundtrip Title = %q, want %q", fm.Title, tt.title)
			}

			if fm.Created != tt.created {
				t.Errorf("roundtrip Created = %q, want %q", fm.Created, tt.created)
			}

			// Body must contain the level-1 heading.
			heading := "# " + tt.title
			if !strings.Contains(body, heading) {
				t.Errorf("scaffold body missing heading %q; got: %q", heading, body)
			}
		})
	}
}
