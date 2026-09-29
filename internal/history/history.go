package history

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// Entry is one finished transfer.
type Entry struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Kind      string    `json:"kind"` // "upload" or "download"
	Size      int64     `json:"size"`
	Millis    int64     `json:"duration_ms"`
	SHA256    string    `json:"sha256,omitempty"`
	Status    string    `json:"status"` // "completed" or "cancelled"
	CreatedAt time.Time `json:"finished_at"`
}

// Retention caps stored history age.
const Retention = 90 * 24 * time.Hour

// DB wraps the SQLite history store.
type DB struct {
	db *sql.DB
}

// DefaultPath returns ~/.local/share/lanbox/transfers.db.
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "lanbox", "transfers.db"), nil
}

// Open creates the store and schema (single table, WAL mode).
func Open(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS transfers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		kind TEXT NOT NULL,
		size INTEGER NOT NULL DEFAULT 0,
		duration_ms INTEGER NOT NULL DEFAULT 0,
		sha256 TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'completed',
		finished_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_transfers_finished ON transfers(finished_at DESC)`); err != nil {
		db.Close()
		return nil, err
	}
	return &DB{db: db}, nil
}

// Close releases the store. History persists on disk.
func (d *DB) Close() error {
	return d.db.Close()
}

// Record appends one finished transfer and prunes past retention.
func (d *DB) Record(e Entry) error {
	if _, err := d.db.Exec(
		`INSERT INTO transfers (name, kind, size, duration_ms, sha256, status, finished_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		e.Name, e.Kind, e.Size, e.Millis, e.SHA256, e.Status, e.CreatedAt.UTC().Format(time.RFC3339),
	); err != nil {
		return fmt.Errorf("history record: %w", err)
	}
	_, err := d.db.Exec(`DELETE FROM transfers WHERE finished_at < ?`,
		time.Now().Add(-Retention).UTC().Format(time.RFC3339))
	return err
}

// List returns newest-first entries with limit/offset paging.
func (d *DB) List(limit, offset int) ([]Entry, error) {
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := d.db.Query(
		`SELECT id, name, kind, size, duration_ms, sha256, status, finished_at
		 FROM transfers ORDER BY id DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Entry
	for rows.Next() {
		var e Entry
		var ts string
		if err := rows.Scan(&e.ID, &e.Name, &e.Kind, &e.Size, &e.Millis, &e.SHA256, &e.Status, &ts); err != nil {
			return nil, err
		}
		e.CreatedAt, _ = time.Parse(time.RFC3339, ts)
		out = append(out, e)
	}
	if out == nil {
		out = []Entry{}
	}
	return out, rows.Err()
}
