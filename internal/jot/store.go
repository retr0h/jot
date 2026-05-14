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

// Package jot holds the core logic — SQLite store, @task parser, editor integration, and kvlt-backed encryption.
package jot

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite" // SQLite driver — pure Go, no CGo.
)

const (
	dateFmt = "2006-01-02"
)

// Note is a single markdown note persisted in the store.
type Note struct {
	ID        int64
	Slug      string
	Title     string
	Secure    bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Task is a @task item linked to a Note.
type Task struct {
	ID          int64
	NoteID      int64
	Description string
	Status      string // "open" or "done"
	DueDate     *time.Time
	DoneAt      *time.Time
	LineNumber  int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Label is a tag that can be attached to notes or tasks.
type Label struct {
	ID   int64
	Name string
}

// SearchResult is returned by SearchNotes for each matching note.
type SearchResult struct {
	NoteID int64
	Slug   string
	Title  string
}

// Store wraps a SQLite database with WAL mode and foreign keys enabled.
type Store struct {
	db *sql.DB
}

// OpenStore opens (or creates) the SQLite database at dbPath, applies
// all migrations, and returns a ready-to-use Store.
func OpenStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %q: %w", dbPath, err)
	}

	// SQLite performs best with a single writer connection when WAL is in use.
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable WAL: %w", err)
	}
	if _, err := db.Exec(`PRAGMA foreign_keys=ON`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

// Close releases the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// migrate creates all tables and virtual tables when they do not yet exist.
func (s *Store) migrate() error {
	const schema = `
CREATE TABLE IF NOT EXISTS notes (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    slug       TEXT UNIQUE NOT NULL,
    title      TEXT NOT NULL,
    secure     INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS tasks (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    note_id     INTEGER REFERENCES notes(id) ON DELETE CASCADE,
    description TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'open',
    due_date    TEXT,
    done_at     TEXT,
    line_number INTEGER,
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS labels (
    id   INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL
);
CREATE TABLE IF NOT EXISTS note_labels (
    note_id  INTEGER REFERENCES notes(id) ON DELETE CASCADE,
    label_id INTEGER REFERENCES labels(id) ON DELETE CASCADE,
    PRIMARY KEY (note_id, label_id)
);
CREATE TABLE IF NOT EXISTS task_labels (
    task_id  INTEGER REFERENCES tasks(id) ON DELETE CASCADE,
    label_id INTEGER REFERENCES labels(id) ON DELETE CASCADE,
    PRIMARY KEY (task_id, label_id)
);
CREATE VIRTUAL TABLE IF NOT EXISTS notes_fts USING fts5(
    title, body, content='', contentless_delete=1
);
`
	_, err := s.db.Exec(schema)
	return err
}

// ─── Notes ────────────────────────────────────────────────────────────────────

// CreateNote inserts a new note and returns the persisted record.
func (s *Store) CreateNote(
	slug, title string,
	secure bool,
) (*Note, error) {
	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339)
	secureInt := 0
	if secure {
		secureInt = 1
	}

	res, err := s.db.Exec(
		`INSERT INTO notes (slug, title, secure, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?)`,
		slug, title, secureInt, nowStr, nowStr,
	)
	if err != nil {
		return nil, fmt.Errorf("create note: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("create note last id: %w", err)
	}
	return &Note{
		ID:        id,
		Slug:      slug,
		Title:     title,
		Secure:    secure,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// GetNote retrieves a note by primary key.
func (s *Store) GetNote(id int64) (*Note, error) {
	row := s.db.QueryRow(
		`SELECT id, slug, title, secure, created_at, updated_at
		 FROM notes WHERE id = ?`,
		id,
	)
	return scanNote(row)
}

// GetNoteBySlug retrieves a note by its unique slug.
func (s *Store) GetNoteBySlug(slug string) (*Note, error) {
	row := s.db.QueryRow(
		`SELECT id, slug, title, secure, created_at, updated_at
		 FROM notes WHERE slug = ?`,
		slug,
	)
	return scanNote(row)
}

// ListNotes returns all notes, optionally filtered to those carrying a given
// label name. Pass an empty string to return all notes.
func (s *Store) ListNotes(labelFilter string) ([]*Note, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if labelFilter == "" {
		rows, err = s.db.Query(
			`SELECT id, slug, title, secure, created_at, updated_at
			 FROM notes ORDER BY created_at DESC`,
		)
	} else {
		rows, err = s.db.Query(
			`SELECT n.id, n.slug, n.title, n.secure, n.created_at, n.updated_at
			 FROM notes n
			 JOIN note_labels nl ON nl.note_id = n.id
			 JOIN labels l       ON l.id = nl.label_id
			 WHERE l.name = ?
			 ORDER BY n.created_at DESC`,
			labelFilter,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("list notes: %w", err)
	}
	defer rows.Close()

	var notes []*Note
	for rows.Next() {
		n, err := scanNoteRow(rows)
		if err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

// DeleteNote removes a note by id. Cascades to tasks and note_labels. Also
// removes the note's FTS entry when one exists.
func (s *Store) DeleteNote(id int64) error {
	// Remove from FTS before deleting the row so we still have the rowid.
	if _, err := s.db.Exec(
		`DELETE FROM notes_fts WHERE rowid = ?`, id,
	); err != nil {
		return fmt.Errorf("delete note fts: %w", err)
	}
	if _, err := s.db.Exec(`DELETE FROM notes WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete note: %w", err)
	}
	return nil
}

// ─── Tasks ────────────────────────────────────────────────────────────────────

// CreateTask inserts a new open task linked to noteID and returns the record.
func (s *Store) CreateTask(
	noteID int64,
	description string,
	lineNumber int,
) (*Task, error) {
	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339)

	res, err := s.db.Exec(
		`INSERT INTO tasks (note_id, description, status, line_number, created_at, updated_at)
		 VALUES (?, ?, 'open', ?, ?, ?)`,
		noteID, description, lineNumber, nowStr, nowStr,
	)
	if err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("create task last id: %w", err)
	}
	return &Task{
		ID:          id,
		NoteID:      noteID,
		Description: description,
		Status:      "open",
		LineNumber:  lineNumber,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// GetTask retrieves a task by primary key.
func (s *Store) GetTask(id int64) (*Task, error) {
	row := s.db.QueryRow(
		`SELECT id, note_id, description, status, due_date, done_at,
		        line_number, created_at, updated_at
		 FROM tasks WHERE id = ?`,
		id,
	)
	return scanTask(row)
}

// CompleteTask marks a task as done and records the completion time.
func (s *Store) CompleteTask(id int64) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(
		`UPDATE tasks SET status='done', done_at=?, updated_at=? WHERE id=?`,
		now, now, id,
	)
	if err != nil {
		return fmt.Errorf("complete task: %w", err)
	}
	return nil
}

// ReopenTask sets a task back to open and clears done_at.
func (s *Store) ReopenTask(id int64) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(
		`UPDATE tasks SET status='open', done_at=NULL, updated_at=? WHERE id=?`,
		now, id,
	)
	if err != nil {
		return fmt.Errorf("reopen task: %w", err)
	}
	return nil
}

// SetTaskDue updates the due_date for a task. Pass nil to clear the due date.
func (s *Store) SetTaskDue(
	id int64,
	due *time.Time,
) error {
	var dueStr sql.NullString
	if due != nil {
		dueStr = sql.NullString{String: due.Format(dateFmt), Valid: true}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(
		`UPDATE tasks SET due_date=?, updated_at=? WHERE id=?`,
		dueStr, now, id,
	)
	if err != nil {
		return fmt.Errorf("set task due: %w", err)
	}
	return nil
}

// TasksDue returns open tasks whose due_date falls within [from, to] inclusive.
func (s *Store) TasksDue(from, to time.Time) ([]*Task, error) {
	rows, err := s.db.Query(
		`SELECT id, note_id, description, status, due_date, done_at,
		        line_number, created_at, updated_at
		 FROM tasks
		 WHERE status='open'
		   AND due_date IS NOT NULL
		   AND due_date >= ?
		   AND due_date <= ?
		 ORDER BY due_date ASC`,
		from.Format(dateFmt),
		to.Format(dateFmt),
	)
	if err != nil {
		return nil, fmt.Errorf("tasks due: %w", err)
	}
	defer rows.Close()
	return collectTasks(rows)
}

// ListTasks returns tasks optionally filtered by status ("open", "done", or
// "all") and by label name. Pass an empty labelFilter to skip label filtering.
func (s *Store) ListTasks(
	status, labelFilter string,
) ([]*Task, error) {
	base := `SELECT t.id, t.note_id, t.description, t.status, t.due_date,
	                t.done_at, t.line_number, t.created_at, t.updated_at
	         FROM tasks t`

	args := []any{}

	if labelFilter != "" {
		base += ` JOIN task_labels tl ON tl.task_id = t.id
		          JOIN labels l       ON l.id = tl.label_id`
	}

	where := " WHERE 1=1"
	if status != "" && status != "all" {
		where += " AND t.status=?"
		args = append(args, status)
	}
	if labelFilter != "" {
		where += " AND l.name=?"
		args = append(args, labelFilter)
	}

	query := base + where + " ORDER BY t.created_at DESC"
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()
	return collectTasks(rows)
}

// ─── Labels ───────────────────────────────────────────────────────────────────

// CreateLabel inserts a new label and returns the persisted record.
func (s *Store) CreateLabel(name string) (*Label, error) {
	res, err := s.db.Exec(`INSERT INTO labels (name) VALUES (?)`, name)
	if err != nil {
		return nil, fmt.Errorf("create label: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("create label last id: %w", err)
	}
	return &Label{ID: id, Name: name}, nil
}

// ListLabels returns all labels ordered by name.
func (s *Store) ListLabels() ([]*Label, error) {
	rows, err := s.db.Query(`SELECT id, name FROM labels ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list labels: %w", err)
	}
	defer rows.Close()

	var labels []*Label
	for rows.Next() {
		l := &Label{}
		if err := rows.Scan(&l.ID, &l.Name); err != nil {
			return nil, fmt.Errorf("scan label: %w", err)
		}
		labels = append(labels, l)
	}
	return labels, rows.Err()
}

// DeleteLabel removes a label by id. Cascades to note_labels and task_labels.
func (s *Store) DeleteLabel(id int64) error {
	if _, err := s.db.Exec(`DELETE FROM labels WHERE id=?`, id); err != nil {
		return fmt.Errorf("delete label: %w", err)
	}
	return nil
}

// AddNoteLabel attaches a label to a note (idempotent — INSERT OR IGNORE).
func (s *Store) AddNoteLabel(
	noteID, labelID int64,
) error {
	if _, err := s.db.Exec(
		`INSERT OR IGNORE INTO note_labels (note_id, label_id) VALUES (?, ?)`,
		noteID, labelID,
	); err != nil {
		return fmt.Errorf("add note label: %w", err)
	}
	return nil
}

// AddTaskLabel attaches a label to a task (idempotent — INSERT OR IGNORE).
func (s *Store) AddTaskLabel(
	taskID, labelID int64,
) error {
	if _, err := s.db.Exec(
		`INSERT OR IGNORE INTO task_labels (task_id, label_id) VALUES (?, ?)`,
		taskID, labelID,
	); err != nil {
		return fmt.Errorf("add task label: %w", err)
	}
	return nil
}

// ─── FTS5 ─────────────────────────────────────────────────────────────────────

// IndexNote inserts or replaces the full-text index entry for a note.
func (s *Store) IndexNote(
	noteID int64,
	title, body string,
) error {
	// Contentless FTS5 tables require an explicit rowid on insert.
	if _, err := s.db.Exec(
		`INSERT OR REPLACE INTO notes_fts (rowid, title, body) VALUES (?, ?, ?)`,
		noteID, title, body,
	); err != nil {
		return fmt.Errorf("index note: %w", err)
	}
	return nil
}

// SearchNotes performs an FTS5 full-text search and returns matching notes.
func (s *Store) SearchNotes(query string) ([]*SearchResult, error) {
	rows, err := s.db.Query(
		`SELECT f.rowid, n.slug, n.title
		 FROM notes_fts f
		 JOIN notes n ON n.id = f.rowid
		 WHERE notes_fts MATCH ?
		 ORDER BY rank`,
		query,
	)
	if err != nil {
		return nil, fmt.Errorf("search notes: %w", err)
	}
	defer rows.Close()

	var results []*SearchResult
	for rows.Next() {
		r := &SearchResult{}
		if err := rows.Scan(&r.NoteID, &r.Slug, &r.Title); err != nil {
			return nil, fmt.Errorf("scan search result: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// ─── internal scan helpers ────────────────────────────────────────────────────

type noteScanner interface {
	Scan(dest ...any) error
}

func scanNote(row noteScanner) (*Note, error) {
	var (
		n          Note
		secureInt  int
		createdStr string
		updatedStr string
	)
	if err := row.Scan(
		&n.ID, &n.Slug, &n.Title, &secureInt, &createdStr, &updatedStr,
	); err != nil {
		return nil, fmt.Errorf("scan note: %w", err)
	}
	n.Secure = secureInt != 0

	var err error
	if n.CreatedAt, err = time.Parse(time.RFC3339, createdStr); err != nil {
		return nil, fmt.Errorf("parse note created_at: %w", err)
	}
	if n.UpdatedAt, err = time.Parse(time.RFC3339, updatedStr); err != nil {
		return nil, fmt.Errorf("parse note updated_at: %w", err)
	}
	return &n, nil
}

func scanNoteRow(rows *sql.Rows) (*Note, error) {
	return scanNote(rows)
}

func scanTask(row noteScanner) (*Task, error) {
	var (
		t          Task
		dueDateStr sql.NullString
		doneAtStr  sql.NullString
		createdStr string
		updatedStr string
	)
	if err := row.Scan(
		&t.ID, &t.NoteID, &t.Description, &t.Status,
		&dueDateStr, &doneAtStr,
		&t.LineNumber, &createdStr, &updatedStr,
	); err != nil {
		return nil, fmt.Errorf("scan task: %w", err)
	}

	if dueDateStr.Valid {
		parsed, err := time.Parse(dateFmt, dueDateStr.String)
		if err != nil {
			return nil, fmt.Errorf("parse task due_date: %w", err)
		}
		t.DueDate = &parsed
	}
	if doneAtStr.Valid {
		parsed, err := time.Parse(time.RFC3339, doneAtStr.String)
		if err != nil {
			return nil, fmt.Errorf("parse task done_at: %w", err)
		}
		t.DoneAt = &parsed
	}

	var err error
	if t.CreatedAt, err = time.Parse(time.RFC3339, createdStr); err != nil {
		return nil, fmt.Errorf("parse task created_at: %w", err)
	}
	if t.UpdatedAt, err = time.Parse(time.RFC3339, updatedStr); err != nil {
		return nil, fmt.Errorf("parse task updated_at: %w", err)
	}
	return &t, nil
}

func collectTasks(rows *sql.Rows) ([]*Task, error) {
	var tasks []*Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}
