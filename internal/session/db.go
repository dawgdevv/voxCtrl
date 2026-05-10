package session

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

type Entry struct {
	Timestamp    time.Time
	Intent       string
	Confidence   float64
	Executed     string
	Result       string
	ActiveWindow string
	DurationMs   int64
	ContextJson  string
	store        *Store
	start        time.Time
}

func NewStore() (*Store, error) {
	// Implementation for creating a new store

	dir := filepath.Join(os.Getenv("HOME"), ".local", "share", "voxctrl")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	dbPath := filepath.Join(dir, "sessions.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migrate db: %w", err)
	}

	log.Printf("[Session] DB at %s", dbPath)
	return &Store{db: db}, nil

}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS sessions(
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	timestamp DATETIME NOT NULL,
	intent TEXT NOT NULL,
	confidence REAL NOT NULL,
	executed TEXT NOT NULL,
	result TEXT NOT NULL,
	active_window TEXT,
	duration_ms INTEGER NOT NULL,
	context_json TEXT
)`)
	if err != nil {
		return fmt.Errorf("migrate db: %w", err)
	}
	return nil
}

func (s *Store) NewEntry(intent string) *Entry {
	return &Entry{store: s, Intent: intent, start: time.Now()}
}

func (e *Entry) Finish(err error) {
	e.DurationMs = time.Since(e.start).Milliseconds()
	if err != nil {
		e.Result = "error: " + err.Error()

	} else {
		e.Result = "success"
	}

}

func (s *Store) Save(e *Entry) error {
	_, err := s.db.Exec(`INSERT INTO sessions (timestamp, intent, confidence, executed, result, active_window, duration_ms, context_json) VALUES(?,?,?,?,?,?,?,?)`,
		time.Now().UTC(), e.Intent, e.Confidence, e.Executed, e.Result, e.ActiveWindow, e.DurationMs, e.ContextJson,
	)
	return err
}

func (s *Store) Close() {
	_ = s.db.Close()
}
