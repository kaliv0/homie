package config

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// resetDBPath resets the sync.Once state and sets the XDG env var for DBPath tests.
func resetDBPath(t *testing.T, xdg string) {
	t.Helper()
	once = sync.Once{}
	dbPath = ""
	pathErr = nil
	t.Setenv(xdgConf, xdg)
}

// mustDBPath calls DBPath and fails the test on error.
func mustDBPath(t *testing.T) string {
	t.Helper()
	path, err := DBPath()
	if err != nil {
		t.Fatalf("DBPath() failed: %v", err)
	}
	return path
}

func TestDBPath_WithXDG(t *testing.T) {
	tmpDir := t.TempDir()
	resetDBPath(t, tmpDir)

	path := mustDBPath(t)

	expected := filepath.Join(tmpDir, dbSubdirName, dbFileName)
	if path != expected {
		t.Errorf("expected path=%q, got %q", expected, path)
	}

	dir := filepath.Dir(path)
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("expected directory %q to be created: %v", dir, err)
	}
	if !info.IsDir() {
		t.Errorf("expected %q to be a directory", dir)
	}
	if info.Mode().Perm() != dbConfDirPerm {
		t.Errorf("expected permissions %o, got %o", dbConfDirPerm, info.Mode().Perm())
	}
}

func TestDBPath_WithoutXDG(t *testing.T) {
	tmpDir := t.TempDir()
	resetDBPath(t, "")
	t.Setenv("HOME", tmpDir)

	path := mustDBPath(t)

	expected := filepath.Join(tmpDir, dbConfDirName, dbSubdirName, dbFileName)
	if path != expected {
		t.Errorf("expected path=%q, got %q", expected, path)
	}
}

func TestDBPath_Idempotent(t *testing.T) {
	resetDBPath(t, t.TempDir())

	path1 := mustDBPath(t)
	path2 := mustDBPath(t)

	if path1 != path2 {
		t.Errorf("DBPath() not idempotent: %q != %q", path1, path2)
	}
}

func TestDBPath_XDGWithNestedPath(t *testing.T) {
	tmpDir := t.TempDir()
	nestedDir := filepath.Join(tmpDir, "deep", "nested", "config")
	resetDBPath(t, nestedDir)

	path := mustDBPath(t)

	expected := filepath.Join(nestedDir, dbSubdirName, dbFileName)
	if path != expected {
		t.Errorf("expected path=%q, got %q", expected, path)
	}
	if _, err := os.Stat(filepath.Dir(path)); err != nil {
		t.Fatalf("nested directory not created: %v", err)
	}
}
