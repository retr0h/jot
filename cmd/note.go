// Copyright (c) 2026 John Dewey
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
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
	"path"
	"strings"

	"github.com/spf13/cobra"

	"github.com/retr0h/jot/internal/cli"
)

// noteCmd is the parent for `jot note` — all note management subcommands.
var noteCmd = &cobra.Command{
	Use:   "note",
	Short: "Manage notes",
}

func init() {
	noteCmd.AddCommand(noteNewCmd)
	noteCmd.AddCommand(noteEditCmd)
	noteCmd.AddCommand(noteListCmd)
	noteCmd.AddCommand(noteSearchCmd)
	noteCmd.AddCommand(noteMvCmd)
	rootCmd.AddCommand(noteCmd)
}

// noteArg returns the note path given as the command's argument, else the
// --slug flag's value.
func noteArg(args []string, flag string) string {
	if len(args) > 0 {
		return args[0]
	}
	return flag
}

// completePaths tab-completes a command's one path argument from the notes
// root rather than the filesystem's: every folder ("/work/") and note
// ("/work/sensor-ideas").
func completePaths(
	_ *cobra.Command,
	args []string,
	_ string,
) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	notes, err := svc.ListNotes("", "")
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	seen := make(map[string]bool)
	paths := make([]string, 0, len(notes))
	for _, n := range notes {
		paths = append(paths, "/"+n.Slug)
		for d := path.Dir(n.Slug); d != "." && !seen[d]; d = path.Dir(d) {
			seen[d] = true
			paths = append(paths, "/"+d+"/")
		}
	}
	return paths, cobra.ShellCompDirectiveNoFileComp
}

func pickNote(_ string) (string, error) {
	notes, err := svc.ListNotes("", "")
	if err != nil {
		return "", fmt.Errorf("list notes: %w", err)
	}

	items := make([]string, 0, len(notes))
	for _, n := range notes {
		items = append(items, fmt.Sprintf("%s  %s", n.Slug, n.Title))
	}

	selected, err := cli.Pick(items, "select note")
	if err != nil {
		return "", err
	}

	slug, _, _ := strings.Cut(selected, "  ")
	return slug, nil
}
