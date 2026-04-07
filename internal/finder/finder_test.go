package finder

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/kaliv0/homie/internal/storage"
)

type mockReader struct {
	pages     map[int][]storage.ClipboardItem
	readErr   error
	count     int
	readCalls chan struct{}
}

func (m *mockReader) Read(offset, _ int) ([]storage.ClipboardItem, error) {
	if m.readCalls != nil {
		m.readCalls <- struct{}{}
	}
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

func newMockReader(pages map[int][]storage.ClipboardItem, readErr error, count int) *mockReader {
	return &mockReader{
		pages:     pages,
		readErr:   readErr,
		count:     count,
		readCalls: make(chan struct{}, 64),
	}
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

func waitForReads(t *testing.T, calls <-chan struct{}, n int) {
	t.Helper()
	for range n {
		<-calls
	}
}

type limitsCase struct {
	name    string
	pages   map[int][]storage.ClipboardItem
	total   int
	offset  int
	limit   int
	signals int
	wantMin int
	wantMax int
	initLen int
}

func readsToWait(tc limitsCase) int {
	if tc.limit <= 0 || tc.offset >= tc.total || tc.signals <= 0 {
		return 0
	}

	maxReads := (tc.total - tc.offset + tc.limit - 1) / tc.limit
	if tc.signals < maxReads {
		return tc.signals
	}
	return maxReads
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

// historyLen returns the current history length.
func (f *loadChannelFixture) historyLen() int {
	mu.RLock()
	defer mu.RUnlock()
	return len(f.history)
}

func TestHandleLoadChannel_LoadsPages(t *testing.T) {
	reader := newMockReader(
		map[int][]storage.ClipboardItem{
			5: {{ID: 1, ClipText: "page2-item1"}, {ID: 2, ClipText: "page2-item2"}},
		},
		nil,
		10,
	)
	f := newLoadChannelFixture(t, reader, []storage.ClipboardItem{{ID: 0, ClipText: "page1-item1"}}, 0, 5, 10)

	f.loadMore <- struct{}{}
	waitForReads(t, reader.readCalls, 1)

	if n := f.historyLen(); n != 3 {
		t.Errorf("expected 3 items in history, got %d", n)
	}
}

func TestHandleLoadChannel_StopsAtTotal(t *testing.T) {
	reader := newMockReader(map[int][]storage.ClipboardItem{}, nil, 5)
	countReader := &countingMockReader{reader: reader, callCount: 0}

	f := newLoadChannelFixture(t, countReader, nil, 5, 5, 5)
	f.loadMore <- struct{}{}

	if countReader.callCount != 0 {
		t.Errorf("expected 0 reads when offset >= total, got %d", countReader.callCount)
	}
}

func TestHandleLoadChannel_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	reader := newMockReader(nil, nil, 100)
	var history []storage.ClipboardItem
	var wg sync.WaitGroup
	loadMore := handleLoadChannel(ctx, &history, reader, 0, 5, 100, &wg)

	cancel()

	wg.Wait()
	close(loadMore)
}

func TestHandleLoadChannel_ReadError(t *testing.T) {
	reader := newMockReader(nil, errors.New("db error"), 100)
	f := newLoadChannelFixture(t, reader, nil, 0, 5, 100)

	f.loadMore <- struct{}{}
	waitForReads(t, reader.readCalls, 1)

	if n := f.historyLen(); n != 0 {
		t.Errorf("expected 0 items after read error, got %d", n)
	}
}

func TestHandleLoadChannel_ChannelClose(t *testing.T) {
	reader := newMockReader(nil, nil, 100)
	var history []storage.ClipboardItem
	var wg sync.WaitGroup
	loadMore := handleLoadChannel(t.Context(), &history, reader, 0, 5, 100, &wg)

	close(loadMore)
	wg.Wait()
}

func TestHandleLoadChannel_Limits(t *testing.T) {
	tests := []limitsCase{
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
			reader := newMockReader(tt.pages, nil, tt.total)
			init := make([]storage.ClipboardItem, tt.initLen)
			for i := range init {
				init[i] = storage.ClipboardItem{ID: i + 1, ClipText: "init"}
			}

			f := newLoadChannelFixture(t, reader, init, tt.offset, tt.limit, tt.total)

			for range tt.signals {
				f.loadMore <- struct{}{}
			}
			waitForReads(t, reader.readCalls, readsToWait(tt))

			n := f.historyLen()
			if n < tt.wantMin || n > tt.wantMax {
				t.Errorf("expected history len in [%d, %d], got %d", tt.wantMin, tt.wantMax, n)
			}
		})
	}
}

func TestHandleLoadChannel_MultipleLoads(t *testing.T) {
	reader := newMockReader(
		map[int][]storage.ClipboardItem{
			5:  {{ID: 6, ClipText: "p2-1"}, {ID: 7, ClipText: "p2-2"}},
			10: {{ID: 8, ClipText: "p3-1"}},
			15: {},
		},
		nil,
		15,
	)
	f := newLoadChannelFixture(t, reader, []storage.ClipboardItem{{ID: 1, ClipText: "p1-1"}}, 0, 5, 15)

	f.loadMore <- struct{}{}
	waitForReads(t, reader.readCalls, 1)
	if n := f.historyLen(); n != 3 {
		t.Errorf("after page 2: expected 3 items, got %d", n)
	}

	f.loadMore <- struct{}{}
	waitForReads(t, reader.readCalls, 1)
	if n := f.historyLen(); n != 4 {
		t.Errorf("after page 3: expected 4 items, got %d", n)
	}

	f.loadMore <- struct{}{}
	if n := f.historyLen(); n != 4 {
		t.Errorf("after empty page: expected 4 items, got %d", n)
	}
}

func TestHandleLoadChannel_RapidSignals(t *testing.T) {
	reader := newMockReader(
		map[int][]storage.ClipboardItem{
			5:  {{ID: 2, ClipText: "p2"}},
			10: {{ID: 3, ClipText: "p3"}},
			15: {{ID: 4, ClipText: "p4"}},
		},
		nil,
		20,
	)
	f := newLoadChannelFixture(t, reader, []storage.ClipboardItem{{ID: 1, ClipText: "init"}}, 0, 5, 20)

	for range 5 {
		select {
		case f.loadMore <- struct{}{}:
		default:
		}
	}
	waitForReads(t, reader.readCalls, 1)

	if n := f.historyLen(); n < 2 {
		t.Errorf("expected at least 2 items after rapid signals, got %d", n)
	}
}
