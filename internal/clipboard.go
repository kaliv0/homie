package internal

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"golang.design/x/clipboard"
)

// TrackClipboard watches for clipboard text changes and persists them.
func TrackClipboard(db *Repository) {
	if err := clipboard.Init(); err != nil {
		Logger.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	go func() {
		<-sigChan
		cancel()
	}()

	changes := clipboard.Watch(ctx, clipboard.FmtText)
	for {
		select {
		case item, ok := <-changes:
			if !ok {
				db.Close()
				return
			}
			db.Write(item)
		case <-ctx.Done():
			db.Close()
			return
		}
	}
}
