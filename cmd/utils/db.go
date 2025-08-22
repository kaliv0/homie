package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type ClipboardItem struct {
	ID        uint `gorm:"primaryKey"`
	ClipText  string
	TextHash  string
	TimeStamp time.Time `gorm:"index"`
}

type Repository struct {
	db *gorm.DB
}

func NewRepository(dbPath string, shouldMigrate bool) *Repository {
	// create db if not exists
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		Logger.Fatal(err)
	}

	r := &Repository{db}
	if shouldMigrate {
		if migrateErr := r.db.AutoMigrate(&ClipboardItem{}); migrateErr != nil {
			Logger.Print(migrateErr)

			sqlDB, sqlErr := db.DB()
			if sqlErr != nil {
				Logger.Print(sqlErr)
			} else if closeErr := sqlDB.Close(); closeErr != nil {
				Logger.Print(closeErr)
			}
			os.Exit(1)
		}
	}
	return r
}

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

func (r *Repository) Write(item []byte) {
	hasher := sha256.New()
	hasher.Write(item)
	textHash := hex.EncodeToString(hasher.Sum(nil))

	var existingItem = ClipboardItem{}
	result := r.db.Where(&ClipboardItem{TextHash: textHash}).First(&existingItem)
	if result.Error != nil && result.Error.Error() != "record not found" {
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
	result := r.db.Exec(`DELETE FROM clipboard_items WHERE id IN 
					  (SELECT id FROM clipboard_items ORDER BY time_stamp ASC LIMIT ?)`, deleteCount)
	if result.Error != nil {
		r.Close()
		Logger.Fatal(result.Error)
	}
}

func (r *Repository) DeleteOldest(ttl int) {
	result := r.db.Exec(`DELETE FROM clipboard_items 
       					WHERE time_stamp < datetime('now', concat(?, ' days'), 'localtime') `, fmt.Sprintf("-%d", ttl))
	if result.Error != nil {
		r.Close()
		Logger.Fatal(result.Error)
	}
}

func (r *Repository) Count() int {
	var count int64
	result := r.db.Model(&ClipboardItem{}).Count(&count)
	if result.Error != nil {
		r.Close()
		Logger.Fatal(result.Error)
	}
	return int(count)
}

func (r *Repository) Reset() {
	result := r.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&ClipboardItem{})
	if result.Error != nil {
		r.Close()
		Logger.Fatal(result.Error)
	}
}

func (r *Repository) Close() {
	// get generic db object sql.DB to use its functions
	sqlDB, err := r.db.DB()
	if err != nil {
		Logger.Fatal(err)
	}
	if err = sqlDB.Close(); err != nil {
		Logger.Fatal(err)
	}
}
