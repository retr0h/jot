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
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/retr0h/jot/internal/jot"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Full-text search across notes",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(c *cobra.Command, args []string) error {
		out := c.OutOrStdout()

		query := strings.Join(args, " ")

		store, err := jot.OpenStore(DBPath())
		if err != nil {
			return fmt.Errorf("open store: %w", err)
		}
		defer store.Close()

		results, err := store.SearchNotes(query)
		if err != nil {
			return fmt.Errorf("search notes: %w", err)
		}

		accent := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffb86c"))
		muted := lipgloss.NewStyle().Faint(true)

		for _, r := range results {
			title := accent.Render(r.Title)
			slug := muted.Render(r.Slug)
			fmt.Fprintf(out, "%s  %s\n", title, slug)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)
}
