package finder

import (
	"errors"
	"sync"
	"testing"
	"time"

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

func waitForReads(t *testing.T, calls <-chan struct{}, n int) {
	t.Helper()
	for range n {
		select {
		case <-calls:
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for %d reads", n)
		}
	}
}

func assertNoRead(t *testing.T, calls <-chan struct{}) {
	t.Helper()
	select {
	case <-calls:
		t.Fatal("unexpected Read call")
	case <-time.After(50 * time.Millisecond):
	}
}

type loadChannelFixture struct {
	session  *session
	wg       sync.WaitGroup
	loadMore chan struct{}
}

func newLoadChannelFixture(t *testing.T, reader HistoryReader, initHistory []storage.ClipboardItem,
	offset, limit, total int) *loadChannelFixture {
	t.Helper()

	f := &loadChannelFixture{
		session: &session{history: append([]storage.ClipboardItem{}, initHistory...)},
	}
	f.loadMore = handleLoadChannel(f.session, reader, offset, limit, total, &f.wg)

	t.Cleanup(func() {
		close(f.loadMore)
		f.wg.Wait()
	})
	return f
}

func (f *loadChannelFixture) historyLen() int {
	f.session.mu.RLock()
	defer f.session.mu.RUnlock()
	return len(f.session.history)
}

func TestHandleLoadChannel_LoadsPages(t *testing.T) {
	reader := newMockReader(
		map[int][]storage.ClipboardItem{
			5:  {{ID: 6, ClipText: "p2-1"}, {ID: 7, ClipText: "p2-2"}},
			10: {{ID: 8, ClipText: "p3-1"}},
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

	// next offset would be 15 == total -> no Read
	f.loadMore <- struct{}{}
	assertNoRead(t, reader.readCalls)
	if n := f.historyLen(); n != 4 {
		t.Errorf("at total: expected 4 items, got %d", n)
	}
}

func TestHandleLoadChannel_StopsAtTotal(t *testing.T) {
	reader := newMockReader(map[int][]storage.ClipboardItem{}, nil, 5)
	f := newLoadChannelFixture(t, reader, nil, 5, 5, 5)

	f.loadMore <- struct{}{}
	assertNoRead(t, reader.readCalls)
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
	s := &session{}
	var wg sync.WaitGroup
	loadMore := handleLoadChannel(s, reader, 0, 5, 100, &wg)

	close(loadMore)
	wg.Wait()
}

func TestHandleLoadChannel_Limits(t *testing.T) {
	tests := []struct {
		name    string
		pages   map[int][]storage.ClipboardItem
		total   int
		offset  int
		limit   int
		signals int
		wantLen int
		initLen int
	}{
		{
			name: "limit one sequential loads",
			pages: map[int][]storage.ClipboardItem{
				1: {{ID: 2, ClipText: "second"}},
				2: {{ID: 3, ClipText: "third"}},
			},
			total: 3, offset: 0, limit: 1, signals: 2, wantLen: 3, initLen: 1,
		},
		{
			name: "partial page",
			pages: map[int][]storage.ClipboardItem{
				5: {{ID: 6, ClipText: "partial-1"}, {ID: 7, ClipText: "partial-2"}},
			},
			total: 7, offset: 0, limit: 5, signals: 1, wantLen: 3, initLen: 1,
		},
		{
			name:  "offset already at end",
			pages: map[int][]storage.ClipboardItem{},
			total: 10, offset: 10, limit: 5, signals: 1, wantLen: 0, initLen: 0,
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

			if tt.wantLen == tt.initLen {
				assertNoRead(t, reader.readCalls)
			} else {
				waitForReads(t, reader.readCalls, tt.signals)
			}

			if n := f.historyLen(); n != tt.wantLen {
				t.Errorf("expected history len %d, got %d", tt.wantLen, n)
			}
		})
	}
}
