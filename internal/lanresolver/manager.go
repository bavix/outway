package lanresolver

import (
	"context"
	"sync"
	"time"

	"github.com/bavix/outway/internal/localzone"
)

// Manager manages local zones and DHCP leases with automatic file watching.
type Manager struct {
	zoneDetector *localzone.ZoneDetector
	leaseManager *LeaseManager
	watcher      *localzone.Watcher
	mu           sync.RWMutex
	onChange     func()
}

// NewManager creates a new local zones manager.
func NewManager(
	zoneDetector *localzone.ZoneDetector,
	leaseManager *LeaseManager,
	onChange func(),
) *Manager {
	return &Manager{
		zoneDetector: zoneDetector,
		leaseManager: leaseManager,
		onChange:     onChange,
	}
}

// Start starts the file watcher and loads initial data.
func (m *Manager) Start(ctx context.Context, watchPaths []string) error {
	// Create watcher with 200ms debounce
	const debounceMs = 200

	watcher, err := localzone.NewWatcher(debounceMs * time.Millisecond)
	if err != nil {
		return err
	}

	m.mu.Lock()
	m.watcher = watcher
	m.mu.Unlock()

	// Add callback for file changes
	watcher.AddCallback(func() {
		_ = m.reloadData()

		if m.onChange != nil {
			m.onChange()
		}
	})

	// Watch files
	if err := watcher.WatchFiles(watchPaths); err != nil {
		return err
	}

	// Load initial data
	_ = m.reloadData()

	// Start watcher
	watcher.Start(ctx)

	return nil
}

// Stop stops the file watcher.
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.watcher != nil {
		return m.watcher.Close()
	}

	return nil
}

// reloadData reloads zones and leases from files.
//
//nolint:funcorder,unparam
func (m *Manager) reloadData() error {
	// Reload zones
	if _, err := m.zoneDetector.DetectZones(); err != nil {
		// Log error but continue
		_ = err
	}

	// Reload leases
	if err := m.leaseManager.LoadLeases(); err != nil {
		// Log error but continue
		_ = err
	}

	return nil
}

// GetZoneDetector returns the zone detector.
func (m *Manager) GetZoneDetector() *localzone.ZoneDetector {
	return m.zoneDetector
}

// GetLeaseManager returns the lease manager.
func (m *Manager) GetLeaseManager() *LeaseManager {
	return m.leaseManager
}
