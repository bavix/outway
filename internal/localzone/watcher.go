package localzone

import (
	"context"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher monitors files for changes and triggers callbacks.
type Watcher struct {
	watcher   *fsnotify.Watcher
	callbacks []func()
	mu        sync.RWMutex
	debounce  time.Duration
	timer     *time.Timer
}

// NewWatcher creates a new file watcher.
func NewWatcher(debounce time.Duration) (*Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Watcher{
		watcher:  watcher,
		debounce: debounce,
	}, nil
}

// AddCallback adds a callback function to be called when files change.
func (w *Watcher) AddCallback(callback func()) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.callbacks = append(w.callbacks, callback)
}

// WatchFile starts watching a file for changes.
func (w *Watcher) WatchFile(path string) error {
	// Watch the directory containing the file
	dir := filepath.Dir(path)

	return w.watcher.Add(dir)
}

// WatchFiles starts watching multiple files.
func (w *Watcher) WatchFiles(paths []string) error {
	for _, path := range paths {
		if err := w.WatchFile(path); err != nil {
			return err
		}
	}

	return nil
}

// Start starts the watcher in a goroutine.
func (w *Watcher) Start(ctx context.Context) {
	go w.run(ctx)
}

// run runs the main watcher loop.
//
//nolint:funcorder
func (w *Watcher) run(ctx context.Context) {
	defer func() { _ = w.watcher.Close() }()

	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			// Handle file events
			if w.shouldTrigger(event) {
				w.triggerCallbacks()
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			// Log error but continue
			if err != nil {
				// In a real implementation, you'd use proper logging
				// For now, we'll just continue
				_ = err // Suppress unused variable warning
			}

		case <-ctx.Done():
			return
		}
	}
}

// shouldTrigger determines if an event should trigger callbacks.
//
//nolint:funcorder
func (w *Watcher) shouldTrigger(event fsnotify.Event) bool {
	// We're interested in these events:
	// - Create: file was created
	// - Write: file was written to
	// - Rename: file was renamed (common when files are recreated)
	// - Remove: file was removed
	return event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Rename|fsnotify.Remove) != 0
}

// triggerCallbacks triggers all registered callbacks with debouncing.
//
//nolint:funcorder
func (w *Watcher) triggerCallbacks() {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Cancel previous timer if it exists
	if w.timer != nil {
		w.timer.Stop()
	}

	// Set new timer
	w.timer = time.AfterFunc(w.debounce, func() {
		w.mu.RLock()
		callbacks := make([]func(), len(w.callbacks))
		copy(callbacks, w.callbacks)
		w.mu.RUnlock()

		// Execute all callbacks
		for _, callback := range callbacks {
			callback()
		}
	})
}

// Close closes the watcher.
func (w *Watcher) Close() error {
	if w.timer != nil {
		w.timer.Stop()
	}

	return w.watcher.Close()
}
