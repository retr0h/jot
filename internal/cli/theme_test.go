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

package cli_test

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/retr0h/jot/internal/cli"
)

func TestActiveTheme(t *testing.T) {
	t.Parallel()

	theme := cli.ActiveTheme()
	if theme == nil {
		t.Fatal("ActiveTheme returned nil")
	}
	if theme.Name != "maxheadroom" {
		t.Errorf("Name = %q, want %q", theme.Name, "maxheadroom")
	}
}

func TestRenderFunctions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		render func(w *bytes.Buffer) string
	}{
		{name: "Mute", render: func(w *bytes.Buffer) string { return cli.Mute(w, "dim text") }},
		{
			name:   "Accent",
			render: func(w *bytes.Buffer) string { return cli.Accent(w, "accent text") },
		},
		{name: "OK", render: func(w *bytes.Buffer) string { return cli.OK(w, "ok text") }},
		{name: "Err", render: func(w *bytes.Buffer) string { return cli.Err(w, "err text") }},
		{name: "Info", render: func(w *bytes.Buffer) string { return cli.Info(w, "info text") }},
		{name: "Tag", render: func(w *bytes.Buffer) string { return cli.Tag(w, "tag text") }},
		{name: "Warn", render: func(w *bytes.Buffer) string { return cli.Warn(w, "warn text") }},
		{name: "Soon", render: func(w *bytes.Buffer) string { return cli.Soon(w, "soon text") }},
		{
			name:   "Overdue",
			render: func(w *bytes.Buffer) string { return cli.Overdue(w, "overdue") },
		},
		{name: "Done", render: func(w *bytes.Buffer) string { return cli.Done(w, "done") }},
		{name: "Hash", render: func(w *bytes.Buffer) string { return cli.Hash(w, "abc1234") }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			got := tt.render(&buf)
			if got == "" {
				t.Error("expected non-empty output")
			}
		})
	}
}

func TestBanner(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	got := cli.Banner(&buf)
	if !strings.Contains(got, "█") {
		t.Error("Banner should contain block characters")
	}
	if !strings.Contains(got, "\n") {
		t.Error("Banner should contain newline")
	}
}

func TestSuccess(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	got := cli.Success(&buf, "created note")
	if got == "" {
		t.Fatal("expected non-empty output")
	}
	if !strings.Contains(got, "created note") {
		t.Errorf("Success = %q, want to contain message", got)
	}
}

func TestFailure(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	got := cli.Failure(&buf, "something broke")
	if got == "" {
		t.Fatal("expected non-empty output")
	}
	if !strings.Contains(got, "something broke") {
		t.Errorf("Failure = %q, want to contain message", got)
	}
}

func TestRenderWithBg(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	theme := cli.ActiveTheme()
	got := cli.RenderWithBg(&buf, theme.Err, theme.RowToday, "overdue task")
	if got == "" {
		t.Fatal("expected non-empty output")
	}
	if !strings.Contains(got, "overdue task") {
		t.Errorf("RenderWithBg = %q, want to contain text", got)
	}
}

func TestRowBgStyles(t *testing.T) {
	t.Parallel()

	today := cli.RowBgToday()
	soon := cli.RowBgSoon()
	if today.GetBackground() == nil {
		t.Error("RowBgToday: expected non-nil background")
	}
	if soon.GetBackground() == nil {
		t.Error("RowBgSoon: expected non-nil background")
	}
}

func TestPad(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		width int
		want  string
	}{
		{
			name:  "pads short string",
			input: "hi",
			width: 5,
			want:  "hi   ",
		},
		{
			name:  "returns string unchanged when at width",
			input: "hello",
			width: 5,
			want:  "hello",
		},
		{
			name:  "returns string unchanged when longer",
			input: "hello world",
			width: 5,
			want:  "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := cli.Pad(tt.input, tt.width)
			if got != tt.want {
				t.Errorf("Pad(%q, %d) = %q, want %q", tt.input, tt.width, got, tt.want)
			}
		})
	}
}

func TestPrint(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	cli.Print(&buf, "hello")
	if buf.String() != "hello\n" {
		t.Errorf("Print wrote %q, want %q", buf.String(), "hello\n")
	}
}

func TestPrintf(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	cli.Printf(&buf, "count: %d", 42)
	if buf.String() != "count: 42" {
		t.Errorf("Printf wrote %q, want %q", buf.String(), "count: 42")
	}
}

func TestRenderFunctionsWithFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		render func(w *os.File) string
	}{
		{name: "Mute", render: func(w *os.File) string { return cli.Mute(w, "file output") }},
		{name: "Accent", render: func(w *os.File) string { return cli.Accent(w, "file accent") }},
		{name: "OK", render: func(w *os.File) string { return cli.OK(w, "ok text") }},
		{name: "Err", render: func(w *os.File) string { return cli.Err(w, "err text") }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f, err := os.CreateTemp(t.TempDir(), "theme-test-*")
			if err != nil {
				t.Fatalf("create temp file: %v", err)
			}
			defer func() { _ = f.Close() }()

			got := tt.render(f)
			if got == "" {
				t.Error("expected non-empty output with *os.File writer")
			}
		})
	}
}
