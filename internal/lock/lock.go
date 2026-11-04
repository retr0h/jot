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

// Package lock serializes writers to one notes directory across processes:
// the CLI and every agent's MCP server can edit the same notes at once.
package lock

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// With runs fn while holding an exclusive advisory lock for dir, blocking
// until no other jot writer holds it.
// ponytail: one lock per notes dir; per-note locks if writers ever contend.
func With(
	dir string,
	fn func() error,
) error {
	f, err := os.OpenFile(lockPath(dir), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("lock %q: %w", dir, err)
	}
	defer func() { _ = f.Close() }() // closing the file also releases the lock

	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err == nil {
		err = fn()
	}
	return err
}

// lockPath returns dir's lock file: inside its .git directory when dir is a
// repository, so every path to it (symlinks, aliases, any TMPDIR) shares one
// lock; otherwise in the temp dir, keyed by dir's resolved path.
func lockPath(dir string) string {
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		resolved, _ = filepath.Abs(dir) // a dir that does not exist yet has no links to resolve
	}
	if info, err := os.Stat(filepath.Join(resolved, ".git")); err == nil && info.IsDir() {
		return filepath.Join(resolved, ".git", "jot.lock")
	}
	sum := sha256.Sum256([]byte(resolved))
	return filepath.Join(os.TempDir(), "jot-"+hex.EncodeToString(sum[:8])+".lock")
}
