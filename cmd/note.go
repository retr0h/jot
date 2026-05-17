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

// noteCmd is the parent for `jot note` — all note management subcommands.
//
// Subcommands:
//
//	new      create a new note and open in $EDITOR
//	edit     open an existing note by slug
//	list     list notes, optionally filtered by tag
//	search   full-text search across notes
//	mv       rename a note and rewrite wikilinks
var noteCmd = &cobra.Command{
	Use:     "note",
	Aliases: []string{"n"},
	Short:   "Manage notes",
}

func init() {
	noteCmd.AddCommand(noteNewCmd)
	noteCmd.AddCommand(noteEditCmd)
	noteCmd.AddCommand(noteListCmd)
	noteCmd.AddCommand(noteSearchCmd)
	noteCmd.AddCommand(noteMvCmd)
	rootCmd.AddCommand(noteCmd)
}

func pickNote(notesDir string) (string, error) {
	notes, err := jot.ListNotes(notesDir, "")
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
