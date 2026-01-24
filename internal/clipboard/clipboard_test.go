package clipboard

import (
	"context"
	"errors"
	"testing"
	"time"
)

type mockWriter struct {
	items [][]byte
	err   error
}

func (m *mockWriter) Write(item []byte) error {
	if m.err != nil {
		return m.err
	}
	m.items = append(m.items, item)
	return nil
}

// conditionalMockWriter fails on a specific call number.
type conditionalMockWriter struct {
	failOnCall int
	err        error
	callCount  int
}

func (m *conditionalMockWriter) Write(item []byte) error {
	m.callCount++
	if m.callCount == m.failOnCall {
		return m.err
	}
	return nil
}

// trackClosed sends items to a buffered channel, closes it, and runs TrackClipboard.
func trackClosed(t *testing.T, writer Writer, items ...[]byte) error {
	t.Helper()
	ch := make(chan []byte, len(items))
	for _, item := range items {
		ch <- item
	}
	close(ch)

	return TrackClipboard(t.Context(), writer, ch)
}

func TestTrackClipboard_ReceivesItems(t *testing.T) {
	t.Parallel()

	writer := &mockWriter{}
	err := trackClosed(t, writer, []byte("item1"), []byte("item2"), []byte("item3"))
	if err != nil {
		t.Fatalf("TrackClipboard() failed: %v", err)
	}

	if len(writer.items) != 3 {
		t.Fatalf("expected 3 items written, got %d", len(writer.items))
	}
	if string(writer.items[0]) != "item1" {
		t.Errorf("expected first item=%q, got %q", "item1", writer.items[0])
	}
	if string(writer.items[2]) != "item3" {
		t.Errorf("expected third item=%q, got %q", "item3", writer.items[2])
	}
}

func TestTrackClipboard_ContextCancellation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())

	ch := make(chan []byte)
	writer := &mockWriter{}

	done := make(chan error, 1)
	go func() {
		done <- TrackClipboard(ctx, writer, ch)
	}()

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected nil error on context cancel, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("TrackClipboard did not return after context cancellation")
	}
}

func TestTrackClipboard_ChannelClose(t *testing.T) {
	t.Parallel()
	ch := make(chan []byte)
	writer := &mockWriter{}

	done := make(chan error, 1)
	go func() {
		done <- TrackClipboard(t.Context(), writer, ch)
	}()

	close(ch)

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected nil error on channel close, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("TrackClipboard did not return after channel close")
	}
}

func TestTrackClipboard_WriteError(t *testing.T) {
	t.Parallel()
	writeErr := errors.New("write failed")
	writer := &mockWriter{err: writeErr}

	err := trackClosed(t, writer, []byte("data"))
	if !errors.Is(err, writeErr) {
		t.Fatalf("expected write error, got %v", err)
	}
}

func TestTrackClipboard_ItemVariants(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		item    []byte
		wantLen int
	}{
		{"empty item", []byte(""), 0},
		{"nil item", nil, 0},
		{"large item", make([]byte, 100000), 100000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := &mockWriter{}
			if err := trackClosed(t, writer, tt.item); err != nil {
				t.Fatalf("TrackClipboard() failed: %v", err)
			}
			if len(writer.items) != 1 {
				t.Fatalf("expected 1 item, got %d", len(writer.items))
			}
			if tt.wantLen > 0 && len(writer.items[0]) != tt.wantLen {
				t.Errorf("expected %d bytes, got %d", tt.wantLen, len(writer.items[0]))
			}
		})
	}
}

func TestTrackClipboard_ManyItems(t *testing.T) {
	t.Parallel()
	writer := &mockWriter{}
	items := make([][]byte, 100)
	for i := range items {
		items[i] = []byte("item")
	}

	if err := trackClosed(t, writer, items...); err != nil {
		t.Fatalf("TrackClipboard() failed: %v", err)
	}
	if len(writer.items) != 100 {
		t.Errorf("expected 100 items, got %d", len(writer.items))
	}
}

func TestTrackClipboard_WriteErrorOnSecondItem(t *testing.T) {
	t.Parallel()
	writeErr := errors.New("second write failed")
	writer := &conditionalMockWriter{failOnCall: 2, err: writeErr}

	err := trackClosed(t, writer, []byte("first"), []byte("second"))
	if !errors.Is(err, writeErr) {
		t.Fatalf("expected write error on second item, got %v", err)
	}
	if writer.callCount != 2 {
		t.Errorf("expected 2 write calls, got %d", writer.callCount)
	}
}

func TestTrackClipboard_ContextCancelDuringItems(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())

	ch := make(chan []byte, 5)
	writer := &mockWriter{}

	ch <- []byte("before-cancel")

	done := make(chan error, 1)
	go func() {
		done <- TrackClipboard(ctx, writer, ch)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("TrackClipboard did not exit after cancel")
	}

	if len(writer.items) != 1 {
		t.Errorf("expected 1 item processed before cancel, got %d", len(writer.items))
	}
}
