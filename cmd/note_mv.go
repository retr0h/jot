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
	"io/fs"
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
	noteMvSlugFlag  string
	noteMvTitleFlag string
)

// noteMvCmd implements `jot note mv --slug <old-slug> --title <new-title>`.
// Renames the note file, rewrites the frontmatter title and first heading,
// and replaces [[old-slug]] wikilinks in all other notes. Auto-commits after.
var noteMvCmd = &cobra.Command{
	Use:     "mv",
	Aliases: []string{"m"},
	Short:   "Rename a note and rewrite wikilinks",
	Args:    cobra.NoArgs,
	RunE: func(c *cobra.Command, _ []string) error {
		out := c.OutOrStdout()
		notesDir := NotesDir()

		oldSlug := noteMvSlugFlag
		if oldSlug == "" {
			picked, err := pickNote(notesDir)
			if err != nil {
				return err
			}
			oldSlug = picked
		}

		newTitle := noteMvTitleFlag
		if newTitle == "" {
			return fmt.Errorf("--title/-t is required")
		}

		newSlug := jot.NewSlug(newTitle, time.Now())

		oldPath := filepath.Join(notesDir, oldSlug+".md")
		newPath := filepath.Join(notesDir, newSlug+".md")

		content, err := os.ReadFile(oldPath)
		if err != nil {
			return fmt.Errorf("read note %q: %w", oldSlug, err)
		}

		// Rewrite frontmatter title and first heading in the content.
		updated := rewriteNoteTitle(string(content), newTitle)

		if err := os.MkdirAll(filepath.Dir(newPath), 0o700); err != nil {
			return fmt.Errorf("create dir for %q: %w", newSlug, err)
		}
		if err := os.WriteFile(newPath, []byte(updated), 0o600); err != nil {
			return fmt.Errorf("write renamed note: %w", err)
		}
		if err := os.Remove(oldPath); err != nil {
			return fmt.Errorf("remove old note: %w", err)
		}

		// Rewrite [[oldSlug]] wikilinks in all other notes.
		_ = rewriteWikiLinks(notesDir, oldSlug, newSlug)

		// Auto-commit.
		if repo, repoErr := gitops.OpenRepo(notesDir); repoErr == nil {
			msg := gitops.FormatCommitMessage(
				fmt.Sprintf("note: rename %s -> %s", oldSlug, newSlug),
				"",
			)
			_ = repo.Commit(msg)
		}

		cli.Print(out, cli.Success(
			out,
			"note renamed: "+cli.Mute(out, oldSlug)+" -> "+cli.Accent(out, newSlug),
		))
		return nil
	},
}

func init() {
	noteMvCmd.Flags().
		StringVarP(&noteMvSlugFlag, "slug", "s", "", "current note slug (fzf if omitted)")
	noteMvCmd.Flags().StringVarP(&noteMvTitleFlag, "title", "t", "", "new note title")
	_ = noteMvCmd.MarkFlagRequired("title")
}

// rewriteNoteTitle replaces the frontmatter `title:` line and the first `# `
// heading with newTitle.
func rewriteNoteTitle(content string, newTitle string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "title:") {
			lines[i] = fmt.Sprintf("title: %q", newTitle)
			continue
		}
		if strings.HasPrefix(line, "# ") {
			lines[i] = "# " + newTitle
		}
	}
	return strings.Join(lines, "\n")
}

// rewriteWikiLinks walks notesDir and replaces [[oldSlug]] with [[newSlug]]
// in every .md file. Errors are silently ignored (best effort).
func rewriteWikiLinks(
	notesDir string,
	oldSlug string,
	newSlug string,
) error {
	return filepath.WalkDir(notesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".md") {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		updated := strings.ReplaceAll(
			string(content),
			"[["+oldSlug+"]]",
			"[["+newSlug+"]]",
		)
		if updated != string(content) {
			_ = os.WriteFile(path, []byte(updated), 0o600)
		}
		return nil
	})
}
