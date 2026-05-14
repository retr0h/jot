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
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/retr0h/jot/internal/jot"
)

var dueCmd = &cobra.Command{
	Use:   "due [today|week|month]",
	Short: "Show tasks due within a given period (default: week)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(c *cobra.Command, args []string) error {
		out := c.OutOrStdout()

		period := "week"
		if len(args) == 1 {
			period = args[0]
		}

		store, err := jot.OpenStore(DBPath())
		if err != nil {
			return fmt.Errorf("open store: %w", err)
		}
		defer store.Close()

		now := time.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

		var from, to time.Time
		switch period {
		case "today":
			from = today
			to = today
		case "month":
			from = today
			to = today.AddDate(0, 1, 0)
		default: // "week" and unrecognized values
			from = today
			to = today.AddDate(0, 0, 7)
		}

		tasks, err := store.TasksDue(from, to)
		if err != nil {
			return fmt.Errorf("tasks due: %w", err)
		}

		muted := lipgloss.NewStyle().Faint(true)
		accent := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffb86c"))
		overdue := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff6ec7"))

		for _, t := range tasks {
			var dateStr string
			if t.DueDate != nil {
				dateStr = t.DueDate.Format("2006-01-02")
				if t.DueDate.Before(today) {
					dateStr = overdue.Render(dateStr)
				} else {
					dateStr = muted.Render(dateStr)
				}
			} else {
				dateStr = muted.Render("no due date")
			}

			desc := accent.Render(t.Description)
			taskID := muted.Render(fmt.Sprintf("#%d", t.ID))
			fmt.Fprintf(out, "%s  %s  %s\n", dateStr, desc, taskID)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(dueCmd)
}
