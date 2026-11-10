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
)

var (
	taskDoneSlugFlag   string
	taskDoneDescFlag   string
	taskDoneLineFlag   int
	taskDoneForceFlag  bool
	taskDoneReasonFlag string
)

// taskDoneCmd implements `jot task done <id>`, `jot task done --slug <slug>
// --desc <description>` and `jot task done --slug <slug> --line <n>`. Finds
// the open task by its ID, its line, or its description (an exact match,
// else the only one containing it, case-insensitive; several matches name
// their lines), appends done:YYYY-MM-DD and the required --reason, and writes
// the file back. It refuses while a task it waits on (after:) is open, unless
// --force. The service commits it.
var taskDoneCmd = &cobra.Command{
	Use:     "done [id]",
	Short:   "Mark a task as complete",
	Example: "  jot task done t-7k2m -r \"shipped in v1.2\"\n  jot task done -s todo -D \"call Sam\" -r \"left a voicemail\"\n  jot task done t-7k2m -r \"no longer needed\" --force",
	Args:    cobra.MaximumNArgs(1),
	RunE: func(c *cobra.Command, args []string) error {
		out := c.OutOrStdout()
		notesDir := NotesDir()

		slug := taskDoneSlugFlag
		desc := taskDoneDescFlag

		if len(args) == 1 {
			if err := svc.MarkTaskDoneByID(args[0], taskDoneReasonFlag, taskDoneForceFlag); err != nil { // commits too
				return err
			}
			cli.Print(out, cli.Success(out, "task marked done: "+args[0]))
			return nil
		}

		if taskDoneLineFlag > 0 {
			if slug == "" {
				return fmt.Errorf("--line needs --slug")
			}
			if err := svc.MarkTaskDoneAt(slug, taskDoneLineFlag, taskDoneReasonFlag, taskDoneForceFlag); err != nil { // commits too
				return err
			}
			cli.Print(
				out,
				cli.Success(out, fmt.Sprintf("task marked done: line %d", taskDoneLineFlag)),
			)
			return nil
		}

		if slug == "" || desc == "" {
			pickedSlug, pickedDesc, err := pickTask(notesDir)
			if err != nil {
				return err
			}
			if slug == "" {
				slug = pickedSlug
			}
			if desc == "" {
				desc = pickedDesc
			}
		}

		if err := svc.MarkTaskDone(slug, desc, taskDoneReasonFlag, taskDoneForceFlag); err != nil { // commits too
			return err
		}

		cli.Print(out, cli.Success(
			out,
			fmt.Sprintf("task marked done: %s", cli.Accent(out, desc)),
		))
		return nil
	},
}

func init() {
	taskDoneCmd.Flags().
		StringVarP(&taskDoneSlugFlag, "slug", "s", "", "note slug containing the task (fzf if omitted)")
	taskDoneCmd.Flags().
		StringVarP(&taskDoneDescFlag, "desc", "D", "", "task description (fzf if omitted)")
	taskDoneCmd.Flags().
		IntVarP(&taskDoneLineFlag, "line", "l", 0, "the task's 1-based line in the note file; needs --slug")
	taskDoneCmd.Flags().
		BoolVarP(&taskDoneForceFlag, "force", "f", false, "close it even though a task it waits on is still open")
	taskDoneCmd.Flags().
		StringVarP(&taskDoneReasonFlag, "reason", "r", "", "why the task is closed (required)")
	_ = taskDoneCmd.MarkFlagRequired("reason")
}
