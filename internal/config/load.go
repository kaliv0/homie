package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

const (
	DefaultLimit     = 20
	DefaultKeep      = 20
	DefaultThreshold = 500
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

	Limit     int
	TTL       int
	Keep      int
	Threshold int

	Tool string
}

// Parse builds Config from a populated viper instance (after ReadConfig).
func Parse(v *viper.Viper) (*Config, error) {
	c := &Config{
		Verbose:   v.GetBool(Verbose),
		LogFile:   v.GetString(LogFile),
		PIDFile:   v.GetString(PIDFile),
		Limit:     v.GetInt(Limit),
		TTL:       v.GetInt(TTL),
		Keep:      v.GetInt(Keep),
		Threshold: v.GetInt(Threshold),
		Tool:      v.GetString(Tool),
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

	if c.Limit < 0 {
		warn("config: limit %d is negative, using %d\n", c.Limit, DefaultLimit)
		c.Limit = DefaultLimit
	}

	if c.TTL < 0 {
		warn("config: ttl %d is negative, using 0\n", c.TTL)
		c.TTL = 0
	}

	if c.Keep <= 0 {
		warn("config: keep %d must positive, using %d\n", c.Keep, DefaultKeep)
		c.Keep = DefaultKeep
	}

	if c.Threshold <= 0 {
		warn("config: threshold %d must positive, using %d\n", c.Threshold, DefaultThreshold)
		c.Threshold = DefaultThreshold
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
