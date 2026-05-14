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

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/retr0h/jot/internal/cli"
	"github.com/retr0h/jot/internal/jot"
)

var labelFilterFlag string

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List notes, optionally filtered by label",
	Args:  cobra.NoArgs,
	RunE: func(c *cobra.Command, _ []string) error {
		out := c.OutOrStdout()

		store, err := jot.OpenStore(DBPath())
		if err != nil {
			return fmt.Errorf("open store: %w", err)
		}
		defer store.Close()

		notes, err := store.ListNotes(labelFilterFlag)
		if err != nil {
			return fmt.Errorf("list notes: %w", err)
		}

		muted := lipgloss.NewStyle().Faint(true)
		accent := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffb86c"))

		for _, n := range notes {
			date := muted.Render(n.CreatedAt.Format("2006-01-02"))
			title := accent.Render(n.Title)
			lock := ""
			if n.Secure {
				lock = " " + cli.Info(out, "")
			}
			fmt.Fprintf(out, "%s  %s%s\n", date, title, lock)
		}

		return nil
	},
}

func init() {
	listCmd.Flags().StringVar(&labelFilterFlag, "label", "", "filter notes by label name")
	rootCmd.AddCommand(listCmd)
}
