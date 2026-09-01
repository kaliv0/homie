package config

import (
	"fmt"
	"os"
	"strings"

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

	Tool string
}

// Parse builds Config from a populated viper instance (after ReadConfig).
func Parse(v *viper.Viper) (*Config, error) {
	c := &Config{
		Verbose: v.GetBool(KeyVerbose),
		LogFile: v.GetString(KeyLogFile),
		PIDFile: v.GetString(KeyPIDFile),
		Limit:   v.GetInt(KeyLimit),
		CleanUp: v.GetBool(KeyCleanUp),
		TTL:     v.GetInt(KeyTTL),
		MaxSize: v.GetInt(KeyMaxSize),
		Tool:    strings.TrimSpace(v.GetString(KeyTool)),
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

func (c *Config) normalize() {
	if c.Tool != "" && !knownTool(c.Tool) {
		// NB: always warn user if clipboard tool is messed up
		fmt.Fprintf(os.Stderr,
			"config: unknown tool %q, ignoring (want %q, %q, or %q)\n",
			c.Tool, ClipboardXclip, ClipboardXsel, ClipboardWLClipboard,
		)
		c.Tool = ""
	}

	warn := func(string, ...any) {}
	if c.Verbose {
		warn = func(format string, args ...any) {
			fmt.Fprintf(os.Stderr, format, args...)
		}
	}

	if c.TTL < 0 {
		warn("config: ttl %d is negative, using 0\n", c.TTL)
		c.TTL = 0
	}

	if c.Limit < 0 {
		warn("config: limit %d is negative, using %d\n", c.Limit, DefaultLimit)
		c.Limit = DefaultLimit
	} else if c.Limit == 0 {
		c.Limit = DefaultLimit
	}

	if c.MaxSize < 0 {
		warn("config: max_size %d is negative, using %d\n", c.MaxSize, DefaultMaxSize)
		c.MaxSize = DefaultMaxSize
	} else if c.MaxSize == 0 {
		c.MaxSize = DefaultMaxSize
	}

	c.resolvePaths()
}

func knownTool(tool string) bool {
	switch tool {
	case ClipboardXclip, ClipboardXsel, ClipboardWLClipboard:
		return true
	default:
		return false
	}
}
