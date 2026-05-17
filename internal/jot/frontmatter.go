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
	"errors"
	"fmt"
	"strings"

	"github.com/adrg/frontmatter"
)

// Frontmatter holds the YAML front-matter block parsed from a note file.
// All fields map directly to the YAML keys used in jot's scaffold template.
type Frontmatter struct {
	Title   string   `yaml:"title"`
	Tags    []string `yaml:"tags"`
	Created string   `yaml:"created,omitempty"`
	Secure  bool     `yaml:"secure,omitempty"`
}

// ParseFrontmatter parses the YAML front-matter from content and returns the
// decoded Frontmatter alongside the remaining body text. When no front-matter
// is present the original content is returned unchanged.
func ParseFrontmatter(content string) (Frontmatter, string) {
	var fm Frontmatter
	body, err := frontmatter.Parse(strings.NewReader(content), &fm)
	if err != nil {
		if errors.Is(err, frontmatter.ErrNotFound) {
			return fm, content
		}
		return fm, content
	}
	fm.Tags = normalizeTags(fm.Tags)
	return fm, string(body)
}

// normalizeTags replaces spaces with hyphens in each tag.
func normalizeTags(tags []string) []string {
	for i, t := range tags {
		tags[i] = strings.ReplaceAll(strings.TrimSpace(t), " ", "-")
	}
	return tags
}

// ScaffoldFrontmatter returns a complete note scaffold: a YAML front-matter
// block followed by a level-1 heading, ready for the editor to open.
func ScaffoldFrontmatter(title string, created string) string {
	return fmt.Sprintf(
		"---\ntitle: %q\ntags: []\ncreated: %s\n---\n\n# %s\n\n",
		title,
		created,
		title,
	)
}
