package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

const (
	DefaultMinSize = 20
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

	MinSize int
	CleanUp bool
	TTL     int
	MaxSize int

	Tool string
}

// Parse builds Config from a populated viper instance (after ReadConfig).
func Parse(v *viper.Viper) (*Config, error) {
	c := &Config{
		Verbose: v.GetBool(Verbose),
		LogFile: v.GetString(LogFile),
		PIDFile: v.GetString(PIDFile),
		MinSize: v.GetInt(MinSize),
		CleanUp: v.GetBool(CleanUp),
		TTL:     v.GetInt(TTL),
		MaxSize: v.GetInt(MaxSize),
		Tool:    v.GetString(Tool),
	}
	c.normalize()
	return c, nil
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

	if c.MinSize < 0 {
		warn("config: min_size %d is negative, using %d\n", c.MinSize, DefaultMinSize)
		c.MinSize = DefaultMinSize
	} else if c.MinSize == 0 {
		c.MinSize = DefaultMinSize
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
