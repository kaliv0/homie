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

func applyHomieDefaults(v *viper.Viper) {
	v.SetDefault(Verbose, false)
	v.SetDefault(LogFile, "")
	v.SetDefault(PIDFile, "")
	v.SetDefault(Limit, DefaultLimit)
	v.SetDefault(TTL, DefaultTTL)
	v.SetDefault(Keep, DefaultKeep)
	v.SetDefault(Threshold, DefaultThreshold)
	v.SetDefault(Tool, "")
}

func viperFromYAML(t *testing.T, yaml string) *viper.Viper {
	t.Helper()
	v := viper.New()
	applyHomieDefaults(v)
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
			name:   "normalizes negative values",
			before: Config{Keep: -2, TTL: -7, Threshold: -100, Limit: -4},
			want:   Config{Keep: DefaultKeep, TTL: 0, Threshold: DefaultThreshold, Limit: DefaultLimit},
		},
		{
			name:   "keeps zero keep and threshold",
			before: Config{Keep: 0, Threshold: 0, Limit: 0},
			want:   Config{Keep: 0, Threshold: 0, Limit: 0},
		},
		{
			name:   "keeps zero limit",
			before: Config{Keep: DefaultKeep, Threshold: DefaultThreshold, Limit: 0},
			want:   Config{Keep: DefaultKeep, Threshold: DefaultThreshold, Limit: 0},
		},
		{
			name:   "keeps valid values",
			before: Config{Keep: 12, TTL: 7, Threshold: 300, Limit: 8},
			want:   Config{Keep: 12, TTL: 7, Threshold: 300, Limit: 8},
		},
		{
			name:       "no warn when not verbose",
			before:     Config{Keep: -1, TTL: -1, Threshold: -1, Limit: -1},
			want:       Config{Keep: DefaultKeep, TTL: 0, Threshold: DefaultThreshold, Limit: DefaultLimit},
			checkWarn:  true,
			wantSilent: true,
		},
		{
			name:       "no warn for zero ttl when verbose",
			before:     Config{Keep: DefaultKeep, Threshold: DefaultThreshold, TTL: 0, Limit: DefaultLimit},
			want:       Config{Keep: DefaultKeep, Threshold: DefaultThreshold, TTL: 0, Limit: DefaultLimit},
			verbose:    true,
			checkWarn:  true,
			wantSilent: true,
		},
		{
			name:      "warns when verbose and negative",
			before:    Config{Keep: -2, TTL: -2, Threshold: -3, Limit: -4},
			want:      Config{Keep: DefaultKeep, TTL: 0, Threshold: DefaultThreshold, Limit: DefaultLimit},
			verbose:   true,
			checkWarn: true,
		},
		{
			name:      "clears unknown tool with warning",
			before:    Config{Tool: "pbcopy", Keep: 10, Threshold: 300, Limit: DefaultLimit},
			want:      Config{Tool: "", Keep: 10, Threshold: 300, Limit: DefaultLimit},
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

			if c.Keep != tt.want.Keep || c.TTL != tt.want.TTL || c.Threshold != tt.want.Threshold || c.Limit != tt.want.Limit || c.Tool != tt.want.Tool {
				t.Fatalf("after normalize: got Keep=%d TTL=%d Threshold=%d Limit=%d Tool=%q, want Keep=%d TTL=%d Threshold=%d Limit=%d Tool=%q",
					c.Keep, c.TTL, c.Threshold, c.Limit, c.Tool, tt.want.Keep, tt.want.TTL, tt.want.Threshold, tt.want.Limit, tt.want.Tool)
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
			for _, key := range []string{"keep", "ttl", "threshold", "limit"} {
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
				if cfg.Keep != DefaultKeep || cfg.Threshold != DefaultThreshold || cfg.TTL != 0 || cfg.Limit != DefaultLimit {
					t.Fatalf("unexpected defaults: %+v", cfg)
				}
				if cfg.Tool != "" {
					t.Fatalf("Tool = %q, want empty", cfg.Tool)
				}
			},
		},
		{
			name: "valid config",
			yaml: "keep: 15\nttl: 7\nlimit: 8\ntool: xclip\n",
			check: func(t *testing.T, cfg *Config) {
				if cfg.Keep != 15 || cfg.TTL != 7 || cfg.Limit != 8 || cfg.Tool != ClipboardXclip {
					t.Fatalf("cfg = %+v, unexpected values", cfg)
				}
			},
		},
		{
			name: "limit does not set keep",
			yaml: "limit: 99\nkeep: 10\n",
			check: func(t *testing.T, cfg *Config) {
				if cfg.Keep != 10 {
					t.Fatalf("Keep = %d, want 10", cfg.Keep)
				}
				if cfg.Limit != 99 {
					t.Fatalf("Limit = %d, want 99", cfg.Limit)
				}
			},
		},
		{
			name: "limit alone does not set keep",
			yaml: "limit: 99\n",
			check: func(t *testing.T, cfg *Config) {
				if cfg.Keep != DefaultKeep {
					t.Fatalf("Keep = %d, want default %d", cfg.Keep, DefaultKeep)
				}
				if cfg.Limit != 99 {
					t.Fatalf("Limit = %d, want 99", cfg.Limit)
				}
			},
		},
		{
			name: "preserves explicit zero limit",
			yaml: "limit: 0\n",
			check: func(t *testing.T, cfg *Config) {
				if cfg.Limit != 0 {
					t.Fatalf("Limit = %d, want 0", cfg.Limit)
				}
			},
		},
		{
			name: "preserves explicit zero keep and threshold",
			yaml: "keep: 0\nthreshold: 0\n",
			check: func(t *testing.T, cfg *Config) {
				if cfg.Keep != 0 || cfg.Threshold != 0 {
					t.Fatalf("cfg = %+v, want Keep=0 Threshold=0", cfg)
				}
			},
		},
		{
			name: "preserves explicit zero keep only",
			yaml: "keep: 0\n",
			check: func(t *testing.T, cfg *Config) {
				if cfg.Keep != 0 {
					t.Fatalf("Keep = %d, want 0", cfg.Keep)
				}
				if cfg.Threshold != DefaultThreshold {
					t.Fatalf("Threshold = %d, want default %d", cfg.Threshold, DefaultThreshold)
				}
			},
		},
		{
			name: "preserves explicit zero threshold only",
			yaml: "threshold: 0\n",
			check: func(t *testing.T, cfg *Config) {
				if cfg.Threshold != 0 {
					t.Fatalf("Threshold = %d, want 0", cfg.Threshold)
				}
				if cfg.Keep != DefaultKeep {
					t.Fatalf("Keep = %d, want default %d", cfg.Keep, DefaultKeep)
				}
			},
		},
		{
			name: "normalizes negative values",
			yaml: "keep: -2\nttl: -5\nthreshold: -10\nlimit: -3\n",
			check: func(t *testing.T, cfg *Config) {
				if cfg.Keep != DefaultKeep || cfg.TTL != 0 || cfg.Threshold != DefaultThreshold || cfg.Limit != DefaultLimit {
					t.Fatalf("cfg = %+v, want normalized defaults", cfg)
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
				v.Set(LogFile, "~/homie.log")
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
				applyHomieDefaults(v)
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
