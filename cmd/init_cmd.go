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
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/retr0h/jot/internal/cli"
	"github.com/retr0h/jot/internal/jot"
)

// defaultConfig is the template written to jot.yaml on first init.
// All keys are commented out so the file documents the available knobs
// without overriding the built-in defaults until the user opts in.
const defaultConfig = `# jot configuration — uncomment and adjust as needed.
# All values can also be set via JOT_<KEY> environment variables.

# editor: ""          # preferred editor (falls back to $EDITOR / $VISUAL)
# notes_dir: ""       # override the notes directory (default: <config>/notes)
# db_path: ""         # override the database path (default: <config>/jot.db)

# git:
#   enabled: false      # track notes directory in git
#   auto_commit: false  # commit automatically on every write
`

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize jot — create config dir, notes dir, and database",
	RunE: func(_ *cobra.Command, _ []string) error {
		cfgDir := ConfigDir()
		notesDir := NotesDir()
		dbPath := DBPath()
		out := os.Stdout

		// 1. Create config directory.
		if err := os.MkdirAll(cfgDir, 0o700); err != nil {
			return fmt.Errorf("create config dir %q: %w", cfgDir, err)
		}
		fmt.Println(cli.Success(out, "config dir:  "+cli.Accent(out, cfgDir)))

		// 2. Create notes subdirectory.
		if err := os.MkdirAll(notesDir, 0o700); err != nil {
			return fmt.Errorf("create notes dir %q: %w", notesDir, err)
		}
		fmt.Println(cli.Success(out, "notes dir:   "+cli.Accent(out, notesDir)))

		// 3. Write jot.yaml only when it does not already exist so we
		//    never clobber a user's hand-edited config.
		cfgFile := filepath.Join(cfgDir, "jot.yaml")
		if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
			if err := os.WriteFile(cfgFile, []byte(defaultConfig), 0o600); err != nil {
				return fmt.Errorf("write config file %q: %w", cfgFile, err)
			}
			fmt.Println(cli.Success(out, "config file: "+cli.Accent(out, cfgFile)))
		} else {
			fmt.Println(cli.Info(out, "config file already exists, skipping: "+cfgFile))
		}

		// 4. Open / create the SQLite database.
		store, err := jot.OpenStore(dbPath)
		if err != nil {
			return fmt.Errorf("init database %q: %w", dbPath, err)
		}
		store.Close()
		fmt.Println(cli.Success(out, "database:    "+cli.Accent(out, dbPath)))

		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
