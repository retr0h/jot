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

package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Pick pipes items to fzf and returns the selected line trimmed of
// whitespace. Returns an error if fzf is not installed, the user
// cancels (Ctrl-C / Esc), or no items are provided.
func Pick(
	items []string,
	header string,
) (string, error) {
	if len(items) == 0 {
		return "", fmt.Errorf("no items to select from")
	}

	fzf, err := exec.LookPath("fzf")
	if err != nil {
		return "", fmt.Errorf("fzf not found in PATH: %w", err)
	}

	args := []string{"--ansi", "--reverse"}
	if header != "" {
		args = append(args, "--header", header)
	}

	cmd := exec.Command(fzf, args...)
	cmd.Stdin = strings.NewReader(strings.Join(items, "\n"))
	cmd.Stderr = os.Stderr

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("fzf cancelled")
	}

	return strings.TrimSpace(string(out)), nil
}
