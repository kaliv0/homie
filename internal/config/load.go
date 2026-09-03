package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

const (
	DefaultLimit = 20

	DefaultTTL       = 0
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
	LogFile   string
	PIDFile   string
	Tool      string
	Limit     int
	TTL       int
	Keep      int
	Threshold int
	Verbose   bool
}

// Parse builds Config from a populated viper instance (after ReadConfig).
func Parse(v *viper.Viper) (*Config, error) {
	c := &Config{
		LogFile:   v.GetString(LogFile),
		PIDFile:   v.GetString(PIDFile),
		Tool:      v.GetString(Tool),
		Limit:     v.GetInt(Limit),
		TTL:       v.GetInt(TTL),
		Keep:      v.GetInt(Keep),
		Threshold: v.GetInt(Threshold),
		Verbose:   v.GetBool(Verbose),
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

	for _, n := range []struct {
		name string
		val  *int
		def  int
	}{
		{Limit, &c.Limit, DefaultLimit},
		{TTL, &c.TTL, DefaultTTL},
		{Keep, &c.Keep, DefaultKeep},
		{Threshold, &c.Threshold, DefaultThreshold},
	} {
		if *n.val < 0 {
			warn("config: %s %d is negative, using %d\n", n.name, *n.val, n.def)
			*n.val = n.def
		}
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
