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
	"io"
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
	noteSecureFlag   bool
	noteNewTitleFlag string
)

// noteNewCmd implements `jot note new --title <title> [--secure]`.
// A "/" in the title splits into subdir + title so
// `jot note new --title "network/Switch Config"` stores the note under notes/network/.
var noteNewCmd = &cobra.Command{
	Use:   "new",
	Short: "Create a new note and open it in your editor",
	Args:  cobra.NoArgs,
	RunE: func(c *cobra.Command, _ []string) error {
		out := c.OutOrStdout()
		rawTitle := noteNewTitleFlag
		notesDir := NotesDir()
		piped := cli.IsPiped()

		// No title — open scratch.md.
		if rawTitle == "" {
			if piped {
				return fmt.Errorf("--title is required when reading from stdin")
			}
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
		}

		// The service creates the note ("subdir/Title" puts it in a folder),
		// refuses to overwrite an existing one, and commits it.
		title := rawTitle[strings.LastIndex(rawTitle, "/")+1:]

		if piped {
			raw, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("read stdin: %w", err)
			}
			content := strings.TrimRight(string(raw), "\n")
			if !noteSecureFlag {
				content = jot.ScaffoldFrontmatter(
					title,
					time.Now().Format("2006-01-02"),
				) + content + "\n"
			}
			slug, _, err := svc.CreateNote(rawTitle, content, noteSecureFlag)
			if err != nil {
				return err
			}
			cli.Print(out, cli.Success(out, "note created: "+cli.Accent(out, slug)))
			return nil
		}

		slug, notePath, err := svc.CreateNote(rawTitle, "", false)
		if err != nil {
			return err
		}
		if err := jot.Edit(notePath, ""); err != nil {
			return fmt.Errorf("edit note: %w", err)
		}
		if repo, err := gitops.OpenRepo(notesDir); err == nil {
			_ = repo.Commit(gitops.FormatCommitMessage("note: edit "+filepath.Base(slug), ""))
		}

		cli.Print(out, cli.Success(out, "note created: "+cli.Accent(out, slug)))
		return nil
	},
}

func init() {
	noteNewCmd.Flags().
		StringVarP(&noteNewTitleFlag, "title", "t", "", "note title (use path/title for subdirs; omit for scratch)")
	noteNewCmd.Flags().
		BoolVarP(&noteSecureFlag, "secure", "x", false, "mark note as secure (sets secure: true in front-matter)")
}
