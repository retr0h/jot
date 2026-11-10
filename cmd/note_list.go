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

	"github.com/spf13/cobra"

	"github.com/retr0h/jot/internal/cli"
	"github.com/retr0h/jot/internal/jot"
)

var (
	noteListTagFlag       string
	noteListRecursiveFlag bool
)

// noteListCmd implements `jot note ls [path] [-R] [--tag X]`. Like ls, it
// lists one level of a folder from the notes root ("/work"): its folders,
// each with how many notes it holds, then its notes. -R lists every note
// under it instead. path may also be a note, or a glob ("/work/*-ideas");
// quote a glob so the shell leaves it alone.
var noteListCmd = &cobra.Command{
	Use:               "list [path]",
	Aliases:           []string{"ls"},
	Short:             "List the folders and notes in a path, optionally filtered by tag",
	Example:           "  jot note ls\n  jot note ls /work\n  jot note ls -R /work\n  jot note ls '/work/*-ideas' -T ops",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completePaths,
	RunE: func(c *cobra.Command, args []string) error {
		out := c.OutOrStdout()
		dir := noteArg(args, "")

		var entries []jot.Entry
		if noteListRecursiveFlag {
			notes, err := svc.ListNotes(noteListTagFlag, dir)
			if err != nil {
				return fmt.Errorf("list notes: %w", err)
			}
			for _, n := range notes {
				entries = append(entries, jot.Entry{Note: n})
			}
		} else {
			notes, err := svc.ListNotes(noteListTagFlag, "")
			if err != nil {
				return fmt.Errorf("list notes: %w", err)
			}
			if entries, err = jot.DirEntries(notes, dir); err != nil {
				return fmt.Errorf("list notes: %w", err)
			}
		}

		if len(entries) == 0 {
			if dir != "" {
				return fmt.Errorf("no notes match %s", dir)
			}
			return nil
		}

		type row struct {
			created, title, where, tags string
			folder, secure              bool
		}
		rows := make([]row, 0, len(entries))
		for _, e := range entries {
			if e.Note == nil {
				noun := "notes"
				if e.Count == 1 {
					noun = "note"
				}
				rows = append(rows, row{
					created: "-",
					title:   fmt.Sprintf("%d %s", e.Count, noun),
					where:   "/" + e.Dir + "/",
					tags:    "-",
					folder:  true,
				})
				continue
			}
			n := e.Note
			tags := "-"
			if len(n.Tags) > 0 {
				tags = "#" + strings.Join(n.Tags, " #")
			}
			rows = append(rows, row{
				created: n.Created,
				title:   n.Title,
				where:   "/" + n.Slug,
				tags:    tags,
				secure:  n.Secure,
			})
		}

		dateW, titleW, whereW := len("CREATED"), len("TITLE"), len("PATH")
		for _, r := range rows {
			dateW = max(dateW, len(r.created))
			titleW = max(titleW, len(r.title))
			whereW = max(whereW, len(r.where))
		}

		cli.Printf(out, "%s\n", cli.Mute(out, cli.Pad("CREATED", dateW+2)+
			cli.Pad("TITLE", titleW+2)+
			cli.Pad("PATH", whereW+2)+
			"TAGS"))

		for _, r := range rows {
			date := cli.Info(out, cli.Pad(r.created, dateW+2))
			title := cli.Accent(out, cli.Pad(r.title, titleW+2))
			where := cli.Mute(out, cli.Pad(r.where, whereW+2))
			if r.folder {
				date = cli.Mute(out, cli.Pad(r.created, dateW+2))
				title = cli.Mute(out, cli.Pad(r.title, titleW+2))
				where = cli.Info(out, cli.Pad(r.where, whereW+2))
			}

			tags := cli.Mute(out, r.tags)
			if r.tags != "-" {
				tags = cli.Tag(out, r.tags)
			}

			secure := ""
			if r.secure {
				secure = "  " + cli.Err(out, "[secure]")
			}

			cli.Printf(out, "%s%s%s%s%s\n", date, title, where, tags, secure)
		}

		return nil
	},
}

func init() {
	noteListCmd.Flags().
		StringVarP(&noteListTagFlag, "tag", "T", "", "filter notes by tag/label name")
	noteListCmd.Flags().
		BoolVarP(&noteListRecursiveFlag, "recursive", "R", false, "list every note under the path, not one level")
}
