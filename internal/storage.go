package internal

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/spf13/viper"
)

const (
	DefaultLimit   = 20
	DefaultMaxSize = 500
)

// ClipboardItem represents a clipboard entry persisted in the database.
type ClipboardItem struct {
	ID        uint `gorm:"primaryKey"`
	ClipText  string
	TextHash  string
	TimeStamp time.Time `gorm:"index"`
}

// Repository wraps database access for clipboard items.
type Repository struct {
	db *gorm.DB
}

// NewRepository opens (and optionally migrates) the SQLite database at dbPath.
func NewRepository(dbPath string, shouldMigrate bool) *Repository {
	// create db if not exists
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		Logger.Fatal(err)
	}

	r := &Repository{db}
	if shouldMigrate {
		if err = r.db.AutoMigrate(&ClipboardItem{}); err != nil {
			// best-effort close on migration failure, ignore close errors
			if sqlDB, sqlErr := db.DB(); sqlErr == nil {
				_ = sqlDB.Close()
			}
			Logger.Fatal(err)
		}
	}
	return r
}

// Read returns clipboard items ordered by timestamp descending with pagination.
func (r *Repository) Read(offset, limit int) []ClipboardItem {
	var items []ClipboardItem
	result := r.db.Order("time_stamp desc").
		Offset(offset).
		Limit(limit).
		Find(&items)
	if result.Error != nil {
		r.Close()
		Logger.Fatal(result.Error)
	}
	return items
}

// Write inserts a new clipboard item or bumps timestamp if the text already exists.
func (r *Repository) Write(item []byte) {
	hasher := sha256.New()
	hasher.Write(item)
	textHash := hex.EncodeToString(hasher.Sum(nil))

	var existingItem = ClipboardItem{}
	result := r.db.Where(&ClipboardItem{TextHash: textHash}).First(&existingItem)
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		r.Close()
		Logger.Fatal(result.Error)
	}

	if result.RowsAffected > 0 {
		existingItem.TimeStamp = time.Now()
		result := r.db.Save(&existingItem)
		if result.Error != nil {
			r.Close()
			Logger.Fatal(result.Error)
		}
	} else {
		result := r.db.Create(&ClipboardItem{
			ClipText:  string(item),
			TextHash:  textHash,
			TimeStamp: time.Now(),
		})
		if result.Error != nil {
			r.Close()
			Logger.Fatal(result.Error)
		}
	}
}

// DeleteExcess removes the oldest records by count.
func (r *Repository) DeleteExcess(deleteCount int) {
	result := r.db.Exec(`DELETE FROM clipboard_items WHERE id IN
					  (SELECT id FROM clipboard_items ORDER BY time_stamp LIMIT ?)`, deleteCount)
	if result.Error != nil {
		r.Close()
		Logger.Fatal(result.Error)
	}
}

// DeleteOldest removes records older than the given TTL (days).
func (r *Repository) DeleteOldest(ttl int) {
	result := r.db.Exec(`DELETE FROM clipboard_items
       					WHERE time_stamp < datetime('now', ? || ' days', 'localtime')`, fmt.Sprintf("-%d", ttl))
	if result.Error != nil {
		r.Close()
		Logger.Fatal(result.Error)
	}
}

// Count returns the total number of clipboard records.
func (r *Repository) Count() int {
	var count int64
	result := r.db.Model(&ClipboardItem{}).Count(&count)
	if result.Error != nil {
		r.Close()
		Logger.Fatal(result.Error)
	}
	return int(count)
}

// Reset deletes all clipboard records.
func (r *Repository) Reset() {
	result := r.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&ClipboardItem{})
	if result.Error != nil {
		r.Close()
		Logger.Fatal(result.Error)
	}
}

// Close releases the underlying database connection.
func (r *Repository) Close() {
	// get generic db object sql.DB to use its functions
	sqlDB, err := r.db.DB()
	if err != nil {
		Logger.Print(err)
		return
	}
	if err = sqlDB.Close(); err != nil {
		Logger.Print(err)
	}
}

// CleanOldHistory trims clipboard history based on ttl or max_size settings.
func CleanOldHistory(db *Repository) {
	ReadConfig()
	if shouldClean := viper.GetBool("clean_up"); !shouldClean {
		return
	}

	// ttl takes precedence over 'size limit' strategy
	if ttl := viper.GetInt("ttl"); ttl > 0 {
		db.DeleteOldest(ttl)
		return
	}

	maxSize := viper.GetInt("max_size")
	if maxSize <= 0 {
		maxSize = DefaultMaxSize
	}

	minLimit := viper.GetInt("limit")
	if minLimit <= 0 {
		minLimit = DefaultLimit
	}

	if total := db.Count(); total > maxSize {
		if minLimit >= total {
			return
		}

		if deleteCount := total - minLimit; deleteCount > 0 {
			db.DeleteExcess(deleteCount)
		}
	}
}
