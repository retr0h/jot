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
	"context"

	"github.com/spf13/cobra"
)

// mcpRunner is the narrow consumer-seam interface this cobra command depends
// on. Concrete *mcp.Server satisfies it structurally — the compiler verifies
// at the assignment site in mcp_start.go.
type mcpRunner interface {
	Run(ctx context.Context) error
}

// mcpCmd is the parent for `jot mcp` — the Model Context Protocol server
// surface. Spawned per agent session (Claude Code, Cursor, or any
// MCP-aware host), it exposes every jot operation over stdio JSON-RPC.
//
// Subcommands:
//
//	start    open an MCP server over stdio (the typical agent-spawn lifecycle)
var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Run a Model Context Protocol server for jot",
	Long: `Exposes every jot operation — notes, tasks, labels, and search —
over MCP so an LLM agent can read and mutate your notes directly.

The MCP server is spawned by an agent (Claude Code, Cursor, …) per
session. When the agent disconnects the process exits and the SQLite
store is closed cleanly.

Configure your agent (example for Claude Code) to spawn:

  jot mcp start

…and it gets tools for list_notes, create_note, search_notes,
list_tasks, complete_task, list_labels, and more.`,
}

func init() {
	mcpCmd.AddCommand(mcpStartCmd)
	rootCmd.AddCommand(mcpCmd)
}
