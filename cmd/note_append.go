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
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/retr0h/jot/internal/cli"
)

var (
	noteAppendSlugFlag string
	noteAppendTextFlag string
)

// noteAppendCmd implements `jot note append --slug <slug> --text <text>`,
// reading the text from stdin when piped. Adds it to the end of an existing
// note's body; the service commits it.
var noteAppendCmd = &cobra.Command{
	Use:   "append",
	Short: "Append text to the end of an existing note",
	Example: `  jot note append -s memory/projects --text "jot is the shared agent store"
  git log -1 --format=%s | jot note append -s worklog`,
	Args: cobra.NoArgs,
	RunE: func(c *cobra.Command, _ []string) error {
		out := c.OutOrStdout()

		text := noteAppendTextFlag
		if text == "" && cli.IsPiped() {
			raw, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("read stdin: %w", err)
			}
			text = string(raw)
		}
		if err := svc.AppendNote(noteAppendSlugFlag, text); err != nil { // commits too
			return err
		}

		cli.Print(out, cli.Success(out, "appended to "+cli.Accent(out, noteAppendSlugFlag)))
		return nil
	},
}

func init() {
	noteAppendCmd.Flags().
		StringVarP(&noteAppendSlugFlag, "slug", "s", "", "slug of the note to append to")
	noteAppendCmd.Flags().
		StringVar(&noteAppendTextFlag, "text", "", "text to append (read from stdin when piped and omitted)")
	_ = noteAppendCmd.MarkFlagRequired("slug")
	noteCmd.AddCommand(noteAppendCmd)
}
