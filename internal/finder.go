package internal

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/ktr0731/go-fuzzyfinder"
)

var mu sync.RWMutex

// ListHistory loads clipboard history, presents a fuzzy finder, and returns selected text.
func ListHistory(dbPath string, limit int) string {
	// load history
	db := NewRepository(dbPath, false)
	defer db.Close()

	// display & search
	offset := 0
	history := db.Read(offset, limit)
	total := db.Count()
	loadMore := handleLoadChannel(&history, db, offset, limit, total)
	idxs := findItemIdxs(&history, loadMore)

	// return selected item (from preview window)
	if len(idxs) == 0 {
		return ""
	}

	var out []string
	for _, i := range idxs {
		out = append(out, history[i].ClipText)
	}
	return strings.Join(out, " ")
}

func handleLoadChannel(history *[]ClipboardItem, db *Repository, offset, limit, total int) chan struct{} {
	// signal more items needed -> triggered from fuzzyfinder.WithPreviewWindow
	loadMore := make(chan struct{}, 1)
	go func(history *[]ClipboardItem) {
		for range loadMore {
			if offset < total {
				offset += limit
				page := db.Read(offset, limit)
				if len(page) > 0 {
					mu.Lock()
					*history = append(*history, page...)
					mu.Unlock()
				}
			}
		}
	}(history)
	return loadMore
}

func findItemIdxs(history *[]ClipboardItem, loadMore chan struct{}) []int {
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
				loadMore <- struct{}{}
				return ""
			}
			// return string to display in previewWindow
			return fmt.Sprint((*history)[i].ClipText)
		}),
		// reloads passed history slice automatically when items appended
		fuzzyfinder.WithHotReloadLock(mu.RLocker()),
	)
	if err != nil && !errors.Is(err, fuzzyfinder.ErrAbort) {
		Logger.Fatal(err)
	}
	return idxs
}
