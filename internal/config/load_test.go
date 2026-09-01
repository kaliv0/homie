package config

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func viperFromYAML(t *testing.T, yaml string) *viper.Viper {
	t.Helper()
	v := viper.New()
	if yaml == "" {
		return v
	}
	v.SetConfigType("yaml")
	if err := v.ReadConfig(bytes.NewBufferString(yaml)); err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}
	return v
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stderr
	os.Stderr = w
	t.Cleanup(func() { os.Stderr = orig })
	fn()
	_ = w.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()
	return buf.String()
}

func TestConfig_normalize(t *testing.T) {
	tests := []struct {
		name       string
		before     Config
		want       Config
		verbose    bool
		checkWarn  bool
		wantSilent bool // when checkWarn, expect no stderr
	}{
		{
			name:   "clamps negative values",
			before: Config{Limit: -1, TTL: -7, MaxSize: -100},
			want:   Config{Limit: DefaultLimit, TTL: 0, MaxSize: DefaultMaxSize},
		},
		{
			name:   "clamps zero limit and max_size",
			before: Config{Limit: 0, MaxSize: 0},
			want:   Config{Limit: DefaultLimit, MaxSize: DefaultMaxSize},
		},
		{
			name:   "keeps valid values",
			before: Config{Limit: 15, TTL: 7, MaxSize: 300},
			want:   Config{Limit: 15, TTL: 7, MaxSize: 300},
		},
		{
			name:       "no warn when not verbose",
			before:     Config{Limit: -1, TTL: -1, MaxSize: -1},
			want:       Config{Limit: DefaultLimit, TTL: 0, MaxSize: DefaultMaxSize},
			checkWarn:  true,
			wantSilent: true,
		},
		{
			name:       "no warn for zero values when verbose",
			before:     Config{Limit: 0, MaxSize: 0, TTL: 0},
			want:       Config{Limit: DefaultLimit, MaxSize: DefaultMaxSize, TTL: 0},
			verbose:    true,
			checkWarn:  true,
			wantSilent: true,
		},
		{
			name:      "warns when verbose and negative",
			before:    Config{Limit: -1, TTL: -2, MaxSize: -3},
			want:      Config{Limit: DefaultLimit, TTL: 0, MaxSize: DefaultMaxSize},
			verbose:   true,
			checkWarn: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.before
			c.Verbose = tt.verbose

			var stderr string
			if tt.checkWarn {
				stderr = captureStderr(t, c.normalize)
			} else {
				c.normalize()
			}

			if c.Limit != tt.want.Limit || c.TTL != tt.want.TTL || c.MaxSize != tt.want.MaxSize {
				t.Fatalf("after normalize: got Limit=%d TTL=%d MaxSize=%d, want Limit=%d TTL=%d MaxSize=%d",
					c.Limit, c.TTL, c.MaxSize, tt.want.Limit, tt.want.TTL, tt.want.MaxSize)
			}

			if !tt.checkWarn {
				return
			}
			if tt.wantSilent {
				if stderr != "" {
					t.Fatalf("expected no warnings, got %q", stderr)
				}
				return
			}
			if stderr == "" {
				t.Fatal("expected warnings, got none")
			}
			joined := strings.ToLower(stderr)
			for _, key := range []string{"limit", "ttl", "max_size"} {
				if !strings.Contains(joined, key) {
					t.Errorf("expected warning mentioning %q, got %q", key, stderr)
				}
			}
		})
	}
}

