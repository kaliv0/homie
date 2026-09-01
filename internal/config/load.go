package config

import (
	"fmt"
	"os"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	DefaultLimit   = 20
	DefaultMaxSize = 500
)

const (
	ClipboardXclip       = "xclip"
	ClipboardXsel        = "xsel"
	ClipboardWLClipboard = "wl-clipboard"
)

// Config holds resolved homie settings after load and normalize.
type Config struct {
	Verbose bool
	LogFile string
	PIDFile string

	Limit   int
	CleanUp bool
	TTL     int
	MaxSize int

	ClipboardTool string
}

// Parse builds Config from a populated viper instance (after ReadConfig).
func Parse(v *viper.Viper) (*Config, error) {
	clipboard, err := resolveClipboard(v)
	if err != nil {
		return nil, err
	}

	c := &Config{
		Verbose:       v.GetBool(KeyVerbose),
		LogFile:       v.GetString(KeyLogFile),
		PIDFile:       v.GetString(KeyPIDFile),
		Limit:         v.GetInt(KeyLimit),
		CleanUp:       v.GetBool(KeyCleanUp),
		TTL:           v.GetInt(KeyTTL),
		MaxSize:       v.GetInt(KeyMaxSize),
		ClipboardTool: clipboard,
	}
	c.normalize()
	return c, nil
}

// BindFlag binds a cobra/pflag flag to a viper key. Exits on failure.
func BindFlag(key string, flag *pflag.Flag) {
	if err := viper.BindPFlag(key, flag); err != nil {
		fmt.Fprintf(os.Stderr, "homie: failed to bind %q flag to viper: %v\n", key, err)
		os.Exit(1)
	}
}

func resolveClipboard(v *viper.Viper) (string, error) {
	type toolKey struct {
		key  string
		tool string
	}
	keys := []toolKey{
		{KeyUseXclip, ClipboardXclip},
		{KeyUseXsel, ClipboardXsel},
		{KeyUseWLClipboard, ClipboardWLClipboard},
	}

	var chosen string
	enabled := 0
	for _, tk := range keys {
		if v.GetBool(tk.key) {
			enabled++
			chosen = tk.tool
		}
	}
	if enabled > 1 {
		return "", fmt.Errorf(
			"config: only one clipboard tool may be enabled (%s, %s, %s)",
			KeyUseXclip, KeyUseXsel, KeyUseWLClipboard,
		)
	}
	if enabled == 1 {
		return chosen, nil
	}
	return "", nil
}

func (c *Config) normalize() {
	warn := func(string) {}
	if c.Verbose {
		warn = func(msg string) { fmt.Fprintln(os.Stderr, msg) }
	}

	if c.TTL < 0 {
		warn(fmt.Sprintf("config: ttl %d is negative, using 0", c.TTL))
		c.TTL = 0
	}
	if c.Limit < 0 {
		warn(fmt.Sprintf("config: limit %d is negative, using %d", c.Limit, DefaultLimit))
		c.Limit = DefaultLimit
	} else if c.Limit == 0 {
		c.Limit = DefaultLimit
	}
	if c.MaxSize < 0 {
		warn(fmt.Sprintf("config: max_size %d is negative, using %d", c.MaxSize, DefaultMaxSize))
		c.MaxSize = DefaultMaxSize
	} else if c.MaxSize == 0 {
		c.MaxSize = DefaultMaxSize
	}
	c.resolvePaths()
}
