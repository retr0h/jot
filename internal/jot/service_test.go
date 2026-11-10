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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/retr0h/jot/internal/gitops"
	"github.com/retr0h/jot/internal/jot"
)

// ─── CatNote ────────────────────────────────────────────────────────────────

func TestCatNote(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		slug    string
		setup   func(t *testing.T, dir string)
		wantSub string
		wantErr bool
	}{
		{
			name: "returns body of plaintext note",
			slug: "my-note",
			setup: func(t *testing.T, dir string) {
				writeNote(t, dir, "my-note", `---
title: "My Note"
tags: []
created: 2026-05-14
---

# My Note

Hello world.
`)
			},
			wantSub: "Hello world.",
		},
		{
			name:    "missing note returns error",
			slug:    "no-such-note",
			setup:   func(_ *testing.T, _ string) {},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			tt.setup(t, dir)
			svc := newSvc(dir)

			got, err := svc.CatNote(tt.slug)
			if tt.wantErr {
				if err == nil {
					t.Fatal("CatNote: expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("CatNote: unexpected error: %v", err)
			}
			if !strings.Contains(got, tt.wantSub) {
				t.Errorf("CatNote = %q, want substring %q", got, tt.wantSub)
			}
		})
	}
}

// ─── EncryptNote ────────────────────────────────────────────────────────────

const (
	plainNote = "---\ntitle: \"Plain\"\ndescription: \"Deploy keys\"\nowner: john\n" +
		"tags: [ops]\ncreated: 2026-05-14\n---\n\nSecret body.\n"
	crlfNote = "---\r\ntitle: \"CRLF\"\r\ntags: []\r\n---\r\n\r\nWindows body.\r\n"
)

func TestEncryptNote(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		slug     string
		setup    func(t *testing.T, svc *jot.Service)
		wantFile []string // substrings of the file after encrypting
		wantErr  bool
	}{
		{
			name: "moves the body into the vault and keeps every front-matter field",
			slug: "plain",
			setup: func(t *testing.T, svc *jot.Service) {
				writeNote(t, svc.NotesDir, "plain", plainNote)
			},
			wantFile: []string{
				"description: \"Deploy keys\"\n",
				"owner: john\n",
				"tags: [ops]\n",
				"secure: true\n---\n",
			},
		},
		{
			name: "a note without front-matter gets one",
			slug: "bare",
			setup: func(t *testing.T, svc *jot.Service) {
				writeNote(t, svc.NotesDir, "bare", "# Bare\n\nSecret body.\n")
			},
			wantFile: []string{"title: \"Bare\"\n", "secure: true\n---\n"},
		},
		{
			name: "a note in a folder, whose slug kvlt cannot take as a key",
			slug: "ops/plain",
			setup: func(t *testing.T, svc *jot.Service) {
				if err := os.MkdirAll(filepath.Join(svc.NotesDir, "ops"), 0o700); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				writeNote(t, svc.NotesDir, "ops/plain", plainNote)
			},
			wantFile: []string{"owner: john\n", "secure: true\n---\n"},
		},
		{
			name: "already encrypted note returns error",
			slug: "plain",
			setup: func(t *testing.T, svc *jot.Service) {
				writeNote(t, svc.NotesDir, "plain", plainNote)
				if err := svc.EncryptNote("plain"); err != nil {
					t.Fatalf("setup encrypt: %v", err)
				}
			},
			wantErr: true,
		},
		{
			name:    "missing note returns error",
			slug:    "no-such-note",
			setup:   func(_ *testing.T, _ *jot.Service) {},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := newSecureSvc(t, t.TempDir())
			tt.setup(t, svc)

			err := svc.EncryptNote(tt.slug)
			if tt.wantErr {
				if err == nil {
					t.Fatal("EncryptNote: expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("EncryptNote: unexpected error: %v", err)
			}

			data, err := os.ReadFile(filepath.Join(svc.NotesDir, tt.slug+".md"))
			if err != nil {
				t.Fatalf("read note: %v", err)
			}
			for _, sub := range tt.wantFile {
				if !strings.Contains(string(data), sub) {
					t.Errorf("file after encrypt = %q, want substring %q", data, sub)
				}
			}
			if strings.Contains(string(data), "Secret body.") {
				t.Errorf("file after encrypt still holds the body: %q", data)
			}
			body, err := svc.CatNote(tt.slug)
			if err != nil || !strings.Contains(body, "Secret body.") {
				t.Errorf("CatNote after encrypt = %q, %v", body, err)
			}
		})
	}
}

// ─── DecryptNote ────────────────────────────────────────────────────────────

func TestDecryptNote(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		slug     string
		setup    func(t *testing.T, svc *jot.Service) string // returns the slug when created here
		wantFile string                                      // the whole file after decrypting
		wantErr  bool
	}{
		{
			name: "round-trips an encrypted note byte for byte",
			slug: "plain",
			setup: func(t *testing.T, svc *jot.Service) string {
				writeNote(t, svc.NotesDir, "plain", plainNote)
				if err := svc.EncryptNote("plain"); err != nil {
					t.Fatalf("setup encrypt: %v", err)
				}
				return "plain"
			},
			wantFile: plainNote,
		},
		{
			name: "a note created secure in a folder gets a blank line before its body",
			setup: func(t *testing.T, svc *jot.Service) string {
				slug, _, err := svc.CreateNote("ops/Vault", "Top secret.", true)
				if err != nil {
					t.Fatalf("setup create: %v", err)
				}
				return slug
			},
			wantFile: "\n---\n\nTop secret.",
		},
		{
			name: "CRLF line endings round-trip byte for byte",
			slug: "crlf",
			setup: func(t *testing.T, svc *jot.Service) string {
				writeNote(t, svc.NotesDir, "crlf", crlfNote)
				if err := svc.EncryptNote("crlf"); err != nil {
					t.Fatalf("setup encrypt: %v", err)
				}
				return "crlf"
			},
			wantFile: crlfNote,
		},
		{
			name: "a quoted secure key is removed too",
			slug: "plain",
			setup: func(t *testing.T, svc *jot.Service) string {
				path := writeNote(t, svc.NotesDir, "plain", plainNote)
				if err := svc.EncryptNote("plain"); err != nil {
					t.Fatalf("setup encrypt: %v", err)
				}
				data, _ := os.ReadFile(path)
				quoted := strings.Replace(string(data), "secure: true", `"secure": true`, 1)
				if err := os.WriteFile(path, []byte(quoted), 0o600); err != nil {
					t.Fatalf("setup quote: %v", err)
				}
				return "plain"
			},
			wantFile: plainNote,
		},
		{
			name: "plaintext note returns error",
			slug: "plain",
			setup: func(t *testing.T, svc *jot.Service) string {
				writeNote(t, svc.NotesDir, "plain", plainNote)
				return ""
			},
			wantErr: true,
		},
		{
			name:    "missing note returns error",
			slug:    "no-such-note",
			setup:   func(_ *testing.T, _ *jot.Service) string { return "" },
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := newSecureSvc(t, t.TempDir())
			slug := tt.slug
			if created := tt.setup(t, svc); slug == "" {
				slug = created
			}

			err := svc.DecryptNote(slug)
			if tt.wantErr {
				if err == nil {
					t.Fatal("DecryptNote: expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("DecryptNote: unexpected error: %v", err)
			}

			data, err := os.ReadFile(filepath.Join(svc.NotesDir, slug+".md"))
			if err != nil {
				t.Fatalf("read note: %v", err)
			}
			if strings.Contains(string(data), "secure") ||
				!strings.Contains(string(data), tt.wantFile) {
				t.Errorf("file after decrypt = %q, want it to contain %q", data, tt.wantFile)
			}
		})
	}
}

