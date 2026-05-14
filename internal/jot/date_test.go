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

func TestParseDate(t *testing.T) {
	// ref is Thursday, 2026-05-14.
	ref := time.Date(2026, 5, 14, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		input   string
		want    string // YYYY-MM-DD, empty when wantErr is true
		wantErr bool
	}{
		{
			name:  "explicit YYYY-MM-DD",
			input: "2026-05-20",
			want:  "2026-05-20",
		},
		{
			name:  "today",
			input: "today",
			want:  "2026-05-14",
		},
		{
			name:  "tomorrow",
			input: "tomorrow",
			want:  "2026-05-15",
		},
		{
			// Thursday → next Friday is 1 day away: May 15.
			name:  "friday from thursday",
			input: "friday",
			want:  "2026-05-15",
		},
		{
			// Thursday → next Monday: (1-4+7)%7 = 4 days: May 18.
			name:  "monday from thursday",
			input: "monday",
			want:  "2026-05-18",
		},
		{
			// "next week" → next Monday from Thursday = May 18.
			name:  "next week",
			input: "next week",
			want:  "2026-05-18",
		},
		{
			name:    "invalid input",
			input:   "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := jot.ParseDate(tt.input, ref)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseDate(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseDate(%q) unexpected error: %v", tt.input, err)
			}
			if got.Format("2006-01-02") != tt.want {
				t.Errorf("ParseDate(%q) = %q, want %q", tt.input, got.Format("2006-01-02"), tt.want)
			}
		})
	}
}
