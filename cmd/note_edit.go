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
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/retr0h/jot/internal/cli"
	"github.com/retr0h/jot/internal/gitops"
	"github.com/retr0h/jot/internal/jot"
)

var noteEditSlugFlag string

// noteEditCmd implements `jot note edit [--slug <slug>]`.
// Opens notesDir/slug.md in the configured editor and auto-commits after.
// If --slug is omitted, an fzf picker is shown.
var noteEditCmd = &cobra.Command{
	Use:     "edit",
	Aliases: []string{"e"},
	Short:   "Open an existing note in your editor",
	Args:    cobra.NoArgs,
	RunE: func(c *cobra.Command, _ []string) error {
		out := c.OutOrStdout()
		notesDir := NotesDir()

		slug := noteEditSlugFlag
		if slug == "" {
			picked, err := pickNote(notesDir)
			if err != nil {
				return err
			}
			slug = picked
		}

		notePath := filepath.Join(notesDir, slug+".md")

		if err := jot.Edit(notePath, ""); err != nil {
			return fmt.Errorf("edit note: %w", err)
		}

		// Auto-commit via gitops.
		if repo, err := gitops.OpenRepo(notesDir); err == nil {
			msg := gitops.FormatCommitMessage(
				fmt.Sprintf("note: edit %s", filepath.Base(slug)),
				"",
			)
			_ = repo.Commit(msg)
		}

		cli.Print(out, cli.Success(out, "note updated: "+cli.Accent(out, slug)))
		return nil
	},
}

func init() {
	noteEditCmd.Flags().StringVarP(&noteEditSlugFlag, "slug", "s", "", "note slug to edit")
}
