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
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/retr0h/jot/internal/cli"
)

var taskDuePeriodFlag string

// taskDueCmd implements `jot task due [--period today|week|month]`.
var taskDueCmd = &cobra.Command{
	Use:     "due",
	Aliases: []string{"du"},
	Short:   "Show tasks due within a given period (default: week)",
	Args:    cobra.NoArgs,
	RunE: func(c *cobra.Command, _ []string) error {
		out := c.OutOrStdout()

		period := taskDuePeriodFlag

		now := time.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		todayStr := today.Format("2006-01-02")

		var from, to time.Time
		switch period {
		case "today":
			from = today
			to = today
		case "month":
			from = today
			to = today.AddDate(0, 1, 0)
		default:
			from = today
			to = today.AddDate(0, 0, 7)
		}

		tasks, err := svc.TasksDue(from, to)
		if err != nil {
			return fmt.Errorf("tasks due: %w", err)
		}

		if len(tasks) == 0 {
			return nil
		}

		descW, dueW, noteW := len("DESCRIPTION"), len("DUE"), len("NOTE")
		for _, t := range tasks {
			if len(t.Description) > descW {
				descW = len(t.Description)
			}
			if len(t.DueDate) > dueW {
				dueW = len(t.DueDate)
			}
			if len(t.NoteSlug) > noteW {
				noteW = len(t.NoteSlug)
			}
		}

		cli.Printf(out, "%s\n", cli.Mute(
			out,
			cli.Pad("DUE", dueW+2)+cli.Pad("DESCRIPTION", descW+2)+cli.Pad("NOTE", noteW+2)+"TAGS",
		))

		for _, t := range tasks {
			dueCol := cli.Pad(t.DueDate, dueW+2)
			if t.DueDate < todayStr {
				dueCol = cli.Overdue(out, dueCol)
			} else {
				dueCol = cli.Info(out, dueCol)
			}

			desc := cli.Pad(t.Description, descW+2)
			note := cli.Mute(out, cli.Pad(t.NoteSlug, noteW+2))

			tags := cli.Mute(out, "-")
			if len(t.Tags) > 0 {
				tags = cli.Tag(out, "#"+strings.Join(t.Tags, " #"))
			}

			cli.Printf(out, "%s%s%s%s\n", dueCol, desc, note, tags)
		}

		return nil
	},
}

func init() {
	taskDueCmd.Flags().
		StringVarP(&taskDuePeriodFlag, "period", "p", "week", "time period: today, week, or month")
}
