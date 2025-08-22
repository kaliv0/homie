package utils

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/shirou/gopsutil/process"
	"github.com/spf13/viper"
	"golang.design/x/clipboard"
)

var (
	Logger = log.New(os.Stderr, "Homey error: ", log.Llongfile)

	GetDbPath = sync.OnceValue(func() string {
		return getDbPath()
	})
)

func getDbPath() string {
	var subDirsList []string
	xdfConfig := os.Getenv("XDG_CONFIG_HOME")
	if xdfConfig != "" {
		subDirsList = append(subDirsList, xdfConfig)
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			Logger.Fatal(err)
		}
		subDirsList = append(subDirsList, homeDir, ".config")
	}
	subDirsList = append(subDirsList, "homey")
	configDir := filepath.Join(subDirsList...)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		Logger.Fatal(err)
	}
	return filepath.Join(configDir, "homey.db")
}

func StopAllInstances() {
	processes, err := process.Processes()
	if err != nil {
		Logger.Fatal(err)
	}
	for _, p := range processes {
		pName, err := p.Name()
		if err != nil {
			Logger.Fatal(err)
		}
		if pName == "homey" && int32(os.Getpid()) != p.Pid {
			if err := p.Terminate(); err != nil {
				Logger.Fatal(err)
			}
		}
	}
}

func CleanOldHistory(db *Repository) {
	ReadConfig()
	if shouldClean := viper.GetBool("clean_up"); !shouldClean {
		return
	}

	// ttl takes precedence over 'size limit' strategy
	if ttl := viper.GetInt("ttl"); ttl > 0 {
		db.DeleteOldest(ttl)
		return
	}

	maxSize := viper.GetInt("max_size")
	if maxSize == 0 {
		maxSize = 500
	}
	minLimit := viper.GetInt("limit")
	total := db.Count()
	if total > maxSize {
		if minLimit == 0 {
			minLimit = 30
		}
		db.DeleteExcess(total - minLimit)
	}
}

func TrackClipboard(db *Repository) {
	//  init clipboard handler,
	if err := clipboard.Init(); err != nil {
		Logger.Fatal(err)
	}
	// open new Watch chanel -> NB: tracks text but no images
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// setup signal channel
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	// shutdown goroutine
	go func() {
		<-sigChan
		cancel()
	}()

	changes := clipboard.Watch(ctx, clipboard.FmtText)
	// loop through & write to db any changes in clipboard
	for item := range changes {
		db.Write(item)
	}

	// shut down gracefully long-running task
	select {
	case <-ctx.Done():
		db.Close()
	}
}
