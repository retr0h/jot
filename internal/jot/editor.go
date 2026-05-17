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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// ErrEditorUnset is returned when neither $VISUAL nor $EDITOR is set.
var ErrEditorUnset = errors.New("$EDITOR is not set — jot requires a configured editor")

// EditorName resolves the editor from $VISUAL or $EDITOR.
func EditorName() (string, error) {
	if e := os.Getenv("VISUAL"); e != "" {
		return e, nil
	}
	if e := os.Getenv("EDITOR"); e != "" {
		return e, nil
	}
	return "", ErrEditorUnset
}

// Edit opens path in the resolved editor. JOT_NOTES_DIR is set in the
// child environment so editor plugins can locate the notes directory.
func Edit(path string) error {
	editor, err := EditorName()
	if err != nil {
		return err
	}
	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "JOT_NOTES_DIR="+filepath.Dir(path))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("editor %s: %w", editor, err)
	}
	return nil
}
