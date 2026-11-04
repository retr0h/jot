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
	taskAddSlugFlag  string
	taskAddDescFlag  string
	taskAddTitleFlag string
	taskAddDueFlag   string
	taskAddDoneFlag  string
	taskAddTagFlags  []string
)

// taskAddCmd implements `jot task add --slug <slug> --desc <description>`.
// Appends a checkbox task to the note, creating the note when it does not
// exist yet; the service commits it.
var taskAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a task to a note, creating the note if needed",
	Example: `  jot task add -s todo -D "call Sam" --due friday -t work
  jot task add -s done/2026-11-03 --title "Done 2026-11-03" -D "Merged !20" --done -t cos
  jot task add -s done/2026-11-02 -D "Opened !19" --done=2026-11-02 -t cos`,
	Args: cobra.NoArgs,
	RunE: func(c *cobra.Command, _ []string) error {
		out := c.OutOrStdout()

		task, err := jot.NewTask(
			taskAddDescFlag,
			taskAddDueFlag,
			taskAddDoneFlag,
			taskAddTagFlags,
			time.Now(),
		)
		if err != nil {
			return err
		}
		if err := svc.AddTask(taskAddSlugFlag, taskAddTitleFlag, task); err != nil { // commits too
			return err
		}

		cli.Print(out, cli.Success(
			out,
			fmt.Sprintf("task added: %s", cli.Accent(out, task.Description)),
		))
		return nil
	},
}

func init() {
	taskAddCmd.Flags().
		StringVarP(&taskAddSlugFlag, "slug", "s", "", "note slug to add the task to (created if missing)")
	taskAddCmd.Flags().
		StringVarP(&taskAddDescFlag, "desc", "D", "", "task description: one line, no | or #tags")
	taskAddCmd.Flags().
		StringVar(&taskAddTitleFlag, "title", "", "title for a new note (default: the slug's last segment)")
	taskAddCmd.Flags().
		StringVar(&taskAddDueFlag, "due", "", "due date: YYYY-MM-DD, today, tomorrow, friday, next week")
	taskAddCmd.Flags().
		StringVar(&taskAddDoneFlag, "done", "", "add it already done: bare --done is today, --done=DATE another day")
	taskAddCmd.Flags().Lookup("done").NoOptDefVal = "today"
	taskAddCmd.Flags().
		StringSliceVarP(&taskAddTagFlags, "tag", "t", nil, "tag, repeatable or comma-separated")
	_ = taskAddCmd.MarkFlagRequired("slug")
	_ = taskAddCmd.MarkFlagRequired("desc")
}
