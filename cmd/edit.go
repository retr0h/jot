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
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/retr0h/jot/internal/cli"
	"github.com/retr0h/jot/internal/jot"
)

var editCmd = &cobra.Command{
	Use:   "edit <id|slug>",
	Short: "Open an existing note in your editor",
	Args:  cobra.ExactArgs(1),
	RunE: func(c *cobra.Command, args []string) error {
		out := c.OutOrStdout()

		store, err := jot.OpenStore(DBPath())
		if err != nil {
			return fmt.Errorf("open store: %w", err)
		}
		defer store.Close()

		var note *jot.Note
		if id, err := strconv.ParseInt(args[0], 10, 64); err == nil {
			note, err = store.GetNote(id)
			if err != nil {
				return fmt.Errorf("get note %d: %w", id, err)
			}
		} else {
			note, err = store.GetNoteBySlug(args[0])
			if err != nil {
				return fmt.Errorf("get note %q: %w", args[0], err)
			}
		}

		notePath := filepath.Join(NotesDir(), note.Slug+".md")

		oldBytes, err := os.ReadFile(notePath)
		if err != nil {
			return fmt.Errorf("read note file %q: %w", notePath, err)
		}
		oldContent := string(oldBytes)

		if err := jot.Edit(notePath); err != nil {
			return fmt.Errorf("edit note: %w", err)
		}

		newBytes, err := os.ReadFile(notePath)
		if err != nil {
			return fmt.Errorf("read note file after edit %q: %w", notePath, err)
		}
		newContent := string(newBytes)

		added, _, _ := jot.DiffTasks(oldContent, newContent)
		resolutions := make(map[int]string)

		for _, raw := range added {
			if raw.Resolved {
				task, err := store.CreateTask(note.ID, raw.Description, raw.Line)
				if err != nil {
					return fmt.Errorf("create task %q: %w", raw.Description, err)
				}

				if raw.DueDate != "" {
					due, err := jot.ParseDate(raw.DueDate, time.Now())
					if err == nil {
						_ = store.SetTaskDue(task.ID, &due)
					}
				}

				for _, labelName := range raw.Labels {
					label, err := store.CreateLabel(labelName)
					if err != nil {
						labels, lerr := store.ListLabels()
						if lerr != nil {
							continue
						}
						for _, l := range labels {
							if l.Name == labelName {
								label = l
								break
							}
						}
						if label == nil {
							continue
						}
					}
					_ = store.AddTaskLabel(task.ID, label.ID)
				}
			} else {
				p := tea.NewProgram(jot.NewTaskPrompt(raw.Description, nil))
				result, err := p.Run()
				if err != nil {
					return fmt.Errorf("task prompt: %w", err)
				}

				tp := result.(jot.TaskPrompt)
				if tp.Skipped() {
					continue
				}

				task, err := store.CreateTask(note.ID, raw.Description, raw.Line)
				if err != nil {
					return fmt.Errorf("create task %q: %w", raw.Description, err)
				}

				dueStr := tp.DueValue()
				var dueTime *time.Time
				if dueStr != "" {
					due, err := jot.ParseDate(dueStr, time.Now())
					if err == nil {
						dueTime = &due
						_ = store.SetTaskDue(task.ID, dueTime)
					}
				}

				labelNames := tp.LabelValues()
				for _, labelName := range labelNames {
					label, err := store.CreateLabel(labelName)
					if err != nil {
						labels, lerr := store.ListLabels()
						if lerr != nil {
							continue
						}
						for _, l := range labels {
							if l.Name == labelName {
								label = l
								break
							}
						}
						if label == nil {
							continue
						}
					}
					_ = store.AddTaskLabel(task.ID, label.ID)
				}

				resolved := jot.ResolveLine(raw, dueStr, labelNames)
				resolutions[raw.Line] = resolved
			}
		}

		if len(resolutions) > 0 {
			updated := jot.ApplyResolutions(newContent, resolutions)
			if err := os.WriteFile(notePath, []byte(updated), 0o600); err != nil {
				return fmt.Errorf("write resolved note %q: %w", notePath, err)
			}
		}

		fmt.Fprintln(out, cli.Success(out, "note updated: "+cli.Accent(out, note.Slug)))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(editCmd)
}
