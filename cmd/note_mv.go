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

	"github.com/spf13/cobra"

	"github.com/retr0h/jot/internal/cli"
)

var (
	noteMvSlugFlag  string
	noteMvTitleFlag string
)

var noteMvCmd = &cobra.Command{
	Use:               "mv [path]",
	Short:             "Rename a note and rewrite wikilinks",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completePaths,
	RunE: func(c *cobra.Command, args []string) error {
		out := c.OutOrStdout()
		notesDir := NotesDir()

		oldSlug := noteArg(args, noteMvSlugFlag)
		if oldSlug == "" {
			picked, err := pickNote(notesDir)
			if err != nil {
				return err
			}
			oldSlug = picked
		}

		newTitle := noteMvTitleFlag
		if newTitle == "" {
			return fmt.Errorf("--title/-t is required")
		}

		newSlug, err := svc.RenameNote(oldSlug, newTitle) // commits the rename
		if err != nil {
			return err
		}

		cli.Print(out, cli.Success(
			out,
			"note renamed: "+cli.Mute(out, oldSlug)+" -> "+cli.Accent(out, newSlug),
		))
		return nil
	},
}

func init() {
	noteMvCmd.Flags().
		StringVarP(&noteMvSlugFlag, "slug", "s", "", "current note slug (fzf if omitted)")
	noteMvCmd.Flags().StringVarP(&noteMvTitleFlag, "title", "t", "", "new note title")
	_ = noteMvCmd.MarkFlagRequired("title")
}
