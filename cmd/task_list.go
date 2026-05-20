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

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/retr0h/jot/internal/cli"
	"github.com/retr0h/jot/internal/jot"
)

var (
	taskListStatusFlag string
	taskListTagFlag    string
)

// taskListCmd implements `jot task list [--status X] [--tag X]`.
var taskListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tasks, optionally filtered by status or tag",
	Args:  cobra.NoArgs,
	RunE: func(c *cobra.Command, _ []string) error {
		out := c.OutOrStdout()

		tasks, err := svc.AllTasks(taskListStatusFlag, taskListTagFlag)
		if err != nil {
			return fmt.Errorf("list tasks: %w", err)
		}

		if taskListStatusFlag == "open" {
			todayStr := time.Now().Format("2006-01-02")
			done, _ := svc.AllTasks("done", taskListTagFlag)
			for _, t := range done {
				if t.DoneDate == todayStr {
					tasks = append(tasks, t)
				}
			}
		}

		if len(tasks) == 0 {
			return nil
		}

		now := time.Now()
		todayStr := now.Format("2006-01-02")
		weekStr := now.AddDate(0, 0, 7).Format("2006-01-02")

		type row struct {
			done bool
			desc string
			due  string
			note string
			tags string
		}

		rows := make([]row, 0, len(tasks))
		for _, t := range tasks {
			due := t.DueDate
			if due != "" {
				if resolved, err := jot.ParseDate(due, now); err == nil {
					due = resolved.Format("2006-01-02")
				}
			}
			tags := ""
			if len(t.Tags) > 0 {
				tags = "#" + strings.Join(t.Tags, " #")
			}
			rows = append(rows, row{
				done: t.Done,
				desc: t.Description,
				due:  due,
				note: t.NoteSlug,
				tags: tags,
			})
		}

		descW, dueW, noteW := len("DESCRIPTION"), len("DUE"), len("NOTE")
		for _, r := range rows {
			if len(r.desc) > descW {
				descW = len(r.desc)
			}
			if len(r.due) > dueW {
				dueW = len(r.due)
			}
			if len(r.note) > noteW {
				noteW = len(r.note)
			}
		}

		cli.Printf(out, "%s\n", cli.Mute(
			out,
			cli.Pad(
				"STATUS",
				8,
			)+cli.Pad(
				"DESCRIPTION",
				descW+2,
			)+cli.Pad(
				"DUE",
				dueW+2,
			)+cli.Pad(
				"NOTE",
				noteW+2,
			)+"TAGS",
		))

		theme := cli.ActiveTheme()
		for _, r := range rows {
			var rowBg *lipgloss.Style
			switch {
			case r.due != "" && r.due == todayStr:
				bg := cli.RowBgToday()
				rowBg = &bg
			case r.due != "" && r.due > todayStr && r.due <= weekStr:
				bg := cli.RowBgSoon()
				rowBg = &bg
			}

			renderCol := func(style lipgloss.Style, s string) string {
				if rowBg != nil {
					return cli.RenderWithBg(out, style, *rowBg, s)
				}
				return style.Renderer(lipgloss.DefaultRenderer()).Render(s)
			}

			var status string
			if r.done {
				status = renderCol(theme.OK, cli.Pad("[x]", 8))
			} else {
				status = renderCol(theme.Accent, cli.Pad("[ ]", 8))
			}

			desc := cli.Pad(r.desc, descW+2)
			if rowBg != nil {
				desc = cli.RenderWithBg(out, lipgloss.NewStyle(), *rowBg, desc)
			}

			padded := cli.Pad(r.due, dueW+2)
			var dueCol string
			switch {
			case r.due == "":
				dueCol = renderCol(theme.Mute, cli.Pad("-", dueW+2))
			case r.due < todayStr:
				dueCol = renderCol(theme.Err, padded)
			case r.due == todayStr:
				dueCol = renderCol(theme.Warn, padded)
			default:
				dueCol = renderCol(theme.Info, padded)
			}

			note := renderCol(theme.Mute, cli.Pad(r.note, noteW+2))

			tagCol := renderCol(theme.Mute, "-")
			if r.tags != "" {
				tagCol = renderCol(theme.Tag, r.tags)
			}

			cli.Printf(out, "%s%s%s%s%s\n", status, desc, dueCol, note, tagCol)
		}

		return nil
	},
}

func init() {
	taskListCmd.Flags().
		StringVarP(&taskListStatusFlag, "status", "S", "open", "filter by status: open, done, or all")
	taskListCmd.Flags().StringVarP(&taskListTagFlag, "tag", "T", "", "filter by tag/label name")
}
