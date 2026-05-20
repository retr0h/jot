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
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/retr0h/jot/internal/gitops"
	"github.com/retr0h/jot/internal/jot"
)

var scratchCmd = &cobra.Command{
	Use:     "scratch",
	Aliases: []string{"s"},
	Short:   "Open the scratch note",
	Args:    cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		notesDir := NotesDir()
		scratchPath := filepath.Join(notesDir, "scratch.md")

		if _, err := os.Stat(scratchPath); errors.Is(err, fs.ErrNotExist) {
			content := jot.ScaffoldFrontmatter("Scratch", time.Now().Format("2006-01-02"))
			if err := os.WriteFile(scratchPath, []byte(content), 0o600); err != nil {
				return fmt.Errorf("write scratch note: %w", err)
			}
		}

		if err := jot.Edit(scratchPath, ""); err != nil {
			return fmt.Errorf("edit scratch: %w", err)
		}

		if repo, err := gitops.OpenRepo(notesDir); err == nil {
			_ = repo.Commit(gitops.FormatCommitMessage("note: update scratch", ""))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(scratchCmd)
}
