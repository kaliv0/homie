package log

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/kaliv0/homie/internal/config"
)

func TestConfigureVerbose(t *testing.T) {
	t.Cleanup(func() { Configure(0, "") })

	Configure(2, "")
	if Verbose() != 2 {
		t.Fatalf("Verbose() = %d, want 2", Verbose())
	}

	Configure(-1, "")
	if Verbose() != 0 {
		t.Fatalf("negative level should clamp to 0, got %d", Verbose())
	}
}

func TestInfofDebugfNoPanic(t *testing.T) {
	t.Cleanup(func() { Configure(0, "") })
	Configure(0, "")
	Infof("ignored at level 0\n")
	Debugf("ignored at level 0\n")
}

func TestConfigureLogFileTee(t *testing.T) {
	t.Cleanup(func() { Configure(0, "") })

	path := filepath.Join(t.TempDir(), "homie.log")
	Configure(1, path)
	Infof("info-line\n")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(data); got != "homie: info-line\n" {
		t.Fatalf("log file contents = %q, want homie: prefix + message", got)
	}

	if runtime.GOOS != "windows" {
		fi, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := fi.Mode().Perm() & 0o777; got != 0o600 {
			t.Fatalf("log file mode = %#o, want 0600", got)
		}
	}
}

func TestConfigureSameLogPathReused(t *testing.T) {
	t.Cleanup(func() { Configure(0, "") })

	path := filepath.Join(t.TempDir(), "homie.log")
	Configure(1, path)
	Infof("first\n")
	Configure(2, path)
	Debugf("second\n")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !strings.Contains(got, "first") || !strings.Contains(got, "second") {
		t.Fatalf("log file = %q, want both first and second", got)
	}
}

func TestConfigureFromFlags_UsesConfigWhenFlagNotSet(t *testing.T) {
	t.Cleanup(func() { Configure(0, "") })
	viper.Set(config.ViperKeyVerbose, 1)
	viper.Set(config.ViperKeyLogFile, "")

	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().CountP("verbose", "v", "verbosity")
	cmd.Flags().String("log-file", "", "log file")

	ConfigureFromFlags(cmd.Flags())
	if got := Verbose(); got != 1 {
		t.Fatalf("Verbose() = %d, want 1", got)
	}
}

func TestConfigureFromFlags_FlagOverridesConfig(t *testing.T) {
	t.Cleanup(func() { Configure(0, "") })
	viper.Set(config.ViperKeyVerbose, 0)
	viper.Set(config.ViperKeyLogFile, "")

	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().CountP("verbose", "v", "verbosity")
	cmd.Flags().String("log-file", "", "log file")
	if err := cmd.ParseFlags([]string{"-vv"}); err != nil {
		t.Fatal(err)
	}

	ConfigureFromFlags(cmd.Flags())
	if got := Verbose(); got != 2 {
		t.Fatalf("Verbose() = %d, want 2", got)
	}
}

func TestConfigureFromFlags_ExpandsHomeInConfigLogFile(t *testing.T) {
	t.Cleanup(func() { Configure(0, "") })
	viper.Set(config.ViperKeyVerbose, 1)
	viper.Set(config.ViperKeyLogFile, "~/homie-configure-from-command.log")

	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().CountP("verbose", "v", "verbosity")
	cmd.Flags().String("log-file", "", "log file")

	ConfigureFromFlags(cmd.Flags())
	Infof("hello\n")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(homeDir, "homie-configure-from-command.log")
	t.Cleanup(func() { _ = os.Remove(path) })

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected expanded log file at %q: %v", path, err)
	}
}
