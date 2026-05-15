// Copyright (c) 2026 John Dewey

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
// DEALINGS IN THE SOFTWARE.

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/retr0h/jot/internal/gitops"
)

var gitShowNoteFlag string
var gitShowCommitFlag string

// gitShowCmd implements `jot git show --commit <hash> [--note slug]`.
// Prints the patch for the given commit hash, optionally filtered to a note.
var gitShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show a specific commit's patch",
	Args:  cobra.NoArgs,
	RunE: func(c *cobra.Command, args []string) error {
		out := c.OutOrStdout()
		commit := gitShowCommitFlag
		notesDir := NotesDir()

		repo, err := gitops.OpenRepo(notesDir)
		if err != nil {
			return fmt.Errorf("open repo: %w", err)
		}

		path := ""
		if gitShowNoteFlag != "" {
			path = gitShowNoteFlag + ".md"
		}

		patch, err := repo.ShowCommit(commit, path)
		if err != nil {
			return fmt.Errorf("git show: %w", err)
		}

		fmt.Fprint(out, patch)
		return nil
	},
}

func init() {
	gitShowCmd.Flags().StringVar(&gitShowCommitFlag, "commit", "", "commit hash to show")
	_ = gitShowCmd.MarkFlagRequired("commit")
	gitShowCmd.Flags().StringVar(&gitShowNoteFlag, "note", "", "filter output to this note slug")
}
