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
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/retr0h/jot/internal/cli"
	"github.com/retr0h/jot/internal/jot"
)

var (
	taskListStatusFlag    string
	taskListTagFlag       string
	taskListDoneSinceFlag string
)

// taskListCmd implements `jot task list [path] [--status X] [--tag X] [--done-since D]`.
// path narrows it to the notes under a folder, a note, or a glob, as in note ls.
var taskListCmd = &cobra.Command{
	Use:               "list [path]",
	Short:             "List tasks, optionally filtered by path, status, tag or done date",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completePaths,
	RunE: func(c *cobra.Command, args []string) error {
		out := c.OutOrStdout()
		dir := noteArg(args, "")

		status := taskListStatusFlag
		if taskListDoneSinceFlag != "" {
			if _, err := time.Parse("2006-01-02", taskListDoneSinceFlag); err != nil {
				return fmt.Errorf("--done-since must be a YYYY-MM-DD date")
			}
			status = "done"
		}

		tasks, err := svc.AllTasks(status, taskListTagFlag, dir)
		if err != nil {
			return fmt.Errorf("list tasks: %w", err)
		}
		if taskListDoneSinceFlag != "" {
			tasks = jot.DoneSince(tasks, taskListDoneSinceFlag)
		}

		if status == "open" {
			todayStr := time.Now().Format("2006-01-02")
			done, _ := svc.AllTasks("done", taskListTagFlag, dir)
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

		// Blockers and duplicate IDs can sit outside the filtered list.
		all, err := svc.AllTasks("all", "", "")
		if err != nil {
			return fmt.Errorf("list tasks: %w", err)
		}
		byID := jot.IndexByID(all)

		type row struct {
			done  bool
			id    string
			dup   bool
			desc  string
			due   string
			after string // open blockers' IDs
			note  string
			tags  string
		}

		rows := make([]row, 0, len(tasks))
		for _, t := range tasks {
			var after []string
			if !t.Done {
				for _, b := range jot.OpenBlockers(jot.RawTask{After: t.After}, byID) {
					after = append(after, b.ID)
				}
			}
			due := t.DueDate
			if due != "" {
				if resolved, err := jot.ParseDate(due, now); err == nil {
					due = resolved.Format("2006-01-02")
				}
			}
			desc := t.Description
			if t.Reason != "" {
				desc += " (" + t.Reason + ")"
			}
			tags := ""
			if len(t.Tags) > 0 {
				tags = "#" + strings.Join(t.Tags, " #")
			}
			rows = append(rows, row{
				done:  t.Done,
				id:    t.ID,
				dup:   len(byID[t.ID]) > 1,
				desc:  desc,
				due:   due,
				after: strings.Join(after, ","),
				note:  "/" + t.NoteSlug,
				tags:  tags,
			})
		}

		idW, descW, dueW, afterW, noteW := len(
			"ID",
		), len(
			"DESCRIPTION",
		), len(
			"DUE",
		), len(
			"AFTER",
		), len(
			"NOTE",
		)
		for _, r := range rows {
			idW = max(idW, len(r.id))
			descW = max(descW, len(r.desc))
			dueW = max(dueW, len(r.due))
			afterW = max(afterW, len(r.after))
			noteW = max(noteW, len(r.note))
		}

		cli.Printf(out, "%s\n", cli.Mute(
			out,
			cli.Pad("STATUS", 8)+
				cli.Pad("ID", idW+2)+
				cli.Pad("DESCRIPTION", descW+2)+
				cli.Pad("DUE", dueW+2)+
				cli.Pad("AFTER", afterW+2)+
				cli.Pad("NOTE", noteW+2)+
				"TAGS",
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

			idCol := renderCol(theme.Mute, cli.Pad(r.id, idW+2))
			if r.dup {
				idCol = renderCol(theme.Err, cli.Pad(r.id, idW+2))
			}

			afterCol := renderCol(theme.Mute, cli.Pad("-", afterW+2))
			if r.after != "" {
				afterCol = renderCol(theme.Warn, cli.Pad(r.after, afterW+2))
			}

			note := renderCol(theme.Mute, cli.Pad(r.note, noteW+2))

			tagCol := renderCol(theme.Mute, "-")
			if r.tags != "" {
				tagCol = renderCol(theme.Tag, r.tags)
			}

			cli.Printf(out, "%s%s%s%s%s%s%s\n", status, idCol, desc, dueCol, afterCol, note, tagCol)
		}

		for _, id := range slices.Sorted(maps.Keys(byID)) {
			if n := len(byID[id]); n > 1 {
				cli.Printf(
					out,
					"%s\n",
					cli.Err(
						out,
						fmt.Sprintf("id %s is on %d tasks; give one of them a new id", id, n),
					),
				)
			}
		}

		return nil
	},
}

func init() {
	taskListCmd.Flags().
		StringVarP(&taskListStatusFlag, "status", "S", "open", "filter by status: open, done, or all")
	taskListCmd.Flags().StringVarP(&taskListTagFlag, "tag", "T", "", "filter by tag/label name")
	taskListCmd.Flags().
		StringVar(&taskListDoneSinceFlag, "done-since", "", "only tasks done on or after this date, YYYY-MM-DD")
}
