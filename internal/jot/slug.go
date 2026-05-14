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
	"regexp"
	"strings"
	"time"
)

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

// NewSlug returns a URL-safe slug combining the date and a normalized title.
// The date is formatted as YYYY-MM-DD and the title is lowercased with
// non-alphanumeric runs replaced by hyphens.
func NewSlug(title string, date time.Time) string {
	s := strings.ToLower(strings.TrimSpace(title))
	s = nonAlnum.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return date.Format("2006-01-02") + "-" + s
}
