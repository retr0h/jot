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

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/retr0h/jot/internal/cli"
	"github.com/retr0h/jot/internal/gitops"
	"github.com/retr0h/jot/internal/jot"
)

var noteSecureFlag bool
var noteNewTitleFlag string

// noteNewCmd implements `jot note new --title <title> [--secure]`.
// A "/" in the title splits into subdir + title so
// `jot note new --title "network/Switch Config"` stores the note under notes/network/.
var noteNewCmd = &cobra.Command{
	Use:   "new",
	Short: "Create a new note and open it in your editor",
	Args:  cobra.NoArgs,
	RunE: func(c *cobra.Command, args []string) error {
		out := c.OutOrStdout()
		rawTitle := noteNewTitleFlag
		notesDir := NotesDir()

		// Support "subdir/Title" notation.
		subdir := ""
		title := rawTitle
		if idx := strings.LastIndex(rawTitle, "/"); idx >= 0 {
			subdir = rawTitle[:idx]
			title = rawTitle[idx+1:]
		}

		slug := jot.NewSlug(title, time.Now())
		if subdir != "" {
			slug = subdir + "/" + slug
		}

		content := jot.ScaffoldFrontmatter(title, time.Now().Format("2006-01-02"))

		// Write file.
		dir := filepath.Dir(filepath.Join(notesDir, slug))
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("create notes dir %q: %w", dir, err)
		}
		notePath := filepath.Join(notesDir, slug+".md")
		if err := os.WriteFile(notePath, []byte(content), 0o600); err != nil {
			return fmt.Errorf("write note file %q: %w", notePath, err)
		}

		if err := jot.Edit(notePath, EditorPref()); err != nil {
			return fmt.Errorf("edit note: %w", err)
		}

		// Auto-commit via gitops.
		if repo, err := gitops.OpenRepo(notesDir); err == nil {
			msg := gitops.FormatCommitMessage(
				fmt.Sprintf("note: add %s", filepath.Base(slug)),
				"",
			)
			_ = repo.Commit(msg)
		}

		fmt.Fprintln(out, cli.Success(out, "note created: "+cli.Accent(out, slug)))
		return nil
	},
}

func init() {
	noteNewCmd.Flags().StringVar(&noteNewTitleFlag, "title", "", "note title (use path/title for subdirs)")
	_ = noteNewCmd.MarkFlagRequired("title")
	noteNewCmd.Flags().BoolVar(&noteSecureFlag, "secure", false, "mark note as secure (sets secure: true in front-matter)")
}
