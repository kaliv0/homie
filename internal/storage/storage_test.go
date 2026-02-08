package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func setupTestDB(t *testing.T) *Repository {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	repo, err := NewRepository(dbPath)
	if err != nil {
		t.Fatalf("NewRepository(%q) failed: %v", dbPath, err)
	}
	if err := repo.AutoMigrate(); err != nil {
		t.Fatalf("AutoMigrate() failed: %v", err)
	}
	t.Cleanup(func() { _ = repo.Close() })
	return repo
}

// seedItems inserts n unique items with explicit timestamps spaced 1s apart.
func seedItems(t *testing.T, repo *Repository, n int) {
	t.Helper()
	base := time.Now().Add(-time.Duration(n-1) * time.Second)
	for i := range n {
		text := fmt.Sprintf("item-%d", i)
		ts := base.Add(time.Duration(i) * time.Second)
		_, err := repo.db.Exec(
			`INSERT INTO clipboard_items (clip_text, text_hash, time_stamp) VALUES (?, ?, ?)`,
			text, fmt.Sprintf("hash-%d", i), ts)
		if err != nil {
			t.Fatalf("seedItems(%d) failed: %v", i, err)
		}
	}
}

// insertOldItem inserts a clipboard item with a timestamp daysAgo in the past.
func insertOldItem(t *testing.T, repo *Repository, text, hash string, daysAgo int) {
	t.Helper()
	ts := time.Now().Add(-time.Duration(daysAgo) * 24 * time.Hour)
	_, err := repo.db.Exec(`
		INSERT INTO clipboard_items (clip_text, text_hash, time_stamp)
		VALUES (?, ?, ?)
	`, text, hash, ts)
	if err != nil {
		t.Fatalf("insertOldItem(%q) failed: %v", text, err)
	}
}

// assertCount asserts the total count of items in the repo.
func assertCount(t *testing.T, repo *Repository, expected int) {
	t.Helper()
	count, err := repo.Count()
	if err != nil {
		t.Fatalf("Count() failed: %v", err)
	}
	if count != expected {
		t.Errorf("expected count=%d, got %d", expected, count)
	}
}

// setCleanupConfig sets viper keys for CleanOldHistory and registers cleanup.
func setCleanupConfig(t *testing.T, cleanUp bool, ttl, maxSize, limit int) {
	t.Helper()
	viper.Set("clean_up", cleanUp)
	viper.Set("ttl", ttl)
	viper.Set("max_size", maxSize)
	viper.Set("limit", limit)
	t.Cleanup(viper.Reset)
}

func TestNewRepository(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	repo, err := NewRepository(dbPath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer func() {
		_ = repo.Close()
	}()

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatalf("expected database file to be created at %q", dbPath)
	}
}

func TestNewRepository_InvalidPath(t *testing.T) {
	_, err := NewRepository("/nonexistent/path/to/db.sqlite")
	if err == nil {
		t.Fatal("expected error for invalid path, got nil")
	}
}

func TestAutoMigrate(t *testing.T) {
	repo := setupTestDB(t)

	var tableName string
	err := repo.db.Get(&tableName, `SELECT name FROM sqlite_master WHERE type='table' AND name='clipboard_items'`)
	if err != nil {
		t.Fatalf("expected clipboard_items table to exist: %v", err)
	}

	var indexName string
	err = repo.db.Get(&indexName, `SELECT name FROM sqlite_master WHERE type='index' AND name='idx_time_stamp'`)
	if err != nil {
		t.Fatalf("expected idx_time_stamp index to exist: %v", err)
	}
}

