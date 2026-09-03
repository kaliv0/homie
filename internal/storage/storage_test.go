package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
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

// mustRead calls repo.Read and fails the test on error.
func mustRead(t *testing.T, repo *Repository, offset, limit int) []ClipboardItem {
	t.Helper()
	items, err := repo.Read(offset, limit)
	if err != nil {
		t.Fatalf("Read(%d, %d) failed: %v", offset, limit, err)
	}
	return items
}

func assertClipTexts(t *testing.T, items []ClipboardItem, want []string) {
	t.Helper()
	if len(items) != len(want) {
		t.Fatalf("expected %d items, got %d", len(want), len(items))
	}
	for i, w := range want {
		if items[i].ClipText != w {
			t.Errorf("item[%d]: want %q, got %q", i, w, items[i].ClipText)
		}
	}
}

func TestNewRepository(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	repo, err := NewRepository(dbPath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	t.Cleanup(func() { _ = repo.Close() })

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

func TestSetDBFilesPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "perm.db")

	repo, err := NewRepository(dbPath)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}
	t.Cleanup(func() { _ = repo.Close() })

	if err := repo.AutoMigrate(); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	if err := repo.SetDBFilesPermissions(); err != nil {
		t.Fatalf("SetDBFilesPermissions: %v", err)
	}

	for _, path := range []string{dbPath, dbPath + "-wal", dbPath + "-shm"} {
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			t.Fatal(err)
		}
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Errorf("%q: expected mode 0600, got %04o", path, perm)
		}
	}
}

func TestWrite(t *testing.T) {
	tests := []struct {
		name  string
		texts []string
		want  []string
	}{
		{"insert", []string{"hello world"}, []string{"hello world"}},
		{"empty", []string{""}, []string{""}},
		{"large", []string{strings.Repeat("a", 10000)}, []string{strings.Repeat("a", 10000)}},
		{"special characters", []string{
			"hello\nworld",
			"tab\there",
			"quote'test",
			`double"quote`,
			"emoji 🎉",
			"null\x00byte",
			"unicode: こんにちは",
		}, nil}, // count-only via want nil meaning assert all texts
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := setupTestDB(t)
			for _, text := range tt.texts {
				if err := repo.Write([]byte(text)); err != nil {
					t.Fatalf("Write(%q) failed: %v", text, err)
				}
			}
			want := tt.want
			if want == nil {
				want = tt.texts
			}
			// Read is newest-first
			reversed := make([]string, len(want))
			for i, w := range want {
				reversed[len(want)-1-i] = w
			}
			assertClipTexts(t, mustRead(t, repo, 0, len(want)+1), reversed)
		})
	}
}

func TestWrite_Deduplication(t *testing.T) {
	repo := setupTestDB(t)

	if err := repo.Write([]byte("same")); err != nil {
		t.Fatalf("first Write() failed: %v", err)
	}
	items1 := mustRead(t, repo, 0, 10)
	ts1 := items1[0].TimeStamp

	if err := repo.Write([]byte("same")); err != nil {
		t.Fatalf("second Write() failed: %v", err)
	}
	items2 := mustRead(t, repo, 0, 10)
	if len(items2) != 1 {
		t.Fatalf("expected 1 item after dedup, got %d", len(items2))
	}
	if !items2[0].TimeStamp.After(ts1) {
		t.Errorf("expected updated timestamp to be after original: %v vs %v", items2[0].TimeStamp, ts1)
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

func TestRead_Ordering(t *testing.T) {
	repo := setupTestDB(t)
	seedItems(t, repo, 3)

	assertClipTexts(t, mustRead(t, repo, 0, 10), []string{"item-2", "item-1", "item-0"})
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

			items := mustRead(t, repo, tt.offset, tt.limit)
			if len(items) != tt.wantLen {
				t.Errorf("expected %d items, got %d", tt.wantLen, len(items))
			}
		})
	}
}

func TestDeleteExcess(t *testing.T) {
	tests := []struct {
		name        string
		numItems    int
		deleteCount int
		wantCount   int
		wantTexts   []string // newest-first, when set
	}{
		{"delete some", 5, 2, 3, []string{"item-4", "item-3", "item-2"}},
		{"delete zero", 5, 0, 5, nil},
		{"delete more than total", 3, 10, 0, nil},
		{"delete exact total", 5, 5, 0, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := setupTestDB(t)
			seedItems(t, repo, tt.numItems)

			if err := repo.DeleteExcess(tt.deleteCount); err != nil {
				t.Fatalf("DeleteExcess(%d) failed: %v", tt.deleteCount, err)
			}
			assertCount(t, repo, tt.wantCount)
			if tt.wantTexts != nil {
				assertClipTexts(t, mustRead(t, repo, 0, 10), tt.wantTexts)
			}
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

	if err := repo.Write([]byte("before-reset")); err != nil {
		t.Fatalf("Write() failed: %v", err)
	}
	if err := repo.Reset(); err != nil {
		t.Fatalf("Reset() failed: %v", err)
	}
	assertCount(t, repo, 0)

	if err := repo.Write([]byte("after-reset")); err != nil {
		t.Fatalf("Write() after reset failed: %v", err)
	}
	assertClipTexts(t, mustRead(t, repo, 0, 10), []string{"after-reset"})
}

func TestCleanOldHistory_TTL(t *testing.T) {
	t.Parallel()
	repo := setupTestDB(t)
	seedItems(t, repo, 10)
	insertOldItem(t, repo, "old-0", "oldhash-prec0", 20)
	insertOldItem(t, repo, "old-1", "oldhash-prec1", 20)
	// TTL=7 removes only the 2 old items. threshold=5 would trim more but is ignored.
	cfg := CleanupConfig{TTL: 7, Threshold: 5, Keep: 5}
	if err := CleanOldHistory(repo, cfg); err != nil {
		t.Fatalf("CleanOldHistory() failed: %v", err)
	}
	assertCount(t, repo, 10)
}

func TestCleanOldHistory_Threshold(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		numItems  int
		threshold int
		keep      int
		wantCount int
		wantTexts []string // when set, verify surviving rows (newest-first)
	}{
		{"trims to keep", 10, 5, 5, 5, []string{"item-9", "item-8", "item-7", "item-6", "item-5"}},
		{"under threshold no-op", 5, 10, 3, 5, nil},
		{"keep reaches total no-op", 5, 3, 5, 5, nil},
		{"keep zero wipes when over threshold", 10, 5, 0, 0, nil},
		{"threshold zero trims to keep", 10, 0, 3, 3, []string{"item-9", "item-8", "item-7"}},
		{"threshold zero keep zero wipes all", 5, 0, 0, 0, nil},
		{"empty db", 0, 0, 0, 0, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := setupTestDB(t)
			seedItems(t, repo, tt.numItems)

			cfg := CleanupConfig{TTL: 0, Threshold: tt.threshold, Keep: tt.keep}
			if err := CleanOldHistory(repo, cfg); err != nil {
				t.Fatalf("CleanOldHistory() failed: %v", err)
			}
			assertCount(t, repo, tt.wantCount)
			if tt.wantTexts != nil {
				assertClipTexts(t, mustRead(t, repo, 0, 10), tt.wantTexts)
			}
		})
	}
}