// ─── AppendNote ─────────────────────────────────────────────────────────────

func TestAppendNote(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		slug     string
		text     string
		secure   bool
		setup    func(t *testing.T, svc *jot.Service)
		wantBody string // CatNote after the append
		wantErr  bool
	}{
		{
			name: "appends under the front-matter, adding a missing newline",
			slug: "mem",
			text: "Prefers squash merges.\n\n",
			setup: func(t *testing.T, svc *jot.Service) {
				writeNote(
					t,
					svc.NotesDir,
					"mem",
					"---\ntitle: \"Mem\"\ntags: []\n---\n\n# Mem\n\nUses jot.",
				)
			},
			wantBody: "\n# Mem\n\nUses jot.\nPrefers squash merges.\n",
		},
		{
			name: "appends to a note without front-matter",
			slug: "bare",
			text: "second",
			setup: func(t *testing.T, svc *jot.Service) {
				writeNote(t, svc.NotesDir, "bare", "first\n")
			},
			wantBody: "first\nsecond\n",
		},
		{
			name: "appends to an empty body",
			slug: "empty",
			text: "only line",
			setup: func(t *testing.T, svc *jot.Service) {
				writeNote(t, svc.NotesDir, "empty", "---\ntitle: \"Empty\"\ntags: []\n---\n")
			},
			wantBody: "\nonly line\n",
		},
		{
			name:   "appends to a secure note in the vault",
			slug:   "vault",
			text:   "more",
			secure: true,
			setup: func(t *testing.T, svc *jot.Service) {
				writeNote(
					t,
					svc.NotesDir,
					"vault",
					"---\ntitle: \"Vault\"\ntags: []\n---\n\nsecret\n",
				)
				if err := svc.EncryptNote("vault"); err != nil {
					t.Fatalf("setup encrypt: %v", err)
				}
			},
			wantBody: "\nsecret\nmore\n",
		},
		{
			name:    "blank text returns error",
			slug:    "bare",
			text:    " \n",
			setup:   func(_ *testing.T, _ *jot.Service) {},
			wantErr: true,
		},
		{
			name:    "missing note returns error",
			slug:    "no-such-note",
			text:    "x",
			setup:   func(_ *testing.T, _ *jot.Service) {},
			wantErr: true,
		},
		{
			name: "secure note without a vault returns error",
			slug: "orphan",
			text: "x",
			setup: func(t *testing.T, svc *jot.Service) {
				writeNote(
					t,
					svc.NotesDir,
					"orphan",
					"---\ntitle: \"O\"\ntags: []\nsecure: true\n---\n",
				)
			},
			wantErr: true,
		},
		{
			name:   "secure note missing from the vault returns error",
			slug:   "orphan",
			text:   "x",
			secure: true,
			setup: func(t *testing.T, svc *jot.Service) {
				writeNote(
					t,
					svc.NotesDir,
					"orphan",
					"---\ntitle: \"O\"\ntags: []\nsecure: true\n---\n",
				)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := newSvc(t.TempDir())
			if tt.secure {
				svc = newSecureSvc(t, t.TempDir())
			}
			tt.setup(t, svc)

			err := svc.AppendNote(tt.slug, tt.text)
			if tt.wantErr {
				if err == nil {
					t.Fatal("AppendNote: expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("AppendNote: unexpected error: %v", err)
			}
			if body, err := svc.CatNote(tt.slug); err != nil || body != tt.wantBody {
				t.Errorf("body after append = %q, %v; want %q", body, err, tt.wantBody)
			}
		})
	}
}

// ─── UpdateNote ─────────────────────────────────────────────────────────────

func TestUpdateNote(t *testing.T) {
	t.Parallel()

	const head = "---\ntitle: \"Mem\"\nowner: john\ntags: []\n---\n"

	tests := []struct {
		name     string
		slug     string
		content  string
		secure   bool
		setup    func(t *testing.T, svc *jot.Service)
		wantFile string // the whole file after the update, when set
		wantBody string // CatNote after the update, when set
		wantErr  bool
	}{
		{
			name:    "replaces the body, keeps the front-matter and adds the blank line after it",
			slug:    "mem",
			content: "New body",
			setup: func(t *testing.T, svc *jot.Service) {
				writeNote(t, svc.NotesDir, "mem", head+"\nOld body\n")
			},
			wantFile: head + "\nNew body\n",
		},
		{
			name:    "an unclosed --- block is body text, not front-matter",
			slug:    "mem",
			content: "---\nnot front-matter\n",
			setup: func(t *testing.T, svc *jot.Service) {
				writeNote(t, svc.NotesDir, "mem", head+"\nOld body\n")
			},
			wantFile: head + "\n---\nnot front-matter\n",
		},
		{
			name:    "content with its own front-matter replaces the whole file",
			slug:    "mem",
			content: "---\ntitle: \"Mem\"\ndescription: \"Agent memory\"\ntags: []\n---\n\nAll new\n",
			setup: func(t *testing.T, svc *jot.Service) {
				writeNote(t, svc.NotesDir, "mem", head+"\nOld body\n")
			},
			wantFile: "---\ntitle: \"Mem\"\ndescription: \"Agent memory\"\ntags: []\n---\n\nAll new\n",
		},
		{
			name:    "content that sets secure: true returns error",
			slug:    "mem",
			content: "---\ntitle: \"Mem\"\nsecure: true\n---\n\nplaintext secret\n",
			setup: func(t *testing.T, svc *jot.Service) {
				writeNote(t, svc.NotesDir, "mem", head+"\nOld body\n")
			},
			wantErr: true,
		},
		{
			name:    "updates a secure note's body in the vault",
			slug:    "vault",
			content: "rotated",
			secure:  true,
			setup: func(t *testing.T, svc *jot.Service) {
				writeNote(t, svc.NotesDir, "vault", "---\ntitle: \"Vault\"\ntags: []\n---\n\nold\n")
				if err := svc.EncryptNote("vault"); err != nil {
					t.Fatalf("setup encrypt: %v", err)
				}
			},
			wantBody: "rotated",
		},
		{
			name:    "replacing a secure note's front-matter returns error",
			slug:    "vault",
			content: "---\ntitle: \"Vault\"\ntags: []\n---\n\nplain now\n",
			secure:  true,
			setup: func(t *testing.T, svc *jot.Service) {
				writeNote(t, svc.NotesDir, "vault", "---\ntitle: \"Vault\"\ntags: []\n---\n\nold\n")
				if err := svc.EncryptNote("vault"); err != nil {
					t.Fatalf("setup encrypt: %v", err)
				}
			},
			wantErr: true,
		},
		{
			name:    "missing note returns error",
			slug:    "no-such-note",
			content: "x",
			setup:   func(_ *testing.T, _ *jot.Service) {},
			wantErr: true,
		},
		{
			name:    "missing note with whole-file content returns error",
			slug:    "no-such-note",
			content: head,
			setup:   func(_ *testing.T, _ *jot.Service) {},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := newSvc(t.TempDir())
			if tt.secure {
				svc = newSecureSvc(t, t.TempDir())
			}
			tt.setup(t, svc)

			err := svc.UpdateNote(tt.slug, tt.content)
			if tt.wantErr {
				if err == nil {
					t.Fatal("UpdateNote: expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("UpdateNote: unexpected error: %v", err)
			}
			if tt.wantFile != "" {
				data, err := os.ReadFile(filepath.Join(svc.NotesDir, tt.slug+".md"))
				if err != nil || string(data) != tt.wantFile {
					t.Errorf("file after update = %q, %v; want %q", data, err, tt.wantFile)
				}
			}
			if tt.wantBody != "" {
				if body, err := svc.CatNote(tt.slug); err != nil || body != tt.wantBody {
					t.Errorf("body after update = %q, %v; want %q", body, err, tt.wantBody)
				}
			}
		})
	}
}

// ─── CreateNote ─────────────────────────────────────────────────────────────

func TestCreateNote(t *testing.T) {
	t.Parallel()

	today := time.Now().Format("2006-01-02")
	defaults := "title: \"Jot\"\ntags: []\ncreated: " + today + "\n"

	tests := []struct {
		name        string
		title       string
		content     string
		secure      bool
		dir         func(t *testing.T) string
		before      func(t *testing.T, svc *jot.Service)
		wantSlugSub string
		wantBodySub string
		wantFile    string // the whole file, when set
		wantErr     bool
	}{
		{
			name:        "dir/Title creates the note in that folder",
			title:       "ops/Runbook",
			wantSlugSub: "ops/",
			wantBodySub: "title: \"Runbook\"",
		},
		{
			name:        "names the slug from the title alone, without a date",
			title:       "memory/Jot",
			wantSlugSub: "memory/jot",
			wantBodySub: defaults,
		},
		{
			name:  "an existing note in a folder is not overwritten",
			title: "memory/Jot",
			before: func(t *testing.T, svc *jot.Service) {
				if _, _, err := svc.CreateNote("memory/Jot", "first", false); err != nil {
					t.Fatalf("setup create: %v", err)
				}
			},
			wantErr: true,
		},
		{
			name:    "a title with no letters or digits returns error",
			title:   "memory/!!!",
			wantErr: true,
		},
		{
			name:     "supplied front-matter gains the missing defaults",
			title:    "Jot",
			content:  "---\ndescription: \"Agent memory\"\n---\n\nbody\n",
			wantFile: "---\n" + defaults + "description: \"Agent memory\"\n---\n\nbody\n",
		},
		{
			name:     "supplied fields win over the defaults",
			title:    "Jot",
			content:  "---\n\"title\": \"Mine\"\ntags: [ops]\ncreated: 2026-01-01\n---\n\nbody\n",
			wantFile: "---\n\"title\": \"Mine\"\ntags: [ops]\ncreated: 2026-01-01\n---\n\nbody\n",
		},
		{
			name:    "CRLF front-matter keeps its line endings",
			title:   "Jot",
			content: "---\r\ndescription: x\r\n---\r\n\r\nbody\r\n",
			wantFile: "---\r\ntitle: \"Jot\"\r\ntags: []\r\ncreated: " + today +
				"\r\ndescription: x\r\n---\r\n\r\nbody\r\n",
		},
		{
			name:     "a block that ends the content keeps its closing line",
			title:    "Jot",
			content:  "---\ndescription: x\n---",
			wantFile: "---\n" + defaults + "description: x\n---",
		},
		{
			name:     "content without front-matter gets a whole block",
			title:    "Jot",
			content:  "Hello.",
			wantFile: "---\n" + defaults + "---\n\nHello.",
		},
		{
			name:    "a folder escaping the notes dir returns error",
			title:   "../escape/Title",
			wantErr: true,
		},
		{
			name:  "a folder symlinked out of the notes dir returns error",
			title: "out/Title",
			before: func(t *testing.T, svc *jot.Service) {
				symlink(t, t.TempDir(), filepath.Join(svc.NotesDir, "out"))
			},
			wantErr: true,
		},
		{
			name:  "an existing note is not overwritten",
			title: "Twice",
			before: func(t *testing.T, svc *jot.Service) {
				if _, _, err := svc.CreateNote("Twice", "first", false); err != nil {
					t.Fatalf("setup create: %v", err)
				}
			},
			wantErr: true,
		},
		{
			name:        "secure note keeps only frontmatter on disk",
			title:       "Vault Note",
			content:     "Top secret.",
			secure:      true,
			wantSlugSub: "vault-note",
			wantBodySub: "secure: true",
		},
		{
			name:        "creates note with content",
			title:       "Test Note",
			content:     "# Test Note\n\nHello.",
			wantSlugSub: "test-note",
			wantBodySub: "Hello.",
		},
		{
			name:        "empty content uses scaffold",
			title:       "Scaffold Note",
			content:     "",
			wantSlugSub: "scaffold-note",
			wantBodySub: "title:",
		},
		{
			name:    "unwritable dir returns error",
			title:   "Fail Note",
			content: "body",
			dir: func(t *testing.T) string {
				dir := t.TempDir()
				p := filepath.Join(dir, "readonly")
				if err := os.MkdirAll(p, 0o500); err != nil {
					t.Fatalf("setup: %v", err)
				}
				noWrite := filepath.Join(p, "nested")
				return noWrite
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var dir string
			if tt.dir != nil {
				dir = tt.dir(t)
			} else {
				dir = t.TempDir()
			}
			svc := newSvc(dir)
			if tt.secure {
				svc = newSecureSvc(t, dir)
			}
			if tt.before != nil {
				tt.before(t, svc)
			}

			slug, path, err := svc.CreateNote(tt.title, tt.content, tt.secure)
			if tt.wantErr {
				if err == nil {
					t.Fatal("CreateNote: expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("CreateNote: %v", err)
			}
			if !strings.Contains(slug, tt.wantSlugSub) {
				t.Errorf("slug = %q, want substring %q", slug, tt.wantSlugSub)
			}

			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read created file: %v", err)
			}
			if !strings.Contains(string(data), tt.wantBodySub) {
				t.Errorf("file content = %q, want substring %q", string(data), tt.wantBodySub)
			}
			if tt.wantFile != "" && string(data) != tt.wantFile {
				t.Errorf("file content = %q, want %q", data, tt.wantFile)
			}
			if tt.secure {
				body, err := svc.CatNote(slug)
				if strings.Contains(string(data), tt.content) || err != nil || body != tt.content {
					t.Errorf("secure note: file %q, CatNote = %q, %v", data, body, err)
				}
			}
		})
	}
}

// ─── DeleteNote ─────────────────────────────────────────────────────────────

func TestDeleteNote(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		slug    string
		setup   func(t *testing.T, dir string)
		wantErr bool
	}{
		{
			name: "deletes existing note",
			slug: "my-note",
			setup: func(t *testing.T, dir string) {
				writeNote(t, dir, "my-note", "# Note\n")
			},
		},
		{
			name:  "missing note is not an error",
			slug:  "no-such-note",
			setup: func(_ *testing.T, _ string) {},
		},
		{
			name:    "slug escaping the notes dir returns error",
			slug:    "../outside",
			setup:   func(_ *testing.T, _ string) {},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			tt.setup(t, dir)
			svc := newSvc(dir)

			err := svc.DeleteNote(tt.slug)
			if tt.wantErr {
				if err == nil {
					t.Fatal("DeleteNote: expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("DeleteNote: unexpected error: %v", err)
			}

			path := filepath.Join(dir, tt.slug+".md")
			if _, err := os.Stat(path); err == nil {
				t.Error("expected file to be deleted")
			}
		})
	}
}

// ─── RenameNote ─────────────────────────────────────────────────────────────

func TestRenameNote(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		oldSlug    string
		newTitle   string
		secure     bool // encrypt oldSlug after setup
		setup      func(t *testing.T, dir string)
		wantNewSub string
		check      func(t *testing.T, svc *jot.Service, newSlug string)
		wantErr    bool
	}{
		{
			name:     "a note is renamed from the new title alone",
			oldSlug:  "memory/jot",
			newTitle: "Agents",
			setup: func(t *testing.T, dir string) {
				if err := os.MkdirAll(filepath.Join(dir, "memory"), 0o700); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				writeNote(t, dir, "memory/jot", "---\ntitle: \"Jot\"\ntags: []\n---\n\nbody\n")
			},
			check: func(t *testing.T, _ *jot.Service, newSlug string) {
				if newSlug != "memory/agents" {
					t.Errorf("newSlug = %q, want %q", newSlug, "memory/agents")
				}
			},
		},
		{
			name:     "a dated note loses its date",
			oldSlug:  "2026-01-02-standup",
			newTitle: "Retro",
			setup: func(t *testing.T, dir string) {
				writeNote(t, dir, "2026-01-02-standup", "---\ntitle: \"Standup\"\ntags: []\n---\n")
			},
			check: func(t *testing.T, _ *jot.Service, newSlug string) {
				if newSlug != "retro" {
					t.Errorf("newSlug = %q, want %q", newSlug, "retro")
				}
			},
		},
		{
			name:     "a title with no letters or digits returns error",
			oldSlug:  "memory/jot",
			newTitle: "!!!",
			setup: func(t *testing.T, dir string) {
				if err := os.MkdirAll(filepath.Join(dir, "memory"), 0o700); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				writeNote(t, dir, "memory/jot", "# Jot\n")
			},
			wantErr: true,
		},
		{
			name:     "a note in a folder stays there, and both link forms follow it",
			oldSlug:  "ops/old-note",
			newTitle: "New Title",
			setup: func(t *testing.T, dir string) {
				if err := os.MkdirAll(filepath.Join(dir, "ops"), 0o700); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				writeNote(t, dir, "ops/old-note", "---\ntitle: \"Old\"\ntags: []\n---\n\n# Old\n")
				writeNote(t, dir, "linker", "See [[ops/old-note]] and [[old-note]].\n")
			},
			wantNewSub: "ops/",
			check: func(t *testing.T, svc *jot.Service, newSlug string) {
				data, err := os.ReadFile(filepath.Join(svc.NotesDir, "linker.md"))
				want := "See [[" + newSlug + "]] and [[" + filepath.Base(newSlug) + "]].\n"
				if err != nil || string(data) != want {
					t.Errorf("linker = %q, %v; want %q", data, err, want)
				}
			},
		},
		{
			name:     "a base-name link stays when another note has that name",
			oldSlug:  "ops/old-note",
			newTitle: "New Title",
			setup: func(t *testing.T, dir string) {
				for _, d := range []string{"ops", "other"} {
					if err := os.MkdirAll(filepath.Join(dir, d), 0o700); err != nil {
						t.Fatalf("mkdir: %v", err)
					}
				}
				writeNote(t, dir, "ops/old-note", "---\ntitle: \"Old\"\ntags: []\n---\n\n# Old\n")
				writeNote(t, dir, "other/old-note", "# Other\n")
				writeNote(t, dir, "linker", "See [[ops/old-note]] and [[old-note]].\n")
			},
			wantNewSub: "ops/",
			check: func(t *testing.T, svc *jot.Service, newSlug string) {
				data, err := os.ReadFile(filepath.Join(svc.NotesDir, "linker.md"))
				want := "See [[" + newSlug + "]] and [[old-note]].\n"
				if err != nil || string(data) != want {
					t.Errorf("linker = %q, %v; want %q", data, err, want)
				}
			},
		},
		{
			name:     "a secure note's body follows it to the new slug",
			oldSlug:  "ops/vault",
			newTitle: "New Vault",
			secure:   true,
			setup: func(t *testing.T, dir string) {
				if err := os.MkdirAll(filepath.Join(dir, "ops"), 0o700); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				writeNote(t, dir, "ops/vault", "---\ntitle: \"Vault\"\ntags: []\n---\n\nsecret\n")
			},
			wantNewSub: "ops/new-vault",
			check: func(t *testing.T, svc *jot.Service, newSlug string) {
				if body, err := svc.CatNote(newSlug); err != nil || body != "\nsecret\n" {
					t.Errorf("CatNote(%q) = %q, %v; want %q", newSlug, body, err, "\nsecret\n")
				}
				old := filepath.Join(svc.NotesDir, ".kvlt/secrets/local/jot/ops%2Fvault.age")
				if _, err := os.Stat(old); err == nil {
					t.Errorf("old vault entry %s still exists", old)
				}
			},
		},
		{
			name:     "a secure note without a vault returns error",
			oldSlug:  "vault",
			newTitle: "New Vault",
			setup: func(t *testing.T, dir string) {
				writeNote(t, dir, "vault", "---\ntitle: \"Vault\"\nsecure: true\ntags: []\n---\n")
			},
			wantErr: true,
		},
		{
			name:     "slug escaping the notes dir returns error",
			oldSlug:  "../outside",
			newTitle: "Whatever",
			setup:    func(_ *testing.T, _ string) {},
			wantErr:  true,
		},
		{
			name:     "renames note and updates title",
			oldSlug:  "old-note",
			newTitle: "New Title",
			setup: func(t *testing.T, dir string) {
				writeNote(t, dir, "old-note", `---
title: "Old Title"
tags: []
created: 2026-05-14
---

# Old Title

Content here.
`)
			},
			wantNewSub: "new-title",
		},
		{
			name:     "rewrites wikilinks in other notes",
			oldSlug:  "old-note",
			newTitle: "New Title",
			setup: func(t *testing.T, dir string) {
				writeNote(t, dir, "old-note", `---
title: "Old"
tags: []
created: 2026-05-14
---

# Old
`)
				writeNote(t, dir, "linker", `---
title: "Linker"
tags: []
created: 2026-05-14
---

See [[old-note]] for details.
`)
			},
			wantNewSub: "new-title",
		},
		{
			name:     "missing note returns error",
			oldSlug:  "no-such-note",
			newTitle: "Whatever",
			setup:    func(_ *testing.T, _ string) {},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			tt.setup(t, dir)
			svc := newSvc(dir)
			if tt.secure {
				svc = newSecureSvc(t, dir)
				if err := svc.EncryptNote(tt.oldSlug); err != nil {
					t.Fatalf("setup encrypt: %v", err)
				}
			}

			newSlug, err := svc.RenameNote(tt.oldSlug, tt.newTitle)
			if tt.wantErr {
				if err == nil {
					t.Fatal("RenameNote: expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("RenameNote: unexpected error: %v", err)
			}
			if !strings.Contains(newSlug, tt.wantNewSub) {
				t.Errorf("newSlug = %q, want substring %q", newSlug, tt.wantNewSub)
			}

			oldPath := filepath.Join(dir, tt.oldSlug+".md")
			if _, err := os.Stat(oldPath); err == nil {
				t.Error("old file still exists after rename")
			}

			newPath := filepath.Join(dir, newSlug+".md")
			data, err := os.ReadFile(newPath)
			if err != nil {
				t.Fatalf("read renamed file: %v", err)
			}
			if !strings.Contains(string(data), tt.newTitle) {
				t.Errorf("renamed file missing new title %q in content", tt.newTitle)
			}
			if tt.check != nil {
				tt.check(t, svc, newSlug)
			}
		})
	}
}

// ─── MarkTaskDone ───────────────────────────────────────────────────────────

func TestMarkTaskDone(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		slug    string
		desc    string
		reason  string
		setup   func(t *testing.T, dir string)
		want    []string // substrings of the note after marking
		wantErr bool
	}{
		{
			name:   "marks matching task done",
			slug:   "tasks-note",
			desc:   "Review PR",
			reason: "shipped",
			setup: func(t *testing.T, dir string) {
				writeNote(t, dir, "tasks-note", testNote)
			},
		},
		{
			name:   "an exact match wins over a longer one",
			slug:   "deploys",
			desc:   "deploy",
			reason: "shipped",
			setup: func(t *testing.T, dir string) {
				writeNote(t, dir, "deploys", "- [ ] Deploy staging\n- [ ] Deploy\n")
			},
			want: []string{"- [ ] Deploy staging\n", "- [x] Deploy | done:"},
		},
		{
			name:   "keeps a nested task's indentation",
			slug:   "nested",
			desc:   "child",
			reason: "shipped",
			setup: func(t *testing.T, dir string) {
				writeNote(t, dir, "nested", "- [ ] parent\n  - [ ] child step\n")
			},
			want: []string{"\n  - [x] child step | done:"},
		},
		{
			name:   "records the reason after the done date",
			slug:   "deploys",
			desc:   "Deploy",
			reason: " superseded by the new pipeline ",
			setup: func(t *testing.T, dir string) {
				writeNote(t, dir, "deploys", "- [ ] Deploy | id:t-0001 #ops\n")
			},
			want: []string{"- [x] Deploy | id:t-0001 | done:" + time.Now().Format("2006-01-02") +
				" | reason:superseded by the new pipeline #ops\n"},
		},
		{
			name:   "a missing reason returns error",
			slug:   "deploys",
			desc:   "Deploy",
			reason: "  ",
			setup: func(t *testing.T, dir string) {
				writeNote(t, dir, "deploys", "- [ ] Deploy\n")
			},
			wantErr: true,
		},
		{
			name:   "a reason with a pipe returns error",
			slug:   "deploys",
			desc:   "Deploy",
			reason: "done | due:friday",
			setup: func(t *testing.T, dir string) {
				writeNote(t, dir, "deploys", "- [ ] Deploy\n")
			},
			wantErr: true,
		},
		{
			name:   "a multi-line reason returns error",
			slug:   "deploys",
			desc:   "Deploy",
			reason: "done\n- [ ] sneaky",
			setup: func(t *testing.T, dir string) {
				writeNote(t, dir, "deploys", "- [ ] Deploy\n")
			},
			wantErr: true,
		},
		{
			name:   "a reason with a #tag returns error",
			slug:   "deploys",
			desc:   "Deploy",
			reason: "moved to #ops",
			setup: func(t *testing.T, dir string) {
				writeNote(t, dir, "deploys", "- [ ] Deploy\n")
			},
			wantErr: true,
		},
		{
			name:   "several partial matches return error",
			slug:   "deploys",
			desc:   "Deploy",
			reason: "shipped",
			setup: func(t *testing.T, dir string) {
				writeNote(t, dir, "deploys", "- [ ] Deploy staging\n- [ ] Deploy prod\n")
			},
			wantErr: true,
		},
		{
			name:   "two exact matches return error",
			slug:   "deploys",
			desc:   "Deploy",
			reason: "shipped",
			setup: func(t *testing.T, dir string) {
				writeNote(t, dir, "deploys", "- [ ] Deploy\n- [ ] Deploy\n")
			},
			wantErr: true,
		},
		{
			name:    "empty description returns error",
			slug:    "tasks-note",
			desc:    "  ",
			reason:  "shipped",
			setup:   func(_ *testing.T, _ string) {},
			wantErr: true,
		},
		{
			name:    "slug escaping the notes dir returns error",
			slug:    "../outside",
			desc:    "anything",
			reason:  "shipped",
			setup:   func(_ *testing.T, _ string) {},
			wantErr: true,
		},
		{
			name:   "case insensitive match",
			slug:   "tasks-note",
			desc:   "review pr",
			reason: "shipped",
			setup: func(t *testing.T, dir string) {
				writeNote(t, dir, "tasks-note", testNote)
			},
		},
		{
			name:   "no matching task returns error",
			slug:   "tasks-note",
			desc:   "nonexistent task",
			reason: "shipped",
			setup: func(t *testing.T, dir string) {
				writeNote(t, dir, "tasks-note", testNote)
			},
			wantErr: true,
		},
		{
			name:    "missing note returns error",
			slug:    "no-such-note",
			desc:    "anything",
			reason:  "shipped",
			setup:   func(_ *testing.T, _ string) {},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			tt.setup(t, dir)
			svc := newSvc(dir)

			err := svc.MarkTaskDone(tt.slug, tt.desc, tt.reason, false)
			if tt.wantErr {
				if err == nil {
					t.Fatal("MarkTaskDone: expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("MarkTaskDone: unexpected error: %v", err)
			}

			data, err := os.ReadFile(filepath.Join(dir, tt.slug+".md"))
			if err != nil {
				t.Fatalf("read note: %v", err)
			}
			if !strings.Contains(string(data), "[x]") {
				t.Error("expected task to be marked done with [x]")
			}
			for _, sub := range tt.want {
				if !strings.Contains(string(data), sub) {
					t.Errorf("note = %q, want substring %q", data, sub)
				}
			}
		})
	}
}

// ─── MarkTaskDoneAt ─────────────────────────────────────────────────────────

func TestMarkTaskDoneAt(t *testing.T) {
	t.Parallel()

	// testNote's tasks sit on file lines 11 (open), 12 (done) and 13 (open).
	tests := []struct {
		name    string
		slug    string
		line    int
		want    string
		wantErr bool
	}{
		{
			name: "marks the open task on that line",
			slug: "tasks-note",
			line: 13,
			want: "- [x] Deploy staging | done:",
		},
		{name: "a done task returns error", slug: "tasks-note", line: 12, wantErr: true},
		{
			name:    "a line that is not a task returns error",
			slug:    "tasks-note",
			line:    1,
			wantErr: true,
		},
		{name: "line zero returns error", slug: "tasks-note", line: 0, wantErr: true},
		{name: "a line past the end returns error", slug: "tasks-note", line: 99, wantErr: true},
		{name: "missing note returns error", slug: "no-such-note", line: 1, wantErr: true},
		{
			name:    "slug escaping the notes dir returns error",
			slug:    "../outside",
			line:    1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			writeNote(t, dir, "tasks-note", testNote)
			svc := newSvc(dir)

			err := svc.MarkTaskDoneAt(tt.slug, tt.line, "shipped", false)
			if tt.wantErr {
				if err == nil {
					t.Fatal("MarkTaskDoneAt: expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("MarkTaskDoneAt: unexpected error: %v", err)
			}
			data, err := os.ReadFile(filepath.Join(dir, tt.slug+".md"))
			if err != nil || !strings.Contains(string(data), tt.want) {
				t.Errorf("note = %q, %v; want substring %q", data, err, tt.want)
			}
			if !strings.Contains(string(data), "- [ ] Review PR") {
				t.Error("marked a different task than the one on that line")
			}
		})
	}
}

// ─── AddTask ────────────────────────────────────────────────────────────────

func TestAddTask(t *testing.T) {
	t.Parallel()

	done := jot.RawTask{
		Description: "Merged !20",
		Done:        true,
		DoneDate:    "2026-11-02",
		Labels:      []string{"cos"},
	}

	tests := []struct {
		name    string
		slug    string
		title   string
		task    jot.RawTask
		setup   func(t *testing.T, dir string)
		want    []string // substrings of the resulting file
		wantErr bool
	}{
		{
			name:  "creates a nested note with its title and the task",
			slug:  "done/2026-11-02",
			title: "Done 2026-11-02",
			task:  done,
			setup: func(_ *testing.T, _ string) {},
			want: []string{
				`title: "Done 2026-11-02"`,
				"# Done 2026-11-02\n",
				"\n- [x] Merged !20 | id:t-",
				" | done:2026-11-02 #cos\n",
			},
		},
		{
			name:  "title defaults to the slug's last segment",
			slug:  "inbox/todo",
			task:  jot.RawTask{Description: "call Sam"},
			setup: func(_ *testing.T, _ string) {},
			want:  []string{`title: "todo"`, "- [ ] call Sam | id:t-"},
		},
		{
			name: "appends to an existing note without a trailing newline",
			slug: "tasks-note",
			task: jot.RawTask{
				Description: "Tag release",
				DueDate:     "2026-05-20",
				Labels:      []string{"ops"},
			},
			setup: func(t *testing.T, dir string) {
				writeNote(t, dir, "tasks-note", strings.TrimRight(testNote, "\n"))
			},
			want: []string{
				"title: \"Test Note\"",
				"\n- [ ] Tag release | id:t-",
				" | due:2026-05-20 #ops\n",
			},
		},
		{
			name:  "done without a date is done today",
			slug:  "today",
			task:  jot.RawTask{Description: "x", Done: true},
			setup: func(_ *testing.T, _ string) {},
			want:  []string{"- [x] x | id:t-", " | done:"},
		},
		{
			name: "note path blocked by a file is a read error",
			slug: "blocker/note",
			task: done,
			setup: func(t *testing.T, dir string) {
				writeNote(t, dir, "blocker", testNote)
				if err := os.Rename(filepath.Join(dir, "blocker.md"), filepath.Join(dir, "blocker")); err != nil {
					t.Fatalf("setup: %v", err)
				}
			},
			wantErr: true,
		},
		{
			name: "folder that cannot be created",
			slug: "locked/sub/note",
			task: done,
			setup: func(t *testing.T, dir string) {
				if err := os.Mkdir(filepath.Join(dir, "locked"), 0o500); err != nil {
					t.Fatalf("setup: %v", err)
				}
			},
			wantErr: true,
		},
		{
			name:    "empty slug",
			task:    done,
			setup:   func(_ *testing.T, _ string) {},
			wantErr: true,
		},
		{
			name:    "slug escaping the notes dir",
			slug:    "../outside",
			task:    done,
			setup:   func(_ *testing.T, _ string) {},
			wantErr: true,
		},
		{
			name:    "empty description",
			slug:    "n",
			task:    jot.RawTask{Description: "   "},
			setup:   func(_ *testing.T, _ string) {},
			wantErr: true,
		},
		{
			name:    "description with a pipe",
			slug:    "n",
			task:    jot.RawTask{Description: "a | b"},
			setup:   func(_ *testing.T, _ string) {},
			wantErr: true,
		},
		{
			name:    "multi-line description",
			slug:    "n",
			task:    jot.RawTask{Description: "a\nb"},
			setup:   func(_ *testing.T, _ string) {},
			wantErr: true,
		},
		{
			name:    "description with an inline tag",
			slug:    "n",
			task:    jot.RawTask{Description: "replied in #eng-sensor"},
			setup:   func(_ *testing.T, _ string) {},
			wantErr: true,
		},
		{
			name:    "invalid tag",
			slug:    "n",
			task:    jot.RawTask{Description: "x", Labels: []string{"two words"}},
			setup:   func(_ *testing.T, _ string) {},
			wantErr: true,
		},
		{
			name:    "invalid date",
			slug:    "n",
			task:    jot.RawTask{Description: "x", DueDate: "friday"},
			setup:   func(_ *testing.T, _ string) {},
			wantErr: true,
		},
		{
			name:    "a date shaped right but not on the calendar",
			slug:    "n",
			task:    jot.RawTask{Description: "x", Done: true, DoneDate: "2026-13-45"},
			setup:   func(_ *testing.T, _ string) {},
			wantErr: true,
		},
		{
			name: "secure note is refused",
			slug: "vault",
			task: done,
			setup: func(t *testing.T, dir string) {
				writeNote(t, dir, "vault", "---\ntitle: \"v\"\ntags: []\nsecure: true\n---\n")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			tt.setup(t, dir)
			svc := newSvc(dir)

			_, err := svc.AddTask(tt.slug, tt.title, tt.task)
			if tt.wantErr {
				if err == nil {
					t.Fatal("AddTask: expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("AddTask: unexpected error: %v", err)
			}

			data, err := os.ReadFile(filepath.Join(dir, tt.slug+".md"))
			if err != nil {
				t.Fatalf("read note: %v", err)
			}
			for _, sub := range tt.want {
				if !strings.Contains(string(data), sub) {
					t.Errorf("note = %q, want substring %q", data, sub)
				}
			}

			// Round trip: the parser reads back exactly the task written.
			tasks, err := svc.AllTasks("all", "", "")
			if err != nil {
				t.Fatalf("AllTasks: %v", err)
			}
			last := tasks[len(tasks)-1]
			if last.Description != strings.TrimSpace(tt.task.Description) {
				t.Errorf(
					"round trip description = %q, want %q",
					last.Description,
					tt.task.Description,
				)
			}
		})
	}
}

// ─── task dependencies ──────────────────────────────────────────────────────

// depNote's tasks sit on file lines 6 (blocker), 7 (blocked by it) and 8
// (blocked by a task that no longer exists).
const depNote = "---\ntitle: \"todo\"\ntags: []\n---\n\n" +
	"- [ ] write docs | id:t-0001\n" +
	"- [ ] ship it | id:t-0002 | after:t-0001\n" +
	"- [ ] orphan | id:t-0003 | after:t-gone\n"

// lockedNote writes a one-task note at slug and then sets its mode, so reads
// (0o000) or writes (0o400) of it fail.
func lockedNote(t *testing.T, dir, slug string, mode os.FileMode) {
	t.Helper()
	if err := os.Chmod(writeNote(t, dir, slug, "- [ ] x | id:t-0009\n"), mode); err != nil {
		t.Fatalf("chmod: %v", err)
	}
}

func TestAddTaskIDs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		missingDir bool // the notes directory does not exist yet
		setup      func(t *testing.T, dir string)
		task       jot.RawTask
		want       string // substring of the written line, after the new ID
		wantErr    bool
	}{
		{name: "assigns an ID", task: jot.RawTask{Description: "call Sam"}},
		{
			name:       "creates a notes directory that does not exist yet",
			missingDir: true,
			task:       jot.RawTask{Description: "call Sam"},
		},
		{
			name:    "an unreadable note fails the ID check",
			setup:   func(t *testing.T, dir string) { lockedNote(t, dir, "locked", 0o000) },
			task:    jot.RawTask{Description: "call Sam"},
			wantErr: true,
		},
		{
			name:    "a note it cannot write",
			setup:   func(t *testing.T, dir string) { lockedNote(t, dir, "inbox", 0o400) },
			task:    jot.RawTask{Description: "call Sam"},
			wantErr: true,
		},
		{
			name: "keeps blockers that exist, in lower case",
			task: jot.RawTask{Description: "call Sam", After: []string{"T-0001"}},
			want: " | after:t-0001",
		},
		{
			name:    "refuses a blocker that does not exist",
			task:    jot.RawTask{Description: "call Sam", After: []string{"t-zzzz"}},
			wantErr: true,
		},
		{
			name:    "refuses an ID another task has",
			task:    jot.RawTask{Description: "call Sam", ID: "t-0002"},
			wantErr: true,
		},
	}

	idRe := regexp.MustCompile(`^t-[0-9a-hjkmnp-tv-z]{4}$`)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if tt.missingDir {
				dir = filepath.Join(dir, "notes")
			} else {
				writeNote(t, dir, "todo", depNote)
			}
			if tt.setup != nil {
				tt.setup(t, dir)
			}
			svc := newSvc(dir)

			got, err := svc.AddTask("inbox", "", tt.task)
			if tt.wantErr {
				if err == nil {
					t.Fatal("AddTask: expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("AddTask: unexpected error: %v", err)
			}
			if !idRe.MatchString(got.ID) {
				t.Fatalf("AddTask ID = %q, want t- and 4 Crockford base32 characters", got.ID)
			}
			data, err := os.ReadFile(filepath.Join(dir, "inbox.md"))
			if line := "- [ ] call Sam | id:" + got.ID + tt.want; err != nil ||
				!strings.Contains(string(data), line) {
				t.Errorf("note = %q, %v; want line %q", data, err, line)
			}
		})
	}
}

func TestMarkTaskDoneBlocked(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(t *testing.T, svc *jot.Service)
		op      func(svc *jot.Service) error
		wantErr string // substring of the error; "" means it succeeds
	}{
		{
			name:    "by ID, refuses while a blocker is open",
			op:      func(svc *jot.Service) error { return svc.MarkTaskDoneByID("T-0002", "shipped", false) },
			wantErr: `blocked by t-0001 "write docs"`,
		},
		{
			name:    "by description, refuses while a blocker is open",
			op:      func(svc *jot.Service) error { return svc.MarkTaskDone("todo", "ship it", "shipped", false) },
			wantErr: "blocked by t-0001",
		},
		{
			name:    "by line, refuses while a blocker is open",
			op:      func(svc *jot.Service) error { return svc.MarkTaskDoneAt("todo", 7, "shipped", false) },
			wantErr: "blocked by t-0001",
		},
		{
			name: "a blocker in another note blocks",
			setup: func(t *testing.T, svc *jot.Service) {
				writeNote(t, svc.NotesDir, "work", "- [ ] deploy | id:t-0004 | after:t-0001\n")
			},
			op:      func(svc *jot.Service) error { return svc.MarkTaskDoneByID("t-0004", "shipped", false) },
			wantErr: "blocked by t-0001",
		},
		{
			name: "force closes a blocked task",
			op:   func(svc *jot.Service) error { return svc.MarkTaskDoneByID("t-0002", "shipped", true) },
		},
		{
			name: "closing the blocker unblocks it",
			setup: func(t *testing.T, svc *jot.Service) {
				if err := svc.MarkTaskDoneByID("t-0001", "shipped", false); err != nil {
					t.Fatalf("setup: %v", err)
				}
			},
			op: func(svc *jot.Service) error { return svc.MarkTaskDoneByID("t-0002", "shipped", false) },
		},
		{
			name: "a blocker that no longer exists does not block",
			op:   func(svc *jot.Service) error { return svc.MarkTaskDoneByID("t-0003", "shipped", false) },
		},
		{
			name:    "an unknown ID",
			op:      func(svc *jot.Service) error { return svc.MarkTaskDoneByID("t-zzzz", "shipped", false) },
			wantErr: "no task with id t-zzzz",
		},
		{
			name: "an ID on two tasks is refused",
			setup: func(t *testing.T, svc *jot.Service) {
				writeNote(t, svc.NotesDir, "clone", "- [ ] clash | id:t-0001\n")
			},
			op:      func(svc *jot.Service) error { return svc.MarkTaskDoneByID("t-0001", "shipped", false) },
			wantErr: "id t-0001 is on 2 tasks",
		},
		{
			name: "force skips the check, by description",
			op:   func(svc *jot.Service) error { return svc.MarkTaskDone("todo", "ship it", "shipped", true) },
		},
		{
			name:    "an unreadable note fails the check, by description",
			setup:   func(t *testing.T, svc *jot.Service) { lockedNote(t, svc.NotesDir, "locked", 0o000) },
			op:      func(svc *jot.Service) error { return svc.MarkTaskDone("todo", "write docs", "shipped", false) },
			wantErr: "permission denied",
		},
		{
			name:    "an unreadable note fails the check, by line",
			setup:   func(t *testing.T, svc *jot.Service) { lockedNote(t, svc.NotesDir, "locked", 0o000) },
			op:      func(svc *jot.Service) error { return svc.MarkTaskDoneAt("todo", 6, "shipped", false) },
			wantErr: "permission denied",
		},
		{
			name:    "an unreadable note fails the check, by ID",
			setup:   func(t *testing.T, svc *jot.Service) { lockedNote(t, svc.NotesDir, "locked", 0o000) },
			op:      func(svc *jot.Service) error { return svc.MarkTaskDoneByID("t-0001", "shipped", false) },
			wantErr: "permission denied",
		},
		{
			name: "a task already done",
			setup: func(t *testing.T, svc *jot.Service) {
				if err := svc.MarkTaskDoneByID("t-0001", "shipped", false); err != nil {
					t.Fatalf("setup: %v", err)
				}
			},
			op:      func(svc *jot.Service) error { return svc.MarkTaskDoneByID("t-0001", "shipped", false) },
			wantErr: "already done",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			writeNote(t, dir, "todo", depNote)
			svc := newSvc(dir)
			if tt.setup != nil {
				tt.setup(t, svc)
			}

			err := tt.op(svc)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want one containing %q", err, tt.wantErr)
			}
		})
	}
}

