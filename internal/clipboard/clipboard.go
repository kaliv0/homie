package clipboard

import (
	"bytes"
	"context"
	"fmt"
	"time"

	gclip "golang.design/x/clipboard"
)

const writeTimeout = 200 * time.Millisecond

// Writer persists clipboard content.
type Writer interface {
	Write(item []byte) error
}

// TrackClipboard watches for clipboard text changes and persists them.
func TrackClipboard(ctx context.Context, w Writer, changes <-chan gclip.Data) error {
	for {
		select {
		case item, ok := <-changes:
			if !ok {
				return nil
			}
			if len(bytes.TrimSpace(item.Bytes)) == 0 {
				return nil
			}

			if err := w.Write(item.Bytes); err != nil {
				return err
			}
		case <-ctx.Done():
			return nil
		}
	}
}

// WriteSelection writes text to the system clipboard.
func WriteSelection(text string) error {
	if err := gclip.Init(); err != nil {
		return fmt.Errorf("failed to initialize clipboard: %w", err)
	}

	_, err := gclip.Write(context.Background(), gclip.FmtText, []byte(text))
	if err != nil {
		return fmt.Errorf("failed to write to clipboard: %w", err)
	}

	// keep gclip's serve goroutine alive so the compositor can settle the new selection
	// [this library still feels like a complete garbage]
	time.Sleep(writeTimeout)
	return nil
}
