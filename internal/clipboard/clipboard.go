package clipboard

import (
	"context"
	"fmt"

	gclip "golang.design/x/clipboard"

	"github.com/kaliv0/homie/internal/storage"
)

// TrackClipboard watches for clipboard text changes and persists them.
func TrackClipboard(ctx context.Context, db *storage.Repository) error {
	if err := gclip.Init(); err != nil {
		return fmt.Errorf("failed to initialize clipboard: %w", err)
	}
	changes := gclip.Watch(ctx, gclip.FmtText)

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
