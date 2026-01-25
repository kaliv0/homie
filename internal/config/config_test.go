package config

import (
	"os"
	"path/filepath"
	"strings"
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
}

func TestDBPath_WithoutXDG(t *testing.T) {
	resetDBPath(t, "")

	path := mustDBPath(t)

	homeDir, _ := os.UserHomeDir()
	expected := filepath.Join(homeDir, dbConfDirName, dbSubdirName, dbFileName)
	if path != expected {
		t.Errorf("expected path=%q, got %q", expected, path)
	}
}

func TestDBPath_Properties(t *testing.T) {
	tests := []struct {
		name  string
		check func(t *testing.T, path string)
	}{
		{"creates directory with correct permissions", func(t *testing.T, path string) {
			info, err := os.Stat(filepath.Dir(path))
			if err != nil {
				t.Fatalf("directory not created: %v", err)
			}
			if info.Mode().Perm() != dbConfDirPerm {
				t.Errorf("expected permissions %o, got %o", dbConfDirPerm, info.Mode().Perm())
			}
		}},
		{"correct filename", func(t *testing.T, path string) {
			if filepath.Base(path) != dbFileName {
				t.Errorf("expected filename=%q, got %q", dbFileName, filepath.Base(path))
			}
		}},
		{"correct subdir", func(t *testing.T, path string) {
			if filepath.Base(filepath.Dir(path)) != dbSubdirName {
				t.Errorf("expected subdir=%q, got %q", dbSubdirName, filepath.Base(filepath.Dir(path)))
			}
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetDBPath(t, t.TempDir())
			path := mustDBPath(t)
			tt.check(t, path)
		})
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

func TestDBPath_WithoutXDG_UsesHomeConfig(t *testing.T) {
	tmpDir := t.TempDir()
	resetDBPath(t, "")
	t.Setenv("HOME", tmpDir)

	path := mustDBPath(t)

	if !strings.Contains(path, dbConfDirName) {
		t.Errorf("expected path to contain %q when XDG not set, got %q", dbConfDirName, path)
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

func TestReadConfig_NoFile(t *testing.T) {
	ReadConfig = func() error {
		return readConfig()
	}

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	err := ReadConfig()
	if err != nil {
		if !strings.Contains(err.Error(), "failed to read config file") {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}

func TestReadConfig_WithValidFile(t *testing.T) {
	ReadConfig = func() error {
		return readConfig()
	}

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configContent := []byte("limit: 50\nclean_up: true\nttl: 14\n")
	if err := os.WriteFile(filepath.Join(tmpDir, confFileName), configContent, 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	if err := ReadConfig(); err != nil {
		t.Fatalf("ReadConfig() with valid file failed: %v", err)
	}
}

func TestReadConfig_WithInvalidYAML(t *testing.T) {
	ReadConfig = func() error {
		return readConfig()
	}

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configContent := []byte("invalid: yaml: content: [unbalanced")
	if err := os.WriteFile(filepath.Join(tmpDir, confFileName), configContent, 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Should not panic regardless of parse result
	_ = ReadConfig()
}
