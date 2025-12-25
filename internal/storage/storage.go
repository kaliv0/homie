package storage

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/viper"

	"github.com/kaliv0/homie/internal/config"
	"github.com/kaliv0/homie/internal/log"
)

const (
	MaxDbConnections = 2
	ConnMaxLifetime  = 12 * time.Hour
	DbBusyTimeout    = 5000 // 5s in milliseconds
	JournalMode      = "WAL"
	DbSync           = "NORMAL"

	DefaultLimit   = 20
	DefaultMaxSize = 500
)

// ClipboardItem represents a clipboard entry persisted in the database.
type ClipboardItem struct {
	ID        int       `db:"id"`
	ClipText  string    `db:"clip_text"`
	TextHash  string    `db:"text_hash"`
	TimeStamp time.Time `db:"time_stamp"`
}

// Repository wraps database access for clipboard items.
type Repository struct {
	db *sqlx.DB
}

// NewRepository opens the SQLite database at dbPath.
func NewRepository(dbPath string) (*Repository, error) {
	// create db if not exists
	db, err := sqlx.Connect("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database at %q: %w", dbPath, err)
	}

	// verify connection
	if err = db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping database at %q: %w", dbPath, err)
	}

	// set SQLite connection pool settings suited for a single-file DB
	db.SetMaxOpenConns(MaxDbConnections)
	db.SetMaxIdleConns(MaxDbConnections)
	db.SetConnMaxLifetime(ConnMaxLifetime) // TrackingClipboard is a long-running background task

	// set SQLite pragmas
	pragmas := []string{
		fmt.Sprintf(`PRAGMA busy_timeout = %d`, DbBusyTimeout),
		fmt.Sprintf(`PRAGMA journal_mode = %s`, JournalMode),
		fmt.Sprintf(`PRAGMA synchronous = %s`, DbSync),
	}

	for _, pragma := range pragmas {
		if _, err = db.Exec(pragma); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("failed to set pragma %q: %w", pragma, err)
		}
	}

	return &Repository{db}, nil
}

// AutoMigrate creates the clipboard_items table if it doesn't exist.
func (r *Repository) AutoMigrate() error {
	_, err := r.db.Exec(`
		CREATE TABLE IF NOT EXISTS clipboard_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			clip_text TEXT NOT NULL,
			text_hash TEXT NOT NULL,
			time_stamp DATETIME NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create clipboard_items table: %w", err)
	}
	// Create index on time_stamp for better query performance
	_, err = r.db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_time_stamp ON clipboard_items(time_stamp)
	`)
	if err != nil {
		return fmt.Errorf("failed to create index idx_time_stamp on clipboard_items: %w", err)
	}
	return nil
}

// Read returns clipboard items ordered by timestamp descending.
func (r *Repository) Read(offset, limit int) ([]ClipboardItem, error) {
	var items []ClipboardItem
	err := r.db.Select(&items, `
		SELECT id, clip_text, text_hash, time_stamp 
		FROM clipboard_items 
		ORDER BY time_stamp DESC 
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to read clipboard items (offset=%d, limit=%d): %w", offset, limit, err)
	}
	return items, nil
}

// Write inserts a new clipboard item or updates timestamp if it already exists.
func (r *Repository) Write(item []byte) error {
	hasher := sha256.New()
	if _, err := hasher.Write(item); err != nil {
		return fmt.Errorf("failed to hash clipboard item (length=%d): %w", len(item), err)
	}
	textHash := hex.EncodeToString(hasher.Sum(nil))

	var existingItem ClipboardItem
	err := r.db.Get(&existingItem, `
		SELECT id, clip_text, text_hash, time_stamp 
		FROM clipboard_items 
		WHERE text_hash = ?
	`, textHash)

	if errors.Is(err, sql.ErrNoRows) {
		_, err = r.db.Exec(`
			INSERT INTO clipboard_items (clip_text, text_hash, time_stamp) 
			VALUES (?, ?, ?)
		`, string(item), textHash, time.Now())
		if err != nil {
			return fmt.Errorf("failed to insert clipboard item (hash=%s, length=%d): %w", textHash, len(item), err)
		}
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to check for existing clipboard item (hash=%s): %w", textHash, err)
	}

	_, err = r.db.Exec(`
		UPDATE clipboard_items 
		SET time_stamp = ? 
		WHERE id = ?
	`, time.Now(), existingItem.ID)
	if err != nil {
		return fmt.Errorf("failed to update timestamp for clipboard item (id=%d, hash=%s): %w", existingItem.ID, textHash, err)
	}
	return nil
}

// DeleteExcess removes the oldest records.
func (r *Repository) DeleteExcess(deleteCount int) error {
	_, err := r.db.Exec(`
		DELETE FROM clipboard_items 
		WHERE id IN (
			SELECT id FROM clipboard_items 
			ORDER BY time_stamp 
			LIMIT ?
		)
	`, deleteCount)
	if err != nil {
		return fmt.Errorf("failed to delete excess clipboard items (count=%d): %w", deleteCount, err)
	}
	return nil
}

// DeleteOldest removes records older than the given TTL.
func (r *Repository) DeleteOldest(ttl int) error {
	_, err := r.db.Exec(`
		DELETE FROM clipboard_items
		WHERE time_stamp < datetime('now', concat(?, ' days'), 'localtime')
	`, "-"+strconv.Itoa(ttl))
	if err != nil {
		return fmt.Errorf("failed to delete oldest clipboard items (ttl=%d days): %w", ttl, err)
	}
	return nil
}

// Count returns the total number of records.
func (r *Repository) Count() (int, error) {
	var count int
	err := r.db.Get(&count, `SELECT COUNT(*) FROM clipboard_items`)
	if err != nil {
		return 0, fmt.Errorf("failed to count clipboard items: %w", err)
	}
	return count, nil
}

// Reset deletes all records.
func (r *Repository) Reset() error {
	_, err := r.db.Exec(`DELETE FROM clipboard_items`)
	if err != nil {
		return fmt.Errorf("failed to reset clipboard history: %w", err)
	}
	return nil
}

// Close releases the database connection.
func (r *Repository) Close() error {
	return r.db.Close()
}

// CleanOldHistory trims clipboard history based on ttl or max_size settings.
func CleanOldHistory(db *Repository) error {
	if err := config.ReadConfig(); err != nil {
		log.Logger().Println(err)
	}
	if shouldClean := viper.GetBool("clean_up"); !shouldClean {
		return nil
	}

	// ttl takes precedence over 'size limit' strategy
	if ttl := viper.GetInt("ttl"); ttl > 0 {
		return db.DeleteOldest(ttl)
	}

	maxSize := viper.GetInt("max_size")
	if maxSize <= 0 {
		maxSize = DefaultMaxSize
	}

	minLimit := viper.GetInt("limit")
	if minLimit <= 0 {
		minLimit = DefaultLimit
	}

	total, err := db.Count()
	if err != nil {
		return err
	}

	if total > maxSize {
		if minLimit >= total {
			return nil
		}

		if deleteCount := total - minLimit; deleteCount > 0 {
			if err = db.DeleteExcess(deleteCount); err != nil {
				return err
			}
		}
	}
	return nil
}