func TestWrite_Insert(t *testing.T) {
	repo := setupTestDB(t)

	if err := repo.Write([]byte("hello world")); err != nil {
		t.Fatalf("Write() failed: %v", err)
	}

	items, err := repo.Read(0, 10)
	if err != nil {
		t.Fatalf("Read() failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].ClipText != "hello world" {
		t.Errorf("expected clip_text=%q, got %q", "hello world", items[0].ClipText)
	}
}

func TestWrite_Deduplication(t *testing.T) {
	repo := setupTestDB(t)

	if err := repo.Write([]byte("duplicate")); err != nil {
		t.Fatalf("first Write() failed: %v", err)
	}
	if err := repo.Write([]byte("duplicate")); err != nil {
		t.Fatalf("second Write() failed: %v", err)
	}

	assertCount(t, repo, 1)
}

func TestWrite_EmptyItem(t *testing.T) {
	repo := setupTestDB(t)

	if err := repo.Write([]byte("")); err != nil {
		t.Fatalf("Write(empty) failed: %v", err)
	}

	items, err := repo.Read(0, 10)
	if err != nil {
		t.Fatalf("Read() failed: %v", err)
	}
	if len(items) != 1 || items[0].ClipText != "" {
		t.Errorf("expected 1 empty item, got %v", items)
	}
}

func TestWrite_LargeItem(t *testing.T) {
	repo := setupTestDB(t)

	largeText := strings.Repeat("a", 10000)
	if err := repo.Write([]byte(largeText)); err != nil {
		t.Fatalf("Write(large) failed: %v", err)
	}

	items, err := repo.Read(0, 10)
	if err != nil {
		t.Fatalf("Read() failed: %v", err)
	}
	if len(items) != 1 || len(items[0].ClipText) != 10000 {
		t.Errorf("expected 1 item of 10000 chars, got %d items", len(items))
	}
}

func TestWrite_DuplicateUpdatesTimestamp(t *testing.T) {
	repo := setupTestDB(t)

	if err := repo.Write([]byte("same")); err != nil {
		t.Fatalf("first Write() failed: %v", err)
	}
	items1, _ := repo.Read(0, 10)
	ts1 := items1[0].TimeStamp

	if err := repo.Write([]byte("same")); err != nil {
		t.Fatalf("second Write() failed: %v", err)
	}
	items2, _ := repo.Read(0, 10)
	ts2 := items2[0].TimeStamp

	if !ts2.After(ts1) {
		t.Errorf("expected updated timestamp to be after original: %v vs %v", ts2, ts1)
	}
}

func TestWrite_MultipleUniqueItems(t *testing.T) {
	repo := setupTestDB(t)
	for i := range 20 {
		if err := repo.Write(fmt.Appendf(nil, "item-%d", i)); err != nil {
			t.Fatalf("Write(item-%d) failed: %v", i, err)
		}
	}
	assertCount(t, repo, 20)
}

func TestWrite_SpecialCharacters(t *testing.T) {
	repo := setupTestDB(t)

	specials := []string{
		"hello\nworld",
		"tab\there",
		"quote'test",
		`double"quote`,
		"emoji 🎉",
		"null\x00byte",
		"unicode: こんにちは",
	}

	for _, text := range specials {
		if err := repo.Write([]byte(text)); err != nil {
			t.Fatalf("Write(%q) failed: %v", text, err)
		}
	}
	assertCount(t, repo, len(specials))
}

func TestRead_Ordering(t *testing.T) {
	repo := setupTestDB(t)
	seedItems(t, repo, 3)

	items, err := repo.Read(0, 10)
	if err != nil {
		t.Fatalf("Read() failed: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	if items[0].ClipText != "item-2" {
		t.Errorf("expected first item=%q, got %q", "item-2", items[0].ClipText)
	}
	if items[2].ClipText != "item-0" {
		t.Errorf("expected last item=%q, got %q", "item-0", items[2].ClipText)
	}
}

func TestRead_Limits(t *testing.T) {
	tests := []struct {
		name     string
		numItems int
		offset   int
		limit    int
		wantLen  int
	}{
		{"zero limit", 3, 0, 0, 0},
		{"limit one", 5, 0, 1, 1},
		{"limit larger than count", 3, 0, 100, 3},
		{"offset beyond count", 3, 10, 5, 0},
		{"offset partial", 5, 3, 10, 2},
		{"empty table", 0, 0, 10, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := setupTestDB(t)
			seedItems(t, repo, tt.numItems)

			items, err := repo.Read(tt.offset, tt.limit)
			if err != nil {
				t.Fatalf("Read(%d, %d) failed: %v", tt.offset, tt.limit, err)
			}
			if len(items) != tt.wantLen {
				t.Errorf("expected %d items, got %d", tt.wantLen, len(items))
			}
		})
	}
}

func TestRead_Pagination(t *testing.T) {
	repo := setupTestDB(t)
	seedItems(t, repo, 5)

	page1, err := repo.Read(0, 2)
	if err != nil {
		t.Fatalf("Read(0, 2) failed: %v", err)
	}
	page2, err := repo.Read(2, 2)
	if err != nil {
		t.Fatalf("Read(2, 2) failed: %v", err)
	}

	if len(page1) != 2 || len(page2) != 2 {
		t.Fatalf("expected 2 items per page, got %d and %d", len(page1), len(page2))
	}
	if page1[0].ClipText == page2[0].ClipText {
		t.Error("page1 and page2 returned the same items")
	}
}

func TestRead_PaginationConsistency(t *testing.T) {
	repo := setupTestDB(t)
	seedItems(t, repo, 10)

	var all []ClipboardItem
	for offset := 0; offset < 10; offset += 3 {
		page, err := repo.Read(offset, 3)
		if err != nil {
			t.Fatalf("Read(%d, 3) failed: %v", offset, err)
		}
		all = append(all, page...)
	}

	if len(all) != 10 {
		t.Fatalf("expected 10 items across all pages, got %d", len(all))
	}

	seen := make(map[int]bool)
	for _, item := range all {
		if seen[item.ID] {
			t.Errorf("duplicate item id=%d found across pages", item.ID)
		}
		seen[item.ID] = true
	}
}

func TestCount(t *testing.T) {
	repo := setupTestDB(t)
	seedItems(t, repo, 3)
	assertCount(t, repo, 3)
}

func TestCount_Empty(t *testing.T) {
	repo := setupTestDB(t)
	assertCount(t, repo, 0)
}

func TestDeleteExcess(t *testing.T) {
	tests := []struct {
		name        string
		numItems    int
		deleteCount int
		wantCount   int
	}{
		{"delete some", 5, 2, 3},
		{"delete zero", 5, 0, 5},
		{"delete more than total", 3, 10, 0},
		{"delete exact total", 5, 5, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := setupTestDB(t)
			seedItems(t, repo, tt.numItems)

			if err := repo.DeleteExcess(tt.deleteCount); err != nil {
				t.Fatalf("DeleteExcess(%d) failed: %v", tt.deleteCount, err)
			}
			assertCount(t, repo, tt.wantCount)
		})
	}
}

