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

// Config holds resolved homie settings after load and normalize.
type Config struct {
	LogFile   string
	PIDFile   string
	Limit     int
	TTL       int
	Keep      int
	Threshold int
	Verbose   bool
}

// Parse builds Config from a populated viper instance (after ReadConfig).
func Parse(v *viper.Viper) *Config {
	c := &Config{
		LogFile:   v.GetString(LogFile),
		PIDFile:   v.GetString(PIDFile),
		Limit:     v.GetInt(Limit),
		TTL:       v.GetInt(TTL),
		Keep:      v.GetInt(Keep),
		Threshold: v.GetInt(Threshold),
		Verbose:   v.GetBool(Verbose),
	}
	c.normalize()
	return c
}

func (c *Config) normalize() {
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
