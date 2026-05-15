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

import "regexp"

var (
	// wikiLinkRe matches [[slug]] style links.
	wikiLinkRe = regexp.MustCompile(`\[\[([^\]]+)\]\]`)
	// inlineTagRe matches #tag tokens that start at a word boundary.
	inlineTagRe = regexp.MustCompile(`(?:^|\s)#([A-Za-z][A-Za-z0-9_/-]*)`)
)

// ParseLinks extracts unique [[slug]] wiki-links from content.
// The returned slice is in first-seen order with duplicates removed.
func ParseLinks(content string) []string {
	matches := wikiLinkRe.FindAllStringSubmatch(content, -1)
	seen := make(map[string]struct{}, len(matches))
	result := make([]string, 0, len(matches))
	for _, m := range matches {
		slug := m[1]
		if _, ok := seen[slug]; !ok {
			seen[slug] = struct{}{}
			result = append(result, slug)
		}
	}
	return result
}

// ParseInlineTags extracts unique #tag tokens from content.
// Tags must start with a letter and may contain letters, digits, underscores,
// hyphens, and forward slashes. The returned slice is in first-seen order.
func ParseInlineTags(content string) []string {
	matches := inlineTagRe.FindAllStringSubmatch(content, -1)
	seen := make(map[string]struct{}, len(matches))
	result := make([]string, 0, len(matches))
	for _, m := range matches {
		tag := m[1]
		if _, ok := seen[tag]; !ok {
			seen[tag] = struct{}{}
			result = append(result, tag)
		}
	}
	return result
}

// ParseLineTags extracts all #tag tokens from a single line without
// deduplication. Used by the task parser to collect per-line tags.
func ParseLineTags(line string) []string {
	matches := inlineTagRe.FindAllStringSubmatch(line, -1)
	result := make([]string, 0, len(matches))
	for _, m := range matches {
		result = append(result, m[1])
	}
	return result
}