func TestDeleteOldest(t *testing.T) {
	tests := []struct {
		name      string
		oldDays   int
		oldCount  int
		newCount  int
		ttl       int
		wantCount int
	}{
		{"removes old items", 10, 1, 1, 7, 1},
		{"keeps recent with ttl=0", 0, 0, 1, 0, 1},
		{"removes all old", 30, 3, 0, 7, 0},
		{"mixed ages", 20, 2, 3, 7, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := setupTestDB(t)

			for i := range tt.oldCount {
				insertOldItem(t, repo, fmt.Sprintf("old-%d", i), fmt.Sprintf("hash-%s-%d", tt.name, i), tt.oldDays)
			}
			seedItems(t, repo, tt.newCount)

			if err := repo.DeleteOldest(tt.ttl); err != nil {
				t.Fatalf("DeleteOldest(%d) failed: %v", tt.ttl, err)
			}
			assertCount(t, repo, tt.wantCount)
		})
	}
}

func TestReset(t *testing.T) {
	repo := setupTestDB(t)
	seedItems(t, repo, 3)

	if err := repo.Reset(); err != nil {
		t.Fatalf("Reset() failed: %v", err)
	}
	assertCount(t, repo, 0)
}

func TestReset_AlreadyEmpty(t *testing.T) {
	repo := setupTestDB(t)

	if err := repo.Reset(); err != nil {
		t.Fatalf("Reset() on empty table failed: %v", err)
	}
	assertCount(t, repo, 0)
}

