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

package jot

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Edit opens path in nvim. When lineNum is non-empty nvim opens at that
// line (+N). JOT_NOTES_DIR is set in the child environment so editor
// plugins (obsidian.nvim) can locate the notes directory.
func Edit(
	path string,
	lineNum string,
) error {
	args := []string{path}
	if lineNum != "" {
		args = []string{"+" + lineNum, path}
	}
	cmd := exec.Command("nvim", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "JOT_NOTES_DIR="+filepath.Dir(path))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("nvim: %w", err)
	}
	return nil
}
