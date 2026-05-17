// Copyright (c) 2026 John Dewey
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
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

	"github.com/retr0h/jot/internal/cli"
	"github.com/retr0h/jot/internal/gitops"
)

var gitLogSlugFlag string

// gitLogCmd implements `jot git log [--slug <slug>]`.
// Opens the notes directory as a git repo and prints the commit log.
// If slug is provided, only commits that touched that file are shown.
var gitLogCmd = &cobra.Command{
	Use:     "log",
	Aliases: []string{"l"},
	Short:   "Show git log for the notes directory",
	Args:    cobra.NoArgs,
	RunE: func(c *cobra.Command, _ []string) error {
		out := c.OutOrStdout()
		notesDir := NotesDir()

		repo, err := gitops.OpenRepo(notesDir)
		if err != nil {
			return fmt.Errorf("open repo: %w", err)
		}

		path := ""
		if gitLogSlugFlag != "" {
			path = gitLogSlugFlag + ".md"
		}

		entries, err := repo.Log(path, 0)
		if err != nil {
			return fmt.Errorf("git log: %w", err)
		}

		for _, e := range entries {
			hash := cli.Hash(out, e.Hash)
			date := cli.Mute(out, e.Date.Format("2006-01-02"))
			cli.Printf(out, "%s  %s  %s\n", hash, e.Message, date)
		}

		return nil
	},
}

func init() {
	gitLogCmd.Flags().
		StringVarP(&gitLogSlugFlag, "slug", "s", "", "filter log to a specific note slug")
}
