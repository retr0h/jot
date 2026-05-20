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
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/retr0h/jot/internal/cli"
	"github.com/retr0h/jot/internal/gitops"
	"github.com/retr0h/jot/internal/jot"
)

var noteSearchQueryFlag string

var noteSearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Grep notes with ripgrep and open selection in nvim",
	Args:  cobra.ArbitraryArgs,
	RunE: func(c *cobra.Command, args []string) error {
		out := c.OutOrStdout()
		notesDir := NotesDir()

		if noteSearchQueryFlag == "" && len(args) > 0 {
			noteSearchQueryFlag = strings.Join(args, " ")
		}

		rg, err := exec.LookPath("rg")
		if err != nil {
			return fmt.Errorf("rg not found in PATH: %w", err)
		}
		fzf, err := exec.LookPath("fzf")
		if err != nil {
			return fmt.Errorf("fzf not found in PATH: %w", err)
		}

		rgArgs := []string{
			"--no-config",
			"--type=md",
			"--smart-case",
			"--color=always",
			"--no-heading",
			"--line-number",
			"--with-filename",
		}
		if noteSearchQueryFlag != "" {
			rgArgs = append(rgArgs, "--fixed-strings", "-e", noteSearchQueryFlag)
		} else {
			rgArgs = append(rgArgs, ".")
		}
		rgArgs = append(rgArgs, notesDir)

		rgCmd := exec.Command(rg, rgArgs...)

		fzfArgs := []string{
			"--ansi",
			"--reverse",
			"--delimiter=:",
			"--preview", "cat {1}",
			"--preview-window", "right:50%:wrap:+{2}-5",
			"--header", "search notes (enter to open)",
		}
		if noteSearchQueryFlag != "" {
			fzfArgs = append(fzfArgs, "--query", noteSearchQueryFlag)
		}

		fzfCmd := exec.Command(fzf, fzfArgs...)
		fzfCmd.Stderr = os.Stderr

		pipe, err := rgCmd.StdoutPipe()
		if err != nil {
			return fmt.Errorf("pipe rg→fzf: %w", err)
		}
		fzfCmd.Stdin = pipe

		if err := rgCmd.Start(); err != nil {
			return fmt.Errorf("start rg: %w", err)
		}

		selection, err := fzfCmd.Output()
		_ = rgCmd.Wait()
		if err != nil {
			return nil
		}

		line := strings.TrimSpace(string(selection))
		if line == "" {
			return nil
		}

		// rg output format: /path/to/file.md:linenum:matched text
		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 2 {
			return nil
		}

		filePath := parts[0]
		lineNum := parts[1]

		if err := jot.Edit(filePath, lineNum); err != nil {
			return fmt.Errorf("edit note: %w", err)
		}

		slug := strings.TrimSuffix(filepath.Base(filePath), ".md")

		if repo, err := gitops.OpenRepo(notesDir); err == nil {
			msg := gitops.FormatCommitMessage(
				fmt.Sprintf("note: edit %s", slug),
				"",
			)
			_ = repo.Commit(msg)
		}

		cli.Print(out, cli.Success(out, "note updated: "+cli.Accent(out, slug)))
		return nil
	},
}

func init() {
	noteSearchCmd.Flags().
		StringVarP(&noteSearchQueryFlag, "query", "q", "", "search query (optional; omit for live grep)")
}
