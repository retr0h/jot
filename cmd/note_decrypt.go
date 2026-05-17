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
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/retr0h/jot/internal/cli"
	"github.com/retr0h/jot/internal/gitops"
	"github.com/retr0h/jot/internal/jot"
)

var noteDecryptSlugFlag string

var noteDecryptCmd = &cobra.Command{
	Use:     "decrypt",
	Aliases: []string{"dec"},
	Short:   "Decrypt a secure note back to plaintext",
	Args:    cobra.NoArgs,
	RunE: func(c *cobra.Command, _ []string) error {
		out := c.OutOrStdout()
		notesDir := NotesDir()
		cfgDir := ConfigDir()

		slug := noteDecryptSlugFlag
		if slug == "" {
			picked, err := pickNote(notesDir)
			if err != nil {
				return err
			}
			slug = picked
		}

		note, err := jot.FindNote(notesDir, slug)
		if err != nil {
			return err
		}
		if !note.Secure {
			return fmt.Errorf("note %q is not encrypted", slug)
		}

		store, err := jot.NewSecureStore(cfgDir, SSHKeys(), cli.PassphrasePrompt)
		if err != nil {
			return fmt.Errorf("open secure store: %w", err)
		}

		body, err := store.ReadNote(context.Background(), slug)
		if err != nil {
			return err
		}

		content := fmt.Sprintf(
			"---\ntitle: %q\ntags: [%s]\ncreated: %s\n---\n\n%s",
			note.Title,
			formatTags(note.Tags),
			note.Created,
			body,
		)
		if err := os.WriteFile(note.Path, []byte(content), 0o600); err != nil {
			return fmt.Errorf("rewrite note file: %w", err)
		}

		if err := store.DeleteNote(context.Background(), slug); err != nil {
			return fmt.Errorf("remove from secure store: %w", err)
		}

		if repo, err := gitops.OpenRepo(notesDir); err == nil {
			msg := gitops.FormatCommitMessage(
				fmt.Sprintf("note: decrypt %s", slug),
				"",
			)
			_ = repo.Commit(msg)
		}

		cli.Print(out, cli.Success(out, "decrypted: "+cli.Accent(out, slug)))
		return nil
	},
}

func init() {
	noteDecryptCmd.Flags().
		StringVarP(&noteDecryptSlugFlag, "slug", "s", "", "note slug to decrypt")
	noteCmd.AddCommand(noteDecryptCmd)
}