// ─── RewriteNoteTitle ───────────────────────────────────────────────────────

func TestRewriteNoteTitle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		content  string
		newTitle string
		wantSub  string
	}{
		{
			name: "replaces frontmatter title and heading",
			content: `---
title: "Old Title"
tags: []
---

# Old Title

Body.
`,
			newTitle: "New Title",
			wantSub:  `title: "New Title"`,
		},
		{
			name:     "handles content without frontmatter",
			content:  "# Old Heading\n\nBody.\n",
			newTitle: "New Heading",
			wantSub:  "# New Heading",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := jot.RewriteNoteTitle(tt.content, tt.newTitle)
			if !strings.Contains(got, tt.wantSub) {
				t.Errorf("got %q, want substring %q", got, tt.wantSub)
			}
		})
	}
}

// ─── RewriteWikiLinks ───────────────────────────────────────────────────────

func TestRewriteWikiLinks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(t *testing.T, dir string)
		oldSlug string
		newSlug string
		wantSub string
	}{
		{
			name: "replaces wikilinks in all notes",
			setup: func(t *testing.T, dir string) {
				writeNote(t, dir, "a", "See [[old-slug]] for details.\n")
				writeNote(t, dir, "b", "Also [[old-slug]] here and [[other]].\n")
			},
			oldSlug: "old-slug",
			newSlug: "new-slug",
			wantSub: "[[new-slug]]",
		},
		{
			name: "no matching links leaves files unchanged",
			setup: func(t *testing.T, dir string) {
				writeNote(t, dir, "c", "See [[unrelated]] only.\n")
			},
			oldSlug: "old-slug",
			newSlug: "new-slug",
			wantSub: "[[unrelated]]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			tt.setup(t, dir)

			err := jot.RewriteWikiLinks(dir, tt.oldSlug, tt.newSlug)
			if err != nil {
				t.Fatalf("RewriteWikiLinks: %v", err)
			}

			entries, _ := os.ReadDir(dir)
			for _, e := range entries {
				if !strings.HasSuffix(e.Name(), ".md") {
					continue
				}
				data, _ := os.ReadFile(filepath.Join(dir, e.Name()))
				content := string(data)
				if strings.Contains(content, "[["+tt.oldSlug+"]]") {
					t.Errorf("file %s still contains old wikilink [[%s]]", e.Name(), tt.oldSlug)
				}
			}
		})
	}
}

