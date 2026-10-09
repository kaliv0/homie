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

func TestDBPath(t *testing.T) {
	tests := []struct {
		name string
		// xdg and want take the temp dir as root
		xdg  func(root string) string
		want func(root string) string
	}{
		{
			name: "with XDG",
			xdg:  func(root string) string { return root },
			want: func(root string) string { return filepath.Join(root, dbSubdirName, dbFileName) },
		},
		{
			name: "XDG with nested path",
			xdg:  func(root string) string { return filepath.Join(root, "deep", "nested", "config") },
			want: func(root string) string {
				return filepath.Join(root, "deep", "nested", "config", dbSubdirName, dbFileName)
			},
		},
		{
			name: "without XDG falls back to HOME",
			xdg:  func(string) string { return "" },
			want: func(root string) string { return filepath.Join(root, dbConfDirName, dbSubdirName, dbFileName) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			t.Setenv("HOME", root)
			resetDBPath(t, tt.xdg(root))

			path := mustDBPath(t)
			if want := tt.want(root); path != want {
				t.Errorf("expected path=%q, got %q", want, path)
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