func TestParse(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	tests := []struct {
		name       string
		yaml       string
		setup      func(*viper.Viper)
		wantErr    string
		check      func(t *testing.T, cfg *Config)
	}{
		{
			name: "defaults from empty viper",
			check: func(t *testing.T, cfg *Config) {
				if cfg.Limit != DefaultLimit || cfg.MaxSize != DefaultMaxSize || cfg.TTL != 0 {
					t.Fatalf("unexpected defaults: %+v", cfg)
				}
				if cfg.ClipboardTool != "" {
					t.Fatalf("ClipboardTool = %q, want empty", cfg.ClipboardTool)
				}
			},
		},
		{
			name: "valid config",
			yaml: "limit: 10\nclean_up: true\nttl: 7\nuse_xclip: true\n",
			check: func(t *testing.T, cfg *Config) {
				if cfg.Limit != 10 || !cfg.CleanUp || cfg.TTL != 7 || cfg.ClipboardTool != ClipboardXclip {
					t.Fatalf("cfg = %+v, unexpected values", cfg)
				}
			},
		},
		{
			name: "clamps negative values",
			yaml: "limit: -1\nttl: -5\nmax_size: -10\n",
			check: func(t *testing.T, cfg *Config) {
				if cfg.Limit != DefaultLimit || cfg.TTL != 0 || cfg.MaxSize != DefaultMaxSize {
					t.Fatalf("cfg = %+v, want clamped defaults", cfg)
				}
			},
		},
		{
			name:    "rejects multiple clipboard tools",
			yaml:    "use_xclip: true\nuse_xsel: true\n",
			wantErr: "clipboard",
		},
		{
			name: "selects xsel",
			yaml: "use_xsel: true\n",
			check: func(t *testing.T, cfg *Config) {
				if cfg.ClipboardTool != ClipboardXsel {
					t.Fatalf("ClipboardTool = %q, want %q", cfg.ClipboardTool, ClipboardXsel)
				}
			},
		},
		{
			name: "selects wl-clipboard",
			yaml: "use_wl-clipboard: true\n",
			check: func(t *testing.T, cfg *Config) {
				if cfg.ClipboardTool != ClipboardWLClipboard {
					t.Fatalf("ClipboardTool = %q, want %q", cfg.ClipboardTool, ClipboardWLClipboard)
				}
			},
		},
		{
			name: "expands log file path",
			setup: func(v *viper.Viper) {
				v.Set(KeyLogFile, "~/homie.log")
			},
			check: func(t *testing.T, cfg *Config) {
				want := filepath.Join(tmpDir, "homie.log")
				if cfg.LogFile != want {
					t.Errorf("LogFile = %q, want %q", cfg.LogFile, want)
				}
			},
		},
		{
			name: "expands pid file path",
			yaml: "pid_file: ~/state/homie.pid\n",
			check: func(t *testing.T, cfg *Config) {
				want := filepath.Join(tmpDir, "state", "homie.pid")
				if cfg.PIDFile != want {
					t.Errorf("PIDFile = %q, want %q", cfg.PIDFile, want)
				}
			},
		},
		{
			name: "default pid file from XDG_RUNTIME_DIR",
			setup: func(v *viper.Viper) {
				t.Setenv("XDG_RUNTIME_DIR", filepath.Join(tmpDir, "runtime"))
			},
			check: func(t *testing.T, cfg *Config) {
				want := filepath.Join(tmpDir, "runtime", pidFileName)
				if cfg.PIDFile != want {
					t.Errorf("PIDFile = %q, want %q", cfg.PIDFile, want)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var v *viper.Viper
			if tt.setup != nil {
				v = viper.New()
				tt.setup(v)
			} else {
				v = viperFromYAML(t, tt.yaml)
			}

			cfg, err := Parse(v)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatal("Parse() expected error, got nil")
				}
				if !strings.Contains(strings.ToLower(err.Error()), tt.wantErr) {
					t.Fatalf("Parse() error = %q, want mention of %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse() unexpected error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, cfg)
			}
		})
	}
}

func TestPreparePIDFile_createsParentDir(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "homie.pid")
	cfg, err := Parse(viperFromYAML(t, "pid_file: "+path+"\n"))
	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	if err := cfg.PreparePIDFile(); err != nil {
		t.Fatalf("PreparePIDFile() failed: %v", err)
	}
	if _, err := os.Stat(filepath.Dir(path)); err != nil {
		t.Fatalf("expected parent directory to be created: %v", err)
	}
}
