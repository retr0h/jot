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
	"github.com/retr0h/jot/internal/gitops"
)

var noteEncryptSlugFlag string

var noteEncryptCmd = &cobra.Command{
	Use:     "encrypt",
	Aliases: []string{"enc"},
	Short:   "Encrypt an existing note via kvlt",
	Args:    cobra.NoArgs,
	RunE: func(c *cobra.Command, _ []string) error {
		out := c.OutOrStdout()
		notesDir := NotesDir()

		slug := noteEncryptSlugFlag
		if slug == "" {
			picked, err := pickNote(notesDir)
			if err != nil {
				return err
			}
			slug = picked
		}

		if err := svc.EncryptNote(slug); err != nil {
			return err
		}

		if repo, err := gitops.OpenRepo(notesDir); err == nil {
			msg := gitops.FormatCommitMessage(
				fmt.Sprintf("note(secure): encrypt %s", slug),
				"",
			)
			_ = repo.Commit(msg)
		}

		cli.Print(out, cli.Success(out, "encrypted: "+cli.Accent(out, slug)))
		return nil
	},
}

func init() {
	noteEncryptCmd.Flags().
		StringVarP(&noteEncryptSlugFlag, "slug", "s", "", "note slug to encrypt")
	noteCmd.AddCommand(noteEncryptCmd)
}
