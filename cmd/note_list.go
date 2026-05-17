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
	"github.com/retr0h/jot/internal/jot"
)

var noteListTagFlag string

// noteListCmd implements `jot note list [--tag X]`.
var noteListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"l", "ls"},
	Short:   "List notes, optionally filtered by tag",
	Args:    cobra.NoArgs,
	RunE: func(c *cobra.Command, _ []string) error {
		out := c.OutOrStdout()

		notes, err := jot.ListNotes(NotesDir(), noteListTagFlag)
		if err != nil {
			return fmt.Errorf("list notes: %w", err)
		}

		if len(notes) == 0 {
			return nil
		}

		dateW, titleW, slugW := len("CREATED"), len("TITLE"), len("SLUG")
		for _, n := range notes {
			if len(n.Created) > dateW {
				dateW = len(n.Created)
			}
			if len(n.Title) > titleW {
				titleW = len(n.Title)
			}
			if len(n.Slug) > slugW {
				slugW = len(n.Slug)
			}
		}

		hdr := cli.Pad("CREATED", dateW+2) + cli.Pad("TITLE", titleW+2) + "SLUG"
		cli.Printf(out, "%s\n", cli.Mute(out, hdr))

		for _, n := range notes {
			date := cli.Info(out, cli.Pad(n.Created, dateW+2))
			title := cli.Accent(out, cli.Pad(n.Title, titleW+2))
			slug := cli.Mute(out, n.Slug)

			secure := ""
			if n.Secure {
				secure = "  " + cli.Err(out, "[secure]")
			}

			cli.Printf(out, "%s%s%s%s\n", date, title, slug, secure)
		}

		return nil
	},
}

func init() {
	noteListCmd.Flags().
		StringVarP(&noteListTagFlag, "tag", "T", "", "filter notes by tag/label name")
}
