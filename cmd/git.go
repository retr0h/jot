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
	"strings"

	"github.com/spf13/cobra"

	"github.com/retr0h/jot/internal/cli"
	"github.com/retr0h/jot/internal/gitops"
)

// gitCmd is the parent for `jot git` — version control subcommands over the
// notes directory.
//
// Subcommands:
//
//	log    show commit history, optionally filtered to a note slug
//	diff   show working-tree diff
//	show   show a specific commit's patch
var gitCmd = &cobra.Command{
	Use:     "git",
	Aliases: []string{"g"},
	Short:   "Git operations on the notes directory",
}

func init() {
	gitCmd.AddCommand(gitLogCmd)
	gitCmd.AddCommand(gitDiffCmd)
	gitCmd.AddCommand(gitShowCmd)
	rootCmd.AddCommand(gitCmd)
}

func pickCommit(notesDir string) (string, error) {
	repo, err := gitops.OpenRepo(notesDir)
	if err != nil {
		return "", fmt.Errorf("open repo: %w", err)
	}

	entries, err := repo.Log("", 50)
	if err != nil {
		return "", fmt.Errorf("git log: %w", err)
	}

	items := make([]string, 0, len(entries))
	for _, e := range entries {
		items = append(
			items,
			fmt.Sprintf("%s  %s  %s", e.Hash[:8], e.Message, e.Date.Format("2006-01-02")),
		)
	}

	selected, err := cli.Pick(items, "select commit")
	if err != nil {
		return "", err
	}

	hash, _, _ := strings.Cut(selected, "  ")
	return hash, nil
}
