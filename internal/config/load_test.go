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

func assertNormalized(t *testing.T, got, want Config) {
	t.Helper()
	if got.Keep != want.Keep || got.TTL != want.TTL || got.Threshold != want.Threshold || got.Limit != want.Limit || got.Tool != want.Tool {
		t.Fatalf("got Keep=%d TTL=%d Threshold=%d Limit=%d Tool=%q, want Keep=%d TTL=%d Threshold=%d Limit=%d Tool=%q",
			got.Keep, got.TTL, got.Threshold, got.Limit, got.Tool,
			want.Keep, want.TTL, want.Threshold, want.Limit, want.Tool)
	}
}

func TestConfig_normalize(t *testing.T) {
	tests := []struct {
		name   string
		before Config
		want   Config
	}{
		{
			name:   "normalizes negative values",
			before: Config{Keep: -2, TTL: -7, Threshold: -100, Limit: -4},
			want:   Config{Keep: DefaultKeep, TTL: 0, Threshold: DefaultThreshold, Limit: DefaultLimit},
		},
		{
			name:   "keeps zero keep threshold and limit",
			before: Config{Keep: 0, Threshold: 0, Limit: 0},
			want:   Config{Keep: 0, Threshold: 0, Limit: 0},
		},
		{
			name:   "keeps valid values",
			before: Config{Keep: 12, TTL: 7, Threshold: 300, Limit: 8, Tool: ClipboardXclip},
			want:   Config{Keep: 12, TTL: 7, Threshold: 300, Limit: 8, Tool: ClipboardXclip},
		},
		{
			name:   "clears unknown tool",
			before: Config{Tool: "pbcopy", Keep: 10, Threshold: 300, Limit: DefaultLimit},
			want:   Config{Tool: "", Keep: 10, Threshold: 300, Limit: DefaultLimit},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.before
			c.normalize()
			assertNormalized(t, c, tt.want)
		})
	}
}

func TestConfig_normalize_warnings(t *testing.T) {
	tests := []struct {
		name      string
		before    Config
		verbose   bool
		wantEmpty bool
		wantSub   []string
	}{
		{
			name:      "silent when not verbose",
			before:    Config{Keep: -1, TTL: -1, Threshold: -1, Limit: -1},
			wantEmpty: true,
		},
		{
			name:      "silent for zero ttl when verbose",
			before:    Config{Keep: DefaultKeep, Threshold: DefaultThreshold, TTL: 0, Limit: DefaultLimit},
			verbose:   true,
			wantEmpty: true,
		},
		{
			name:    "warns when verbose and negative",
			before:  Config{Keep: -2, TTL: -2, Threshold: -3, Limit: -4},
			verbose: true,
			wantSub: []string{"keep", "ttl", "threshold", "limit"},
		},
		{
			name:    "warns for unknown tool",
			before:  Config{Tool: "pbcopy", Keep: 10, Threshold: 300, Limit: DefaultLimit},
			wantSub: []string{"tool"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.before
			c.Verbose = tt.verbose
			stderr := captureStderr(t, c.normalize)

			if tt.wantEmpty {
				if stderr != "" {
					t.Fatalf("expected no warnings, got %q", stderr)
				}
				return
			}
			if stderr == "" {
				t.Fatal("expected warnings, got none")
			}
			joined := strings.ToLower(stderr)
			for _, key := range tt.wantSub {
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
		name string
		yaml string
		want Config
	}{
		{
			name: "defaults from empty viper",
			want: Config{Keep: DefaultKeep, Threshold: DefaultThreshold, TTL: 0, Limit: DefaultLimit},
		},
		{
			name: "valid config",
			yaml: "keep: 15\nttl: 7\nlimit: 8\ntool: xclip\n",
			want: Config{Keep: 15, TTL: 7, Limit: 8, Tool: ClipboardXclip, Threshold: DefaultThreshold},
		},
		{
			name: "limit does not set keep",
			yaml: "limit: 99\nkeep: 10\n",
			want: Config{Keep: 10, Limit: 99, Threshold: DefaultThreshold},
		},
		{
			name: "limit alone does not set keep",
			yaml: "limit: 99\n",
			want: Config{Keep: DefaultKeep, Limit: 99, Threshold: DefaultThreshold},
		},
		{
			name: "preserves explicit zeros",
			yaml: "keep: 0\nthreshold: 0\nlimit: 0\n",
			want: Config{Keep: 0, Threshold: 0, Limit: 0},
		},
		{
			name: "preserves zero keep only",
			yaml: "keep: 0\n",
			want: Config{Keep: 0, Threshold: DefaultThreshold, Limit: DefaultLimit},
		},
		{
			name: "preserves zero threshold only",
			yaml: "threshold: 0\n",
			want: Config{Keep: DefaultKeep, Threshold: 0, Limit: DefaultLimit},
		},
		{
			name: "accepts xsel",
			yaml: "tool: xsel\n",
			want: Config{Tool: ClipboardXsel, Keep: DefaultKeep, Threshold: DefaultThreshold, Limit: DefaultLimit},
		},
		{
			name: "accepts wl-clipboard",
			yaml: "tool: wl-clipboard\n",
			want: Config{Tool: ClipboardWLClipboard, Keep: DefaultKeep, Threshold: DefaultThreshold, Limit: DefaultLimit},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Parse(viperFromYAML(t, tt.yaml))
			assertNormalized(t, *cfg, tt.want)
		})
	}

	t.Run("expands log file path", func(t *testing.T) {
		v := viper.New()
		applyHomieDefaults(v)
		v.Set(LogFile, "~/homie.log")
		cfg := Parse(v)
		want := filepath.Join(tmpDir, "homie.log")
		if cfg.LogFile != want {
			t.Errorf("LogFile = %q, want %q", cfg.LogFile, want)
		}
	})

	t.Run("expands pid file path", func(t *testing.T) {
		cfg := Parse(viperFromYAML(t, "pid_file: ~/state/homie.pid\n"))
		want := filepath.Join(tmpDir, "state", "homie.pid")
		if cfg.PIDFile != want {
			t.Errorf("PIDFile = %q, want %q", cfg.PIDFile, want)
		}
	})

	t.Run("default pid file from XDG_RUNTIME_DIR", func(t *testing.T) {
		t.Setenv("XDG_RUNTIME_DIR", filepath.Join(tmpDir, "runtime"))
		cfg := Parse(viperFromYAML(t, ""))
		want := filepath.Join(tmpDir, "runtime", pidFileName)
		if cfg.PIDFile != want {
			t.Errorf("PIDFile = %q, want %q", cfg.PIDFile, want)
		}
	})
}

func TestPreparePIDFile_createsParentDir(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "homie.pid")
	cfg := Parse(viperFromYAML(t, "pid_file: "+path+"\n"))

	if err := cfg.PreparePIDFile(); err != nil {
		t.Fatalf("PreparePIDFile() failed: %v", err)
	}
	if _, err := os.Stat(filepath.Dir(path)); err != nil {
		t.Fatalf("expected parent directory to be created: %v", err)
	}
}
