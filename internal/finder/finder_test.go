package finder

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/kaliv0/homie/internal/storage"
)

type mockReader struct {
	pages   map[int][]storage.ClipboardItem
	readErr error
	count   int
}

func (m *mockReader) Read(offset, _ int) ([]storage.ClipboardItem, error) {
	if m.readErr != nil {
		return nil, m.readErr
	}
	if page, ok := m.pages[offset]; ok {
		return page, nil
	}
	return nil, nil
}

func (m *mockReader) Count() (int, error) {
	return m.count, nil
}

func (m *mockReader) Close() error {
	return nil
}

// countingMockReader counts Read calls.
type countingMockReader struct {
	reader    *mockReader
	callCount int
}

func (c *countingMockReader) Read(offset, limit int) ([]storage.ClipboardItem, error) {
	c.callCount++
	return c.reader.Read(offset, limit)
}

func (c *countingMockReader) Count() (int, error) {
	return c.reader.Count()
}

func (c *countingMockReader) Close() error {
	return nil
}

// loadChannelFixture is shared test state.
type loadChannelFixture struct {
	history  []storage.ClipboardItem
	wg       sync.WaitGroup
	loadMore chan struct{}
}

// newLoadChannelFixture creates a fixture and starts the load channel goroutine.
func newLoadChannelFixture(t *testing.T, reader HistoryReader, initHistory []storage.ClipboardItem,
	offset, limit, total int) *loadChannelFixture {
	t.Helper()
	ctx := t.Context()

	f := &loadChannelFixture{
		history: append([]storage.ClipboardItem{}, initHistory...),
	}
	f.loadMore = handleLoadChannel(ctx, &f.history, reader, offset, limit, total, &f.wg)

	t.Cleanup(func() {
		close(f.loadMore)
		f.wg.Wait()
	})
	return f
}

// triggerLoad sends a load signal.
func (f *loadChannelFixture) triggerLoad() {
	f.loadMore <- struct{}{}
	time.Sleep(time.Millisecond)
}

// historyLen returns the current history length.
func (f *loadChannelFixture) historyLen() int {
	mu.RLock()
	defer mu.RUnlock()
	return len(f.history)
}

func TestHandleLoadChannel_LoadsPages(t *testing.T) {
	reader := &mockReader{
		pages: map[int][]storage.ClipboardItem{
			5: {{ID: 1, ClipText: "page2-item1"}, {ID: 2, ClipText: "page2-item2"}},
		},
		count: 10,
	}
	f := newLoadChannelFixture(t, reader, []storage.ClipboardItem{{ID: 0, ClipText: "page1-item1"}}, 0, 5, 10)

	f.triggerLoad()

	if n := f.historyLen(); n != 3 {
		t.Errorf("expected 3 items in history, got %d", n)
	}
}

func TestHandleLoadChannel_StopsAtTotal(t *testing.T) {
	reader := &mockReader{pages: map[int][]storage.ClipboardItem{}, count: 5}
	countReader := &countingMockReader{reader: reader, callCount: 0}

	f := newLoadChannelFixture(t, countReader, nil, 5, 5, 5)
	f.triggerLoad()

	if countReader.callCount != 0 {
		t.Errorf("expected 0 reads when offset >= total, got %d", countReader.callCount)
	}
}

func TestHandleLoadChannel_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	reader := &mockReader{count: 100}
	var history []storage.ClipboardItem
	var wg sync.WaitGroup
	loadMore := handleLoadChannel(ctx, &history, reader, 0, 5, 100, &wg)

	cancel()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("goroutine did not exit after context cancel")
	}
	close(loadMore)
}

func TestHandleLoadChannel_ReadError(t *testing.T) {
	reader := &mockReader{readErr: errors.New("db error"), count: 100}
	f := newLoadChannelFixture(t, reader, nil, 0, 5, 100)

	f.triggerLoad()

	if n := f.historyLen(); n != 0 {
		t.Errorf("expected 0 items after read error, got %d", n)
	}
}

func TestHandleLoadChannel_ChannelClose(t *testing.T) {
	reader := &mockReader{count: 100}
	var history []storage.ClipboardItem
	var wg sync.WaitGroup
	loadMore := handleLoadChannel(t.Context(), &history, reader, 0, 5, 100, &wg)

	close(loadMore)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("goroutine did not exit after channel close")
	}
}

