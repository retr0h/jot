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

	"github.com/spf13/cobra"

	"github.com/retr0h/jot/internal/cli"
	"github.com/retr0h/jot/internal/jot"
)

// taskCmd is the parent for `jot task` — all task management subcommands.
//
// Subcommands:
//
//	list   list tasks, optionally filtered by status or tag
//	done   mark a task as complete by slug + description
//	due    show tasks due within a time period
var taskCmd = &cobra.Command{
	Use:     "task",
	Aliases: []string{"t"},
	Short:   "Manage tasks",
}

func init() {
	taskCmd.AddCommand(taskListCmd)
	taskCmd.AddCommand(taskDoneCmd)
	taskCmd.AddCommand(taskDueCmd)
	rootCmd.AddCommand(taskCmd)
}

func pickTask(notesDir string) (string, string, error) {
	tasks, err := jot.AllTasks(notesDir, "open", "")
	if err != nil {
		return "", "", fmt.Errorf("list tasks: %w", err)
	}

	items := make([]string, 0, len(tasks))
	for _, t := range tasks {
		items = append(items, fmt.Sprintf("%s\t%s", t.NoteSlug, t.Description))
	}

	selected, err := cli.Pick(items, "select task")
	if err != nil {
		return "", "", err
	}

	parts := strings.SplitN(selected, "\t", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("unexpected selection format")
	}
	return parts[0], parts[1], nil
}
