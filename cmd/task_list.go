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
	"time"

	"github.com/spf13/cobra"

	"github.com/retr0h/jot/internal/cli"
	"github.com/retr0h/jot/internal/jot"
)

var (
	taskListStatusFlag string
	taskListTagFlag    string
)

// taskListCmd implements `jot task list [--status X] [--tag X]`.
// Shows: status checkbox, description, due date, note slug.
var taskListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tasks, optionally filtered by status or tag",
	Args:  cobra.NoArgs,
	RunE: func(c *cobra.Command, _ []string) error {
		out := c.OutOrStdout()

		tasks, err := jot.AllTasks(NotesDir(), taskListStatusFlag, taskListTagFlag)
		if err != nil {
			return fmt.Errorf("list tasks: %w", err)
		}

		todayStr := time.Now().Format("2006-01-02")

		for _, t := range tasks {
			var checkbox string
			if t.Done != "" {
				checkbox = cli.Done(out, "[x]")
			} else {
				checkbox = cli.Accent(out, "[ ]")
			}

			desc := t.Description

			dueStr := ""
			if t.DueDate != "" {
				if t.DueDate < todayStr {
					dueStr = "  " + cli.Overdue(out, t.DueDate)
				} else {
					dueStr = "  " + cli.Mute(out, t.DueDate)
				}
			}

			slug := ""
			if t.NoteSlug != "" {
				slug = "  " + cli.Mute(out, t.NoteSlug)
			}

			fmt.Fprintf(out, "%s %s%s%s\n", checkbox, desc, dueStr, slug)
		}

		return nil
	},
}

func init() {
	taskListCmd.Flags().StringVar(&taskListStatusFlag, "status", "all", "filter by status: open, done, or all")
	taskListCmd.Flags().StringVar(&taskListTagFlag, "tag", "", "filter by tag/label name")
}
