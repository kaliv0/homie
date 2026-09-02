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
			before: Config{MinSize: -2, TTL: -7, MaxSize: -100},
			want:   Config{MinSize: DefaultMinSize, TTL: 0, MaxSize: DefaultMaxSize},
		},
		{
			name:   "clamps zero min_size and max_size",
			before: Config{MinSize: 0, MaxSize: 0},
			want:   Config{MinSize: DefaultMinSize, MaxSize: DefaultMaxSize},
		},
		{
			name:   "keeps valid values",
			before: Config{MinSize: 12, TTL: 7, MaxSize: 300},
			want:   Config{MinSize: 12, TTL: 7, MaxSize: 300},
		},
		{
			name:       "no warn when not verbose",
			before:     Config{MinSize: -1, TTL: -1, MaxSize: -1},
			want:       Config{MinSize: DefaultMinSize, TTL: 0, MaxSize: DefaultMaxSize},
			checkWarn:  true,
			wantSilent: true,
		},
		{
			name:       "no warn for zero values when verbose",
			before:     Config{MinSize: 0, MaxSize: 0, TTL: 0},
			want:       Config{MinSize: DefaultMinSize, MaxSize: DefaultMaxSize, TTL: 0},
			verbose:    true,
			checkWarn:  true,
			wantSilent: true,
		},
		{
			name:      "warns when verbose and negative",
			before:    Config{MinSize: -2, TTL: -2, MaxSize: -3},
			want:      Config{MinSize: DefaultMinSize, TTL: 0, MaxSize: DefaultMaxSize},
			verbose:   true,
			checkWarn: true,
		},
		{
			name:      "clears unknown tool with warning",
			before:    Config{Tool: "pbcopy", MinSize: 10, MaxSize: 300},
			want:      Config{Tool: "", MinSize: 10, MaxSize: 300},
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

			if c.MinSize != tt.want.MinSize || c.TTL != tt.want.TTL || c.MaxSize != tt.want.MaxSize || c.Tool != tt.want.Tool {
				t.Fatalf("after normalize: got MinSize=%d TTL=%d MaxSize=%d Tool=%q, want MinSize=%d TTL=%d MaxSize=%d Tool=%q",
					c.MinSize, c.TTL, c.MaxSize, c.Tool, tt.want.MinSize, tt.want.TTL, tt.want.MaxSize, tt.want.Tool)
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
			if tt.want.Tool == "" && tt.before.Tool != "" {
				if !strings.Contains(joined, "tool") {
					t.Errorf("expected warning mentioning tool, got %q", stderr)
				}
				return
			}
			for _, key := range []string{"min_size", "ttl", "max_size"} {
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
		name    string
		yaml    string
		setup   func(*viper.Viper)
		wantErr string
		check   func(t *testing.T, cfg *Config)
	}{
		{
			name: "defaults from empty viper",
			check: func(t *testing.T, cfg *Config) {
				if cfg.MinSize != DefaultMinSize || cfg.MaxSize != DefaultMaxSize || cfg.TTL != 0 {
					t.Fatalf("unexpected defaults: %+v", cfg)
				}
				if cfg.Tool != "" {
					t.Fatalf("Tool = %q, want empty", cfg.Tool)
				}
			},
		},
		{
			name: "valid config",
			yaml: "min_size: 15\nclean_up: true\nttl: 7\ntool: xclip\n",
			check: func(t *testing.T, cfg *Config) {
				if cfg.MinSize != 15 || !cfg.CleanUp || cfg.TTL != 7 || cfg.Tool != ClipboardXclip {
					t.Fatalf("cfg = %+v, unexpected values", cfg)
				}
			},
		},
		{
			name: "ignores legacy limit key",
			yaml: "limit: 99\nmin_size: 10\n",
			check: func(t *testing.T, cfg *Config) {
				if cfg.MinSize != 10 {
					t.Fatalf("MinSize = %d, want 10 (limit key must not apply)", cfg.MinSize)
				}
			},
		},
		{
			name: "legacy limit alone does not set min_size",
			yaml: "limit: 99\n",
			check: func(t *testing.T, cfg *Config) {
				if cfg.MinSize != DefaultMinSize {
					t.Fatalf("MinSize = %d, want default %d", cfg.MinSize, DefaultMinSize)
				}
			},
		},
		{
			name: "clamps negative values",
			yaml: "min_size: -2\nttl: -5\nmax_size: -10\n",
			check: func(t *testing.T, cfg *Config) {
				if cfg.MinSize != DefaultMinSize || cfg.TTL != 0 || cfg.MaxSize != DefaultMaxSize {
					t.Fatalf("cfg = %+v, want clamped defaults", cfg)
				}
			},
		},
		{
			name: "clears unknown tool",
			yaml: "tool: pbcopy\n",
			check: func(t *testing.T, cfg *Config) {
				if cfg.Tool != "" {
					t.Fatalf("Tool = %q, want empty", cfg.Tool)
				}
			},
		},
		{
			name: "accepts xsel",
			yaml: "tool: xsel\n",
			check: func(t *testing.T, cfg *Config) {
				if cfg.Tool != ClipboardXsel {
					t.Fatalf("Tool = %q, want %q", cfg.Tool, ClipboardXsel)
				}
			},
		},
		{
			name: "accepts wl-clipboard",
			yaml: "tool: wl-clipboard\n",
			check: func(t *testing.T, cfg *Config) {
				if cfg.Tool != ClipboardWLClipboard {
					t.Fatalf("Tool = %q, want %q", cfg.Tool, ClipboardWLClipboard)
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
