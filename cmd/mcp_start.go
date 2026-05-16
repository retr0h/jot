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
	"log/slog"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	mcppkg "github.com/retr0h/jot/internal/mcp"
)

var mcpStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Run the jot MCP server over stdio",
	Long: `Speaks Model Context Protocol on stdin/stdout — the transport
agents (Claude Code, Cursor, …) expect when they spawn a server as
a subprocess. Blocks until the agent disconnects (the typical MCP
lifecycle); exits cleanly on disconnect.

Logs go to stderr only — stdout is the JSON-RPC wire and writing
anything else there would corrupt the protocol.

  jot mcp start                              # default config dir
  JOT_NOTES_DIR=/tmp/notes jot mcp start    # custom notes directory`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		notesDir := NotesDir()
		if notesDir == "" {
			return fmt.Errorf("mcp start: notes dir could not be determined")
		}

		log := logger.With(slog.String("subsystem", "mcp.start"))
		log.Info(
			"config",
			slog.String("notes_dir", notesDir),
		)

		ctx, cancel := signal.NotifyContext(
			cmd.Context(),
			syscall.SIGINT,
			syscall.SIGTERM,
		)
		defer cancel()

		s, err := mcppkg.New(mcppkg.Config{
			NotesDir: notesDir,
			Logger:   logger,
		})
		if err != nil {
			return fmt.Errorf("mcp start: %w", err)
		}
		defer func() { _ = s.Close() }()

		var srv mcpRunner = s
		return srv.Run(ctx)
	},
}
