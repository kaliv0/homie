package finder

import (
	"errors"
	"strings"
	"sync"

	"github.com/ktr0731/go-fuzzyfinder"

	"github.com/kaliv0/homie/internal/log"
	"github.com/kaliv0/homie/internal/storage"
)

// HistoryReader provides paginated access to clipboard history.
type HistoryReader interface {
	Read(offset, limit int) ([]storage.ClipboardItem, error)
	Count() (int, error)
	Close() error
}

const prompt = "D'OH >> "

type session struct {
	mu      sync.RWMutex
	history []storage.ClipboardItem
}

// ListHistory loads clipboard history and presents a fuzzy finder.
func ListHistory(dbPath string, limit int) (string, error) {
	// load history
	db, err := storage.NewRepository(dbPath)
	if err != nil {
		return "", err
	}

	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			log.Logger().Println(closeErr)
		}
	}()

	// display & search
	offset := 0
	history, err := db.Read(offset, limit)
	if err != nil {
		return "", err
	}
	total, err := db.Count()
	if err != nil {
		return "", err
	}

	s := &session{history: history}

	// stop the pagination goroutine before db.Close() so an in-flight db.Read()
	// isn't interrupted by a closed connection.
	var (
		wg   sync.WaitGroup
		once sync.Once
	)
	loadMore := handleLoadChannel(s, db, offset, limit, total, &wg)
	stop := func() {
		once.Do(func() {
			close(loadMore)
			wg.Wait()
		})
	}
	defer stop()

	idxs, err := findItemIdxs(s, loadMore)
	if err != nil {
		return "", err
	}
	// return selected item (from preview window)
	if len(idxs) == 0 {
		return "", nil
	}
	stop()

	out := make([]string, 0, len(idxs))
	for _, i := range idxs {
		out = append(out, s.history[i].ClipText)
	}
	return strings.Join(out, " "), nil
}

func handleLoadChannel(s *session, db HistoryReader,
	offset, limit, total int, wg *sync.WaitGroup) chan struct{} {
	// signal more items needed -> triggered from fuzzyfinder.WithPreviewWindow
	loadMore := make(chan struct{}, 1)
	wg.Go(func() {
		loadedOffset := offset
		for range loadMore {
			candidateOffset := loadedOffset + limit
			if candidateOffset >= total {
				continue
			}
			loadedOffset = candidateOffset
			page, err := db.Read(loadedOffset, limit)
			if err != nil {
				log.Logger().Printf("failed to load more history items (offset=%d, limit=%d, total=%d): %v\n",
					loadedOffset, limit, total, err)
				continue
			}
			if len(page) > 0 {
				s.mu.Lock()
				s.history = append(s.history, page...)
				s.mu.Unlock()
			}
		}
	})
	return loadMore
}

func findItemIdxs(s *session, loadMore chan struct{}) ([]int, error) {
	idxs, err := fuzzyfinder.FindMulti(
		&s.history,
		// itemFunc -> returns items in main history list
		func(i int) string {
			return s.history[i].ClipText
		},
		// opts for fuzzy-finder window
		fuzzyfinder.WithPreviewWindow(func(i, width, height int) string {
			if i == -1 {
				// no item found while searching
				select {
				case loadMore <- struct{}{}:
				default:
				}
				return ""
			}
			// return string to display in previewWindow
			return s.history[i].ClipText
		}),
		// reloads passed history slice automatically when items appended
		fuzzyfinder.WithHotReloadLock(s.mu.RLocker()),
		fuzzyfinder.WithPromptString(prompt),
	)
	if err != nil && !errors.Is(err, fuzzyfinder.ErrAbort) {
		return nil, err
	}
	return idxs, nil
}