func TestHandleLoadChannel_Limits(t *testing.T) {
	tests := []struct {
		name    string
		pages   map[int][]storage.ClipboardItem
		total   int
		offset  int
		limit   int
		signals int
		wantMin int
		wantMax int
		initLen int
	}{
		{
			name: "limit one sequential loads",
			pages: map[int][]storage.ClipboardItem{
				1: {{ID: 2, ClipText: "second"}},
				2: {{ID: 3, ClipText: "third"}},
			},
			total: 3, offset: 0, limit: 1, signals: 2, wantMin: 3, wantMax: 3, initLen: 1,
		},
		{
			name: "large limit",
			pages: map[int][]storage.ClipboardItem{
				100: {{ID: 2, ClipText: "page2"}},
			},
			total: 200, offset: 0, limit: 100, signals: 1, wantMin: 2, wantMax: 2, initLen: 1,
		},
		{
			name:  "zero total",
			pages: map[int][]storage.ClipboardItem{},
			total: 0, offset: 0, limit: 5, signals: 1, wantMin: 0, wantMax: 0, initLen: 0,
		},
		{
			name:  "offset already at end",
			pages: map[int][]storage.ClipboardItem{},
			total: 10, offset: 10, limit: 5, signals: 1, wantMin: 0, wantMax: 0, initLen: 0,
		},
		{
			name: "partial page",
			pages: map[int][]storage.ClipboardItem{
				5: {{ID: 6, ClipText: "partial-1"}, {ID: 7, ClipText: "partial-2"}},
			},
			total: 7, offset: 0, limit: 5, signals: 1, wantMin: 3, wantMax: 3, initLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := &mockReader{pages: tt.pages, count: tt.total}
			init := make([]storage.ClipboardItem, tt.initLen)
			for i := range init {
				init[i] = storage.ClipboardItem{ID: i + 1, ClipText: "init"}
			}

			f := newLoadChannelFixture(t, reader, init, tt.offset, tt.limit, tt.total)

			for range tt.signals {
				f.triggerLoad()
			}

			n := f.historyLen()
			if n < tt.wantMin || n > tt.wantMax {
				t.Errorf("expected history len in [%d, %d], got %d", tt.wantMin, tt.wantMax, n)
			}
		})
	}
}

func TestHandleLoadChannel_MultipleLoads(t *testing.T) {
	reader := &mockReader{
		pages: map[int][]storage.ClipboardItem{
			5:  {{ID: 6, ClipText: "p2-1"}, {ID: 7, ClipText: "p2-2"}},
			10: {{ID: 8, ClipText: "p3-1"}},
			15: {},
		},
		count: 15,
	}
	f := newLoadChannelFixture(t, reader, []storage.ClipboardItem{{ID: 1, ClipText: "p1-1"}}, 0, 5, 15)

	f.triggerLoad()
	if n := f.historyLen(); n != 3 {
		t.Errorf("after page 2: expected 3 items, got %d", n)
	}

	f.triggerLoad()
	if n := f.historyLen(); n != 4 {
		t.Errorf("after page 3: expected 4 items, got %d", n)
	}

	f.triggerLoad()
	if n := f.historyLen(); n != 4 {
		t.Errorf("after empty page: expected 4 items, got %d", n)
	}
}

func TestHandleLoadChannel_RapidSignals(t *testing.T) {
	reader := &mockReader{
		pages: map[int][]storage.ClipboardItem{
			5:  {{ID: 2, ClipText: "p2"}},
			10: {{ID: 3, ClipText: "p3"}},
			15: {{ID: 4, ClipText: "p4"}},
		},
		count: 20,
	}
	f := newLoadChannelFixture(t, reader, []storage.ClipboardItem{{ID: 1, ClipText: "init"}}, 0, 5, 20)

	for range 5 {
		select {
		case f.loadMore <- struct{}{}:
		default:
		}
	}
	time.Sleep(5 * time.Millisecond)

	if n := f.historyLen(); n < 2 {
		t.Errorf("expected at least 2 items after rapid signals, got %d", n)
	}
}
