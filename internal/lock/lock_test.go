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

package lock_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/retr0h/jot/internal/lock"
)

// TestWith is not parallel: one row points TMPDIR at a missing directory.
func TestWith(t *testing.T) {
	tests := []struct {
		name string
		// setup prepares dir, returning the dir the first writer locks.
		setup func(t *testing.T, dir string) string
		// second is the dir a second writer locks while the first holds dir;
		// nil skips the second writer.
		second   func(dir string) string
		tmpDir   func(t *testing.T) string
		fnErr    error
		wantWait bool
		wantErr  bool
	}{
		{
			name:     "a second writer to the same dir waits for the first",
			second:   func(dir string) string { return dir },
			wantWait: true,
		},
		{
			name:     "the same dir spelled differently still waits",
			second:   func(dir string) string { return dir + "/." },
			wantWait: true,
		},
		{
			name: "a symlink to the dir shares its lock",
			setup: func(t *testing.T, dir string) string {
				link := filepath.Join(t.TempDir(), "alias")
				if err := os.Symlink(dir, link); err != nil {
					t.Fatalf("symlink: %v", err)
				}
				return link
			},
			second:   func(dir string) string { return dir },
			wantWait: true,
		},
		{
			name: "a repository's lock lives in .git, whatever TMPDIR says",
			setup: func(t *testing.T, dir string) string {
				if err := os.Mkdir(filepath.Join(dir, ".git"), 0o700); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "missing"))
				return dir
			},
			second:   func(dir string) string { return dir },
			wantWait: true,
		},
		{
			name: "a dir that does not exist yet can be locked",
			setup: func(_ *testing.T, dir string) string {
				return filepath.Join(dir, "not-yet")
			},
			second:   func(dir string) string { return filepath.Join(dir, "not-yet") },
			wantWait: true,
		},
		{
			name:   "another dir does not wait",
			second: func(dir string) string { return dir + "-other" },
		},
		{
			name:    "fn's error is returned",
			second:  func(dir string) string { return dir + "-other" },
			fnErr:   errors.New("write failed"),
			wantErr: true,
		},
		{
			name:    "an unwritable temp dir is an error",
			tmpDir:  func(t *testing.T) string { return filepath.Join(t.TempDir(), "missing") },
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			first := dir
			if tt.setup != nil {
				first = tt.setup(t, dir)
			}
			if tt.tmpDir != nil {
				t.Setenv("TMPDIR", tt.tmpDir(t))
			}

			holding := make(chan struct{})
			release := make(chan struct{})
			done := make(chan error, 1)
			go func() {
				done <- lock.With(first, func() error {
					close(holding)
					<-release
					return tt.fnErr
				})
			}()

			if tt.second == nil {
				close(release)
				if err := <-done; err == nil {
					t.Fatal("With: expected error, got nil")
				}
				return
			}
			<-holding

			second := make(chan struct{})
			go func() {
				_ = lock.With(tt.second(dir), func() error {
					close(second)
					return nil
				})
			}()

			select {
			case <-second:
				if tt.wantWait {
					t.Error("second writer ran while the first held the lock")
				}
			case <-time.After(100 * time.Millisecond):
				if !tt.wantWait {
					t.Error("writer to another dir was blocked")
				}
			}

			close(release)
			if err := <-done; (err != nil) != tt.wantErr {
				t.Errorf("With error = %v, wantErr %v", err, tt.wantErr)
			}
			select {
			case <-second:
			case <-time.After(2 * time.Second):
				t.Fatal("second writer never ran after release")
			}
		})
	}
}