// ─── FormatTags ─────────────────────────────────────────────────────────────

func TestFormatTags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		tags []string
		want string
	}{
		{
			name: "empty slice",
			tags: nil,
			want: "",
		},
		{
			name: "single tag",
			tags: []string{"work"},
			want: "work",
		},
		{
			name: "multiple tags",
			tags: []string{"work", "sprint", "q2"},
			want: "work, sprint, q2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := jot.FormatTags(tt.tags)
			if got != tt.want {
				t.Errorf("FormatTags(%v) = %q, want %q", tt.tags, got, tt.want)
			}
		})
	}
}

// ─── Committer ──────────────────────────────────────────────────────────────

type recordCommitter struct {
	subjects []string
	err      error
}

func (c *recordCommitter) CommitHeld(subject string) error {
	c.subjects = append(c.subjects, subject)
	return c.err
}

func TestServiceCommitter(t *testing.T) {
	t.Parallel()

	const mem = "---\ntitle: \"Mem\"\ntags: []\n---\n\n- [ ] ship it\n"

	tests := []struct {
		name      string
		secure    bool // vault set up and mem encrypted before the committer is attached
		op        func(svc *jot.Service) error
		commitErr error
		want      []string
		wantErr   string
	}{
		{
			name: "append",
			op:   func(svc *jot.Service) error { return svc.AppendNote("mem", "more") },
			want: []string{"note: append to mem"},
		},
		{
			name: "update",
			op:   func(svc *jot.Service) error { return svc.UpdateNote("mem", "new") },
			want: []string{"note: update mem"},
		},
		{
			name: "a leading slash commits the canonical slug",
			op:   func(svc *jot.Service) error { return svc.AppendNote("/mem", "more") },
			want: []string{"note: append to mem"},
		},
		{
			name: "rename",
			op: func(svc *jot.Service) error {
				_, err := svc.RenameNote("mem", "Memory")
				return err
			},
			want: []string{"note: rename mem -> memory"},
		},
		{
			name: "task done",
			op:   func(svc *jot.Service) error { return svc.MarkTaskDone("mem", "ship it", "shipped", false) },
			want: []string{`task: done "ship it" in mem`},
		},
		{
			name: "task done by line",
			op:   func(svc *jot.Service) error { return svc.MarkTaskDoneAt("mem", 6, "shipped", false) },
			want: []string{"task: done line 6 in mem"},
		},
		{
			name: "task add",
			op: func(svc *jot.Service) error {
				_, err := svc.AddTask("mem", "", jot.RawTask{Description: " write docs "})
				return err
			},
			want: []string{`task: add "write docs" to mem`},
		},
		{
			name: "create",
			op: func(svc *jot.Service) error {
				_, _, err := svc.CreateNote("ops/Fresh", "", false)
				return err
			},
			want: []string{"note: add ops/fresh"},
		},
		{
			name: "create secure",
			op: func(svc *jot.Service) error {
				_, _, err := svc.CreateNote("Vault", "secret", true)
				return err
			},
			secure: true,
			want:   []string{"note(secure): add vault"},
		},
		{
			name:   "decrypt",
			op:     func(svc *jot.Service) error { return svc.DecryptNote("mem") },
			secure: true,
			want:   []string{"note(secure): decrypt mem"},
		},
		{
			name: "encrypt",
			op: func(svc *jot.Service) error {
				if err := svc.DecryptNote("mem"); err != nil {
					return err
				}
				return svc.EncryptNote("mem")
			},
			secure: true,
			want:   []string{"note(secure): decrypt mem", "note(secure): encrypt mem"},
		},
		{
			name: "delete",
			op:   func(svc *jot.Service) error { return svc.DeleteNote("mem") },
			want: []string{"note: delete mem"},
		},
		{
			name:    "a failed write is not committed",
			op:      func(svc *jot.Service) error { return svc.AppendNote("no-such-note", "x") },
			wantErr: "no-such-note",
		},
		{
			name:      "a failed commit is reported, the write kept",
			op:        func(svc *jot.Service) error { return svc.AppendNote("mem", "more") },
			commitErr: errors.New("disk full"),
			want:      []string{"note: append to mem"},
			wantErr:   "saved, but the commit failed: disk full",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			writeNote(t, dir, "mem", mem)
			svc := newSvc(dir)
			if tt.secure {
				svc = newSecureSvc(t, dir)
				if err := svc.EncryptNote("mem"); err != nil {
					t.Fatalf("setup encrypt: %v", err)
				}
			}
			committer := &recordCommitter{err: tt.commitErr}
			svc.Committer = committer

			err := tt.op(svc)
			if tt.wantErr == "" && err != nil || tt.wantErr != "" &&
				(err == nil || !strings.Contains(err.Error(), tt.wantErr)) {
				t.Errorf("error = %v, want %q", err, tt.wantErr)
			}
			if !slices.Equal(committer.subjects, tt.want) {
				t.Errorf("commits = %q, want %q", committer.subjects, tt.want)
			}
		})
	}
}

// Concurrent writers each get their own commit: the lock spans the write and
// its commit, so one writer's commit never sweeps in another's file.
func TestServiceCommitter_concurrent(t *testing.T) {
	t.Parallel()

	const writers = 8
	dir := t.TempDir()
	repo, err := gitops.InitRepo(dir)
	if err != nil {
		t.Fatalf("init repo: %v", err)
	}
	svc := newSvc(dir)
	svc.Committer = repo

	var wg sync.WaitGroup
	errs := make(chan error, writers)
	for i := range writers {
		wg.Go(func() {
			_, _, err := svc.CreateNote(fmt.Sprintf("Note %d", i), "", false)
			errs <- err
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("CreateNote: %v", err)
		}
	}

	entries, err := repo.Log("", 0)
	if err != nil || len(entries) != writers {
		t.Fatalf("Log = %d entries, %v; want %d", len(entries), err, writers)
	}
	for _, e := range entries {
		if len(e.Files) != 1 {
			t.Errorf("commit %q changed %q, want one file", e.Message, e.Files)
		}
	}
}
