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
func NewRepository(dbPath string, shouldMigrate bool) (*Repository, error) {
	// create db if not exists
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	r := &Repository{db}
	if shouldMigrate {
		if migrateErr := r.db.AutoMigrate(&ClipboardItem{}); migrateErr != nil {
			sqlDB, sqlErr := db.DB()
			if sqlErr != nil {
				return nil, fmt.Errorf("auto migrate: %w", migrateErr)
			}
			if dbCloseErr := sqlDB.Close(); dbCloseErr != nil {
				return nil, fmt.Errorf("close database: %w", dbCloseErr)
			}
		}
	}
	return r, nil
}

func (r *Repository) Read(offset, limit int) []ClipboardItem {
	// Read returns clipboard items ordered by timestamp descending with pagination.
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

func (r *Repository) Write(item []byte) {
	// Write inserts a new clipboard item or bumps timestamp if the text already exists.
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

func (r *Repository) DeleteExcess(deleteCount int) {
	// DeleteExcess removes the oldest records by count.
	result := r.db.Exec(`DELETE FROM clipboard_items WHERE id IN
					  (SELECT id FROM clipboard_items ORDER BY time_stamp ASC LIMIT ?)`, deleteCount)
	if result.Error != nil {
		r.Close()
		Logger.Fatal(result.Error)
	}
}

func (r *Repository) DeleteOldest(ttl int) {
	// DeleteOldest removes records older than the given TTL (days).
	result := r.db.Exec(`DELETE FROM clipboard_items
       					WHERE time_stamp < datetime('now', ? || ' days', 'localtime')`, fmt.Sprintf("-%d", ttl))
	if result.Error != nil {
		r.Close()
		Logger.Fatal(result.Error)
	}
}

func (r *Repository) Count() int {
	// Count returns the total number of clipboard records.
	var count int64
	result := r.db.Model(&ClipboardItem{}).Count(&count)
	if result.Error != nil {
		r.Close()
		Logger.Fatal(result.Error)
	}
	return int(count)
}

func (r *Repository) Reset() {
	// Reset deletes all clipboard records.
	result := r.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&ClipboardItem{})
	if result.Error != nil {
		r.Close()
		Logger.Fatal(result.Error)
	}
}

func (r *Repository) Close() {
	// Close releases the underlying database connection.
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
	if maxSize == 0 {
		maxSize = 500
	}
	minLimit := viper.GetInt("limit")
	total := db.Count()
	if total > maxSize {
		if minLimit == 0 {
			minLimit = 30
		}
		db.DeleteExcess(total - minLimit)
	}
}
