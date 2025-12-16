package clipboard

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"golang.design/x/clipboard"

	"github.com/kaliv0/homie/internal/storage"
)

// TrackClipboard watches for clipboard text changes and persists them.
func TrackClipboard(db *storage.Repository) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	go func() {
		select {
		case <-sigChan:
			cancel()
		case <-ctx.Done():
		}
	}()

	if err := clipboard.Init(); err != nil {
		return err
	}

	changes := clipboard.Watch(ctx, clipboard.FmtText)
	defer func() {
		_ = db.Close()
	}()

	for {
		select {
		case item, ok := <-changes:
			if !ok {
				return nil
			}
			if err := db.Write(item); err != nil {
				return err
			}
		case <-ctx.Done():
			return nil
		}
	}
}
