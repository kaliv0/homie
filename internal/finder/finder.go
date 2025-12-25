package finder

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/ktr0731/go-fuzzyfinder"

	"github.com/kaliv0/homie/internal/log"
	"github.com/kaliv0/homie/internal/storage"
)

var mu sync.RWMutex

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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	loadMore := handleLoadChannel(ctx, &history, db, offset, limit, total)
	idxs, err := findItemIdxs(&history, loadMore)
	if err != nil {
		return "", err
	}

	// return selected item (from preview window)
	if len(idxs) == 0 {
		return "", nil
	}

	out := make([]string, 0, len(idxs))
	for _, i := range idxs {
		out = append(out, history[i].ClipText)
	}
	return strings.Join(out, " "), nil
}

func handleLoadChannel(ctx context.Context, history *[]storage.ClipboardItem, db *storage.Repository, offset, limit, total int) chan struct{} {
	// signal more items needed -> triggered from fuzzyfinder.WithPreviewWindow
	loadMore := make(chan struct{}, 1)
	go func() {
		currentOffset := offset
		for {
			select {
			case _, ok := <-loadMore:
				if !ok {
					return
				}
				if currentOffset < total {
					currentOffset += limit
					page, err := db.Read(currentOffset, limit)
					if err != nil {
						log.Logger().Printf("failed to load more history items (offset=%d, limit=%d, total=%d): %v\n",
							currentOffset, limit, total, err)
						continue
					}
					if len(page) > 0 {
						mu.Lock()
						*history = append(*history, page...)
						mu.Unlock()
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return loadMore
}

func findItemIdxs(history *[]storage.ClipboardItem, loadMore chan struct{}) ([]int, error) {
	defer close(loadMore)
	idxs, err := fuzzyfinder.FindMulti(
		history,
		// itemFunc -> returns items in main history list
		func(i int) string {
			return (*history)[i].ClipText
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
			return (*history)[i].ClipText
		}),
		// reloads passed history slice automatically when items appended
		fuzzyfinder.WithHotReloadLock(mu.RLocker()),
	)
	if err != nil && !errors.Is(err, fuzzyfinder.ErrAbort) {
		return nil, err
	}
	return idxs, nil
}
