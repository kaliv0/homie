package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/spf13/viper"

	"github.com/kaliv0/homie/internal/config"
	"github.com/kaliv0/homie/internal/runtime"
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

// NewRepository opens the SQLite database at dbPath.
func NewRepository(dbPath string, shouldMigrate bool) (*Repository, error) {
	// create db if not exists
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	r := &Repository{db}
	if shouldMigrate {
		if err = r.db.AutoMigrate(&ClipboardItem{}); err != nil {
			if sqlDB, sqlErr := db.DB(); sqlErr == nil {
				_ = sqlDB.Close()
			}
			return nil, err
		}
	}
	return r, nil
}

// Read returns clipboard items ordered by timestamp descending.
func (r *Repository) Read(offset, limit int) ([]ClipboardItem, error) {
	var items []ClipboardItem
	result := r.db.Order("time_stamp desc").
		Offset(offset).
		Limit(limit).
		Find(&items)
	if result.Error != nil {
		return nil, result.Error
	}
	return items, nil
}

// Write inserts a new clipboard item or updates timestamp if it already exists.
func (r *Repository) Write(item []byte) error {
	hasher := sha256.New()
	if _, err := hasher.Write(item); err != nil {
		return err
	}
	textHash := hex.EncodeToString(hasher.Sum(nil))

	var existingItem = ClipboardItem{}
	result := r.db.Where(&ClipboardItem{TextHash: textHash}).First(&existingItem)
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return result.Error
	}

	if result.RowsAffected > 0 {
		existingItem.TimeStamp = time.Now()
		result = r.db.Save(&existingItem)
		if result.Error != nil {
			return result.Error
		}
	} else {
		result = r.db.Create(&ClipboardItem{
			ClipText:  string(item),
			TextHash:  textHash,
			TimeStamp: time.Now(),
		})
		if result.Error != nil {
			return result.Error
		}
	}
	return nil
}

// DeleteExcess removes the oldest records.
func (r *Repository) DeleteExcess(deleteCount int) error {
	result := r.db.Exec(`DELETE FROM clipboard_items WHERE id IN
					  (SELECT id FROM clipboard_items ORDER BY time_stamp LIMIT ?)`, deleteCount)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// DeleteOldest removes records older than the given TTL.
func (r *Repository) DeleteOldest(ttl int) error {
	result := r.db.Exec(`DELETE FROM clipboard_items
       					WHERE time_stamp < datetime('now', ? || ' days', 'localtime')`, fmt.Sprintf("-%d", ttl))
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// Count returns the total number of records.
func (r *Repository) Count() (int, error) {
	var count int64
	result := r.db.Model(&ClipboardItem{}).Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}
	return int(count), nil
}

// Reset deletes all records.
func (r *Repository) Reset() error {
	result := r.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&ClipboardItem{})
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// Close releases the database connection.
func (r *Repository) Close() error {
	// get generic db object sql.DB to use its functions
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// CleanOldHistory trims clipboard history based on ttl or max_size settings.
func CleanOldHistory(db *Repository) error {
	if err := config.ReadConfig(); err != nil {
		runtime.Logger().Println(err)
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
			return db.DeleteExcess(deleteCount)
		}
	}
	return nil
}
