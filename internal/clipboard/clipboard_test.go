package clipboard

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	gclip "golang.design/x/clipboard"
)

var errWrite = errors.New("write failed")

// mockWriter records writes -> returns errWrite on call number failOn (0 means never).
type mockWriter struct {
	items  []string
	calls  int
	failOn int
}

func (m *mockWriter) Write(item []byte) error {
	m.calls++
	if m.calls == m.failOn {
		return errWrite
	}
	m.items = append(m.items, string(item))
	return nil
}

func text(s string) gclip.Data {
	return gclip.Data{Format: gclip.FmtText, Bytes: []byte(s)}
}

// assertTrackClipboardDone waits for TrackClipboard to finish and expects a nil error.
func assertTrackClipboardDone(t *testing.T, done <-chan error) {
	t.Helper()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("TrackClipboard did not return")
	}
}

func TestTrackClipboard(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		in        []gclip.Data
		failOn    int
		want      []string
		wantErr   bool
		wantCalls int
	}{
		{name: "receives items", in: []gclip.Data{text("item1"), text("item2"), text("item3")},
			want: []string{"item1", "item2", "item3"}, wantCalls: 3},
		{name: "no items", wantCalls: 0},
		{name: "skips nil", in: []gclip.Data{{Format: gclip.FmtText}}, wantCalls: 0},
		{name: "skips empty", in: []gclip.Data{text("")}, wantCalls: 0},
		{name: "skips spaces", in: []gclip.Data{text("   ")}, wantCalls: 0},
		{name: "skips tabs and newlines", in: []gclip.Data{text("\t\n\r ")}, wantCalls: 0},
		{name: "keeps whitespace padded content", in: []gclip.Data{text("  hello  ")},
			want: []string{"  hello  "}, wantCalls: 1},
		{name: "empty item stops watch", in: []gclip.Data{text("first"), text("  "), text("after-empty")},
			want: []string{"first"}, wantCalls: 1},
		{name: "write error", in: []gclip.Data{text("data")}, failOn: 1,
			wantErr: true, wantCalls: 1},
		{name: "write error on second item", in: []gclip.Data{text("first"), text("second"), text("third")}, failOn: 2,
			want: []string{"first"}, wantErr: true, wantCalls: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ch := make(chan gclip.Data, len(tt.in))
			for _, item := range tt.in {
				ch <- item
			}
			close(ch)

			writer := &mockWriter{failOn: tt.failOn}
			err := TrackClipboard(t.Context(), writer, ch)

			if tt.wantErr {
				if !errors.Is(err, errWrite) {
					t.Fatalf("expected write error, got %v", err)
				}
			} else if err != nil {
				t.Fatalf("TrackClipboard() failed: %v", err)
			}
			if writer.calls != tt.wantCalls {
				t.Errorf("Write calls = %d, want %d", writer.calls, tt.wantCalls)
			}
			if !slices.Equal(writer.items, tt.want) {
				t.Errorf("written = %q, want %q", writer.items, tt.want)
			}
		})
	}
}

func TestTrackClipboard_ContextCancellation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())

	done := make(chan error, 1)
	go func() {
		done <- TrackClipboard(ctx, &mockWriter{}, make(chan gclip.Data))
	}()

	cancel()
	assertTrackClipboardDone(t, done)
}

func TestTrackClipboard_ChannelClose(t *testing.T) {
	t.Parallel()
	ch := make(chan gclip.Data)

	done := make(chan error, 1)
	go func() {
		done <- TrackClipboard(t.Context(), &mockWriter{}, ch)
	}()

	close(ch)
	assertTrackClipboardDone(t, done)
}

func TestTrackClipboard_ContextCancelDuringItems(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())

	ch := make(chan gclip.Data)
	writer := &mockWriter{}

	done := make(chan error, 1)
	go func() {
		done <- TrackClipboard(ctx, writer, ch)
	}()

	ch <- text("before-cancel")
	cancel()
	assertTrackClipboardDone(t, done)

	if len(writer.items) != 1 {
		t.Errorf("expected 1 item processed before cancel, got %d", len(writer.items))
	}
}
