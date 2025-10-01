package localzone

import (
	"context"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog"
)

const (
	debounceDelay = 200 * time.Millisecond
)

// Watcher watches config files for changes and triggers callbacks.
type Watcher struct {
	fsWatcher  *fsnotify.Watcher
	callbacks  []func()
	mu         sync.RWMutex
	debounce   map[string]*time.Timer
	debounceMu sync.Mutex
}

// NewWatcher creates a new file watcher.
func NewWatcher() (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Watcher{
		fsWatcher: fsw,
		debounce:  make(map[string]*time.Timer),
	}, nil
}

// Watch starts watching the given files and calls callbacks on changes.
func (w *Watcher) Watch(ctx context.Context, files []string) {
	logger := zerolog.Ctx(ctx)

	// Add files to watcher (ignore errors if files don't exist)
	for _, f := range files {
		if err := w.fsWatcher.Add(f); err != nil {
			logger.Warn().Err(err).Str("file", f).Msg("failed to watch file")
		}
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				_ = w.fsWatcher.Close()
				return

			case event, ok := <-w.fsWatcher.Events:
				if !ok {
					return
				}

				// Handle Create, Write, Rename, Remove events
				if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) ||
					event.Has(fsnotify.Rename) || event.Has(fsnotify.Remove) {
					logger.Debug().
						Str("file", event.Name).
						Str("op", event.Op.String()).
						Msg("file change detected")

					// Debounce: only trigger callback after no changes for debounceDelay
					w.debounceCallback(event.Name)

					// Re-add file if it was removed/renamed (common with atomic writes)
					if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
						time.Sleep(50 * time.Millisecond) // Wait for file recreation
						_ = w.fsWatcher.Add(event.Name)
					}
				}

			case err, ok := <-w.fsWatcher.Errors:
				if !ok {
					return
				}
				logger.Warn().Err(err).Msg("fsnotify error")
			}
		}
	}()
}

// debounceCallback ensures callbacks are only triggered after debounceDelay of inactivity.
func (w *Watcher) debounceCallback(file string) {
	w.debounceMu.Lock()
	defer w.debounceMu.Unlock()

	// Cancel existing timer for this file
	if timer, exists := w.debounce[file]; exists {
		timer.Stop()
	}

	// Create new timer
	w.debounce[file] = time.AfterFunc(debounceDelay, func() {
		w.triggerCallbacks()

		// Clean up timer
		w.debounceMu.Lock()
		delete(w.debounce, file)
		w.debounceMu.Unlock()
	})
}

// triggerCallbacks calls all registered callbacks.
func (w *Watcher) triggerCallbacks() {
	w.mu.RLock()
	callbacks := w.callbacks
	w.mu.RUnlock()

	for _, cb := range callbacks {
		cb()
	}
}

// OnChange registers a callback to be called when files change.
func (w *Watcher) OnChange(callback func()) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.callbacks = append(w.callbacks, callback)
}

// Close stops the watcher.
func (w *Watcher) Close() error {
	return w.fsWatcher.Close()
}
