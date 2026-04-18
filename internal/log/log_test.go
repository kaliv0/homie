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
	t.Cleanup(func() { Configure(false, "") })

	Configure(true, "")
	if !Verbose() {
		t.Fatal("Verbose() = false, want true")
	}

	Configure(false, "")
	if Verbose() {
		t.Fatal("Verbose() = true, want false")
	}
}

func TestLoggerNonNil(t *testing.T) {
	t.Cleanup(func() { Configure(false, "") })
	Configure(false, "")
	if Logger() == nil {
		t.Fatal("Logger() == nil")
	}
}

func TestConfigureLogFile(t *testing.T) {
	t.Cleanup(func() { Configure(false, "") })

	path := filepath.Join(t.TempDir(), "homie.log")
	Configure(false, path)
	Logger().Printf("info-line\n")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !strings.Contains(got, "D'OH: ") || !strings.Contains(got, "info-line") {
		t.Fatalf("log file contents = %q, want homie: prefix, file:line, and message", got)
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
	t.Cleanup(func() { Configure(false, "") })

	path := filepath.Join(t.TempDir(), "homie.log")
	Configure(false, path)
	Logger().Printf("first\n")
	Configure(true, path)
	Logger().Printf("second\n")

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
	t.Cleanup(func() { Configure(false, "") })
	viper.Set(config.ViperKeyVerbose, true)
	viper.Set(config.ViperKeyLogFile, "")

	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().BoolP("verbose", "v", false, "verbosity")
	cmd.Flags().String("log-file", "", "log file")

	ConfigureFromFlags(cmd.Flags())
	if !Verbose() {
		t.Fatal("Verbose() = false, want true")
	}
}

func TestConfigureFromFlags_FlagOverridesConfig(t *testing.T) {
	t.Cleanup(func() { Configure(false, "") })
	viper.Set(config.ViperKeyVerbose, false)
	viper.Set(config.ViperKeyLogFile, "")

	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().BoolP("verbose", "v", false, "verbosity")
	cmd.Flags().String("log-file", "", "log file")
	if err := cmd.ParseFlags([]string{"-v"}); err != nil {
		t.Fatal(err)
	}

	ConfigureFromFlags(cmd.Flags())
	if !Verbose() {
		t.Fatal("Verbose() = false, want true")
	}
}

func TestConfigureFromFlags_ExpandsHomeInConfigLogFile(t *testing.T) {
	t.Cleanup(func() { Configure(false, "") })
	viper.Set(config.ViperKeyVerbose, true)
	viper.Set(config.ViperKeyLogFile, "~/homie-configure-from-command.log")

	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().BoolP("verbose", "v", false, "verbosity")
	cmd.Flags().String("log-file", "", "log file")

	ConfigureFromFlags(cmd.Flags())
	Logger().Printf("hello\n")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(homeDir, "homie-configure-from-command.log")
	t.Cleanup(func() { _ = os.Remove(path) })

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected expanded log file at %q: %v", path, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "hello") {
		t.Fatalf("log file = %q, want hello", string(data))
	}
}

func TestConfigureFromFlags_ConfigVerboseAndLogFile_Tees(t *testing.T) {
	t.Cleanup(func() { Configure(false, "") })
	viper.Set(config.ViperKeyVerbose, true)
	path := filepath.Join(t.TempDir(), "homie.log")
	viper.Set(config.ViperKeyLogFile, path)

	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().BoolP("verbose", "v", false, "verbosity")
	cmd.Flags().String("log-file", "", "log file")

	ConfigureFromFlags(cmd.Flags())
	Logger().Printf("only-file\n")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "only-file") {
		t.Fatalf("expected log line in file, got %q", string(data))
	}
}

func TestConfigureFromFlags_TeeWhenBothFlagsExplicit(t *testing.T) {
	t.Cleanup(func() { Configure(false, "") })
	viper.Set(config.ViperKeyVerbose, false)
	path := filepath.Join(t.TempDir(), "homie.log")
	viper.Set(config.ViperKeyLogFile, "")

	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().BoolP("verbose", "v", false, "verbosity")
	cmd.Flags().String("log-file", "", "log file")
	if err := cmd.ParseFlags([]string{"-v", "--log-file", path}); err != nil {
		t.Fatal(err)
	}

	ConfigureFromFlags(cmd.Flags())
	Logger().Printf("tee-line\n")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "tee-line") {
		t.Fatalf("expected tee line in file, got %q", string(data))
	}
}