func TestReset_ThenWrite(t *testing.T) {
	repo := setupTestDB(t)

	if err := repo.Write([]byte("before-reset")); err != nil {
		t.Fatalf("Write() failed: %v", err)
	}
	if err := repo.Reset(); err != nil {
		t.Fatalf("Reset() failed: %v", err)
	}
	if err := repo.Write([]byte("after-reset")); err != nil {
		t.Fatalf("Write() after reset failed: %v", err)
	}

	items, err := repo.Read(0, 10)
	if err != nil {
		t.Fatalf("Read() failed: %v", err)
	}
	if len(items) != 1 || items[0].ClipText != "after-reset" {
		t.Errorf("expected single item %q, got %v", "after-reset", items)
	}
}

func TestCleanOldHistory_Disabled(t *testing.T) {
	repo := setupTestDB(t)
	seedItems(t, repo, 3)
	setCleanupConfig(t, false, 0, 0, 0)

	if err := CleanOldHistory(repo); err != nil {
		t.Fatalf("CleanOldHistory() failed: %v", err)
	}
	assertCount(t, repo, 3)
}

func TestCleanOldHistory_TTL(t *testing.T) {
	repo := setupTestDB(t)
	insertOldItem(t, repo, "old", "ttlhash1", 10)
	if err := repo.Write([]byte("recent")); err != nil {
		t.Fatalf("Write() failed: %v", err)
	}
	setCleanupConfig(t, true, 7, 0, 0)

	if err := CleanOldHistory(repo); err != nil {
		t.Fatalf("CleanOldHistory() failed: %v", err)
	}
	assertCount(t, repo, 1)
}

func TestCleanOldHistory_TTLTakesPrecedenceOverMaxSize(t *testing.T) {
	repo := setupTestDB(t)
	seedItems(t, repo, 10)
	insertOldItem(t, repo, "old-0", "oldhash-prec0", 20)
	insertOldItem(t, repo, "old-1", "oldhash-prec1", 20)
	// TTL=7 removes only the 2 old items; max_size=5 would trim more but is ignored
	setCleanupConfig(t, true, 7, 5, 5)

	if err := CleanOldHistory(repo); err != nil {
		t.Fatalf("CleanOldHistory() failed: %v", err)
	}
	assertCount(t, repo, 10)
}

func TestCleanOldHistory_MaxSize(t *testing.T) {
	tests := []struct {
		name      string
		numItems  int
		maxSize   int
		limit     int
		wantCount int
	}{
		{"trims to limit", 10, 5, 5, 5},
		{"under limit no-op", 3, 10, 5, 3},
		{"limit equals total", 10, 5, 10, 10},
		{"max_size one item", 5, 1, 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := setupTestDB(t)
			seedItems(t, repo, tt.numItems)
			setCleanupConfig(t, true, 0, tt.maxSize, tt.limit)

			if err := CleanOldHistory(repo); err != nil {
				t.Fatalf("CleanOldHistory() failed: %v", err)
			}
			assertCount(t, repo, tt.wantCount)
		})
	}
}

func TestCleanOldHistory_DefaultsWhenZeroOrNegative(t *testing.T) {
	tests := []struct {
		name    string
		maxSize int
		limit   int
	}{
		{"zero values", 0, 0},
		{"negative values", -1, -5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := setupTestDB(t)
			seedItems(t, repo, 3)
			setCleanupConfig(t, true, 0, tt.maxSize, tt.limit)

			if err := CleanOldHistory(repo); err != nil {
				t.Fatalf("CleanOldHistory() failed: %v", err)
			}
			// 3 items is well under DefaultMaxSize (500), so no deletion
			assertCount(t, repo, 3)
		})
	}
}
