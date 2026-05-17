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
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/retr0h/jot/internal/cli"
	"github.com/retr0h/jot/internal/gitops"
	"github.com/retr0h/jot/internal/jot"
)

var (
	taskDoneSlugFlag string
	taskDoneDescFlag string
)

// taskDoneCmd implements `jot task done --slug <slug> --desc <description>`.
// Finds the @task line in the note file matching description (substring,
// case-insensitive), appends done:YYYY-MM-DD, and writes the file back.
// Auto-commits after.
var taskDoneCmd = &cobra.Command{
	Use:     "done",
	Aliases: []string{"d"},
	Short:   "Mark a task as complete",
	Args:    cobra.NoArgs,
	RunE: func(c *cobra.Command, _ []string) error {
		out := c.OutOrStdout()
		notesDir := NotesDir()

		slug := taskDoneSlugFlag
		desc := taskDoneDescFlag

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

		notePath := filepath.Join(notesDir, slug+".md")

		data, err := os.ReadFile(notePath)
		if err != nil {
			return fmt.Errorf("read note %q: %w", slug, err)
		}
		content := string(data)

		today := time.Now().Format("2006-01-02")
		lower := strings.ToLower(desc)
		tasks := jot.ParseTasks(content)
		resolutions := make(map[int]string)

		for _, t := range tasks {
			if strings.Contains(strings.ToLower(t.Description), lower) && t.DoneDate == "" {
				t.DoneDate = today
				resolutions[t.Line] = jot.ResolveLine(t, t.DueDate, t.Labels)
			}
		}

		if len(resolutions) == 0 {
			return fmt.Errorf("no open task matching %q found in %s", desc, slug)
		}

		updated := jot.ApplyResolutions(content, resolutions)
		if err := os.WriteFile(notePath, []byte(updated), 0o600); err != nil {
			return fmt.Errorf("write note %q: %w", slug, err)
		}

		// Auto-commit.
		if repo, err := gitops.OpenRepo(notesDir); err == nil {
			msg := gitops.FormatCommitMessage(
				fmt.Sprintf("task: done %q in %s", desc, slug),
				"",
			)
			_ = repo.Commit(msg)
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
}
