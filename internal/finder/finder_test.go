package finder

import (
	"errors"
	"slices"
	"sync"
	"testing"

	"github.com/kaliv0/homie/internal/storage"
)

type mockReader struct {
	pages map[int][]storage.ClipboardItem
	err   error
	calls int // safe to read after the loader goroutine exits
}

func (m *mockReader) Read(offset, _ int) ([]storage.ClipboardItem, error) {
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	return m.pages[offset], nil
}

func (m *mockReader) Count() (int, error) {
	return 0, nil
}

// runLoads sends n signals -> closes the channel -> waits for the loader to drain.
func runLoads(t *testing.T, r HistoryReader, init []storage.ClipboardItem, offset, limit, total, n int) *session {
	t.Helper()
	s := &session{history: slices.Clone(init)}
	var wg sync.WaitGroup
	loadMore := handleLoadChannel(s, r, offset, limit, total, &wg)
	for range n {
		loadMore <- struct{}{}
	}
	close(loadMore)
	wg.Wait()
	return s
}

func initItems(n int) []storage.ClipboardItem {
	items := make([]storage.ClipboardItem, n)
	for i := range items {
		items[i] = storage.ClipboardItem{ID: i + 1, ClipText: "init"}
	}
	return items
}

func TestHandleLoadChannel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		pages     map[int][]storage.ClipboardItem
		err       error
		initLen   int
		offset    int
		limit     int
		total     int
		signals   int
		wantCalls int
		wantTexts []string // when set, verify loaded items after init
		wantLen   int
	}{
		{
			name: "loads pages until total",
			pages: map[int][]storage.ClipboardItem{
				5:  {{ID: 6, ClipText: "p2-1"}, {ID: 7, ClipText: "p2-2"}},
				10: {{ID: 8, ClipText: "p3-1"}},
			},
			// third signal -> next offset 15 == total -> no Read
			initLen: 1, offset: 0, limit: 5, total: 15, signals: 3,
			wantCalls: 2, wantLen: 4, wantTexts: []string{"p2-1", "p2-2", "p3-1"},
		},
		{
			name: "limit one sequential loads",
			pages: map[int][]storage.ClipboardItem{
				1: {{ID: 2, ClipText: "second"}},
				2: {{ID: 3, ClipText: "third"}},
			},
			initLen: 1, offset: 0, limit: 1, total: 3, signals: 2,
			wantCalls: 2, wantLen: 3, wantTexts: []string{"second", "third"},
		},
		{
			name: "partial page",
			pages: map[int][]storage.ClipboardItem{
				5: {{ID: 6, ClipText: "partial-1"}, {ID: 7, ClipText: "partial-2"}},
			},
			initLen: 1, offset: 0, limit: 5, total: 7, signals: 1,
			wantCalls: 1, wantLen: 3,
		},
		{
			name:   "stops at total",
			offset: 5, limit: 5, total: 5, signals: 1,
			wantCalls: 0, wantLen: 0,
		},
		{
			name:   "offset already at end",
			offset: 10, limit: 5, total: 10, signals: 1,
			wantCalls: 0, wantLen: 0,
		},
		{
			name:   "read error keeps history",
			err:    errors.New("db error"),
			offset: 0, limit: 5, total: 100, signals: 1,
			wantCalls: 1, wantLen: 0,
		},
		{
			name:   "channel close without signals",
			offset: 0, limit: 5, total: 100, signals: 0,
			wantCalls: 0, wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			reader := &mockReader{pages: tt.pages, err: tt.err}
			s := runLoads(t, reader, initItems(tt.initLen), tt.offset, tt.limit, tt.total, tt.signals)

			if reader.calls != tt.wantCalls {
				t.Errorf("Read calls = %d, want %d", reader.calls, tt.wantCalls)
			}
			if len(s.history) != tt.wantLen {
				t.Fatalf("history len = %d, want %d", len(s.history), tt.wantLen)
			}
			if tt.wantTexts != nil {
				var got []string
				for _, item := range s.history[tt.initLen:] {
					got = append(got, item.ClipText)
				}
				if !slices.Equal(got, tt.wantTexts) {
					t.Errorf("loaded texts = %q, want %q", got, tt.wantTexts)
				}
			}
		})
	}
}
