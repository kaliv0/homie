package clipboard

import (
	"context"

	gclip "golang.design/x/clipboard"
)

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
			if err := w.Write(item.Bytes); err != nil {
				return err
			}
		case <-ctx.Done():
			return nil
		}
	}
}
