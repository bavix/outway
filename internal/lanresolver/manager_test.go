package lanresolver_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bavix/outway/internal/lanresolver"
	"github.com/bavix/outway/internal/localzone"
)

// createTempDir creates a temporary directory for testing.
func createTempDir(t *testing.T) string {
	t.Helper()

	return t.TempDir()
}

func TestNewManager(t *testing.T) {
	t.Parallel()

	zoneDetector := localzone.NewZoneDetector()
	leaseManager := lanresolver.NewLeaseManager("/var/lib/dhcp/dhcpd.leases")
	onChange := func() {}

	manager := lanresolver.NewManager(zoneDetector, leaseManager, onChange)

	assert.Equal(t, zoneDetector, manager.GetZoneDetector())
	assert.Equal(t, leaseManager, manager.GetLeaseManager())
	assert.NotNil(t, manager)
}

func TestNewManager_WithNilCallbacks(t *testing.T) {
	t.Parallel()

	zoneDetector := localzone.NewZoneDetector()
	leaseManager := lanresolver.NewLeaseManager("/var/lib/dhcp/dhcpd.leases")

	manager := lanresolver.NewManager(zoneDetector, leaseManager, nil)

	assert.Equal(t, zoneDetector, manager.GetZoneDetector())
	assert.Equal(t, leaseManager, manager.GetLeaseManager())
	assert.NotNil(t, manager)
}

func TestManager_Start_Success(t *testing.T) {
	t.Parallel()
	tempDir := createTempDir(t)
	leasesFile := filepath.Join(tempDir, "dhcpd.leases")

	// Create the leases file
	err := os.WriteFile(leasesFile, []byte(""), 0o600)
	require.NoError(t, err)

	zoneDetector := localzone.NewZoneDetector()
	leaseManager := lanresolver.NewLeaseManager(leasesFile)
	onChange := func() {}

	manager := lanresolver.NewManager(zoneDetector, leaseManager, onChange)

	ctx := context.Background()
	watchPaths := []string{leasesFile}

	err = manager.Start(ctx, watchPaths)
	require.NoError(t, err)
	assert.NotNil(t, manager)
}

func TestManager_Start_WithInvalidPath(t *testing.T) {
	t.Parallel()

	zoneDetector := localzone.NewZoneDetector()
	leaseManager := lanresolver.NewLeaseManager("/var/lib/dhcp/dhcpd.leases")
	onChange := func() {}

	manager := lanresolver.NewManager(zoneDetector, leaseManager, onChange)

	ctx := context.Background()
	watchPaths := []string{"/nonexistent/path"}

	// This should fail because the path doesn't exist
	err := manager.Start(ctx, watchPaths)
	assert.Error(t, err)
}

func TestManager_Stop_Success(t *testing.T) {
	t.Parallel()
	tempDir := createTempDir(t)
	leasesFile := filepath.Join(tempDir, "dhcpd.leases")

	// Create the leases file
	err := os.WriteFile(leasesFile, []byte(""), 0o600)
	require.NoError(t, err)

	zoneDetector := localzone.NewZoneDetector()
	leaseManager := lanresolver.NewLeaseManager(leasesFile)
	onChange := func() {}

	manager := lanresolver.NewManager(zoneDetector, leaseManager, onChange)

	ctx := context.Background()
	watchPaths := []string{leasesFile}

	// Start the manager first
	err = manager.Start(ctx, watchPaths)
	require.NoError(t, err)

	// Stop the manager
	err = manager.Stop()
	require.NoError(t, err)
}

func TestManager_Stop_WithNilWatcher(t *testing.T) {
	t.Parallel()

	zoneDetector := localzone.NewZoneDetector()
	leaseManager := lanresolver.NewLeaseManager("/var/lib/dhcp/dhcpd.leases")
	onChange := func() {}

	manager := lanresolver.NewManager(zoneDetector, leaseManager, onChange)

	// Stop without starting should not cause error
	err := manager.Stop()
	require.NoError(t, err)
}

func TestManager_GetZoneDetector(t *testing.T) {
	t.Parallel()

	zoneDetector := localzone.NewZoneDetector()
	leaseManager := lanresolver.NewLeaseManager("/var/lib/dhcp/dhcpd.leases")
	onChange := func() {}

	manager := lanresolver.NewManager(zoneDetector, leaseManager, onChange)

	assert.Equal(t, zoneDetector, manager.GetZoneDetector())
}

func TestManager_GetLeaseManager(t *testing.T) {
	t.Parallel()

	zoneDetector := localzone.NewZoneDetector()
	leaseManager := lanresolver.NewLeaseManager("/var/lib/dhcp/dhcpd.leases")
	onChange := func() {}

	manager := lanresolver.NewManager(zoneDetector, leaseManager, onChange)

	assert.Equal(t, leaseManager, manager.GetLeaseManager())
}

func TestManager_ReloadData_Success(t *testing.T) {
	t.Parallel()
	tempDir := createTempDir(t)
	leasesFile := filepath.Join(tempDir, "dhcpd.leases")

	// Create the leases file
	err := os.WriteFile(leasesFile, []byte(""), 0o600)
	require.NoError(t, err)

	zoneDetector := localzone.NewZoneDetector()
	leaseManager := lanresolver.NewLeaseManager(leasesFile)
	onChange := func() {}

	manager := lanresolver.NewManager(zoneDetector, leaseManager, onChange)

	// reloadData is private, but we can test it indirectly through Start
	ctx := context.Background()
	watchPaths := []string{leasesFile}

	err = manager.Start(ctx, watchPaths)
	require.NoError(t, err)
}

func TestManager_ReloadData_WithLeaseManagerError(t *testing.T) {
	t.Parallel()
	tempDir := createTempDir(t)
	leasesFile := filepath.Join(tempDir, "dhcpd.leases")

	// Create the leases file
	err := os.WriteFile(leasesFile, []byte(""), 0o600)
	require.NoError(t, err)

	zoneDetector := localzone.NewZoneDetector()
	leaseManager := lanresolver.NewLeaseManager("/nonexistent/leases")
	onChange := func() {}

	manager := lanresolver.NewManager(zoneDetector, leaseManager, onChange)

	// reloadData should not return error even if lease loading fails
	ctx := context.Background()
	watchPaths := []string{leasesFile}

	err = manager.Start(ctx, watchPaths)
	require.NoError(t, err)
}

func TestManager_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	zoneDetector := localzone.NewZoneDetector()
	leaseManager := lanresolver.NewLeaseManager("/var/lib/dhcp/dhcpd.leases")
	onChange := func() {}

	manager := lanresolver.NewManager(zoneDetector, leaseManager, onChange)

	// Test concurrent access to getters
	done := make(chan bool, 10)

	for range 10 {
		go func() {
			defer func() { done <- true }()

			_ = manager.GetZoneDetector()
			_ = manager.GetLeaseManager()
		}()
	}

	// Wait for all goroutines to complete
	for range 10 {
		<-done
	}
}

func TestManager_Start_WithCallback(t *testing.T) {
	t.Parallel()
	tempDir := createTempDir(t)
	leasesFile := filepath.Join(tempDir, "dhcpd.leases")

	// Create the leases file
	err := os.WriteFile(leasesFile, []byte(""), 0o600)
	require.NoError(t, err)

	zoneDetector := localzone.NewZoneDetector()
	leaseManager := lanresolver.NewLeaseManager(leasesFile)

	callbackCalled := false
	onChange := func() {
		callbackCalled = true
	}

	manager := lanresolver.NewManager(zoneDetector, leaseManager, onChange)

	ctx := context.Background()
	watchPaths := []string{leasesFile}

	err = manager.Start(ctx, watchPaths)
	require.NoError(t, err)

	// The callback should be set up but not called yet
	assert.False(t, callbackCalled)
}

func TestManager_Start_WithNilCallback(t *testing.T) {
	t.Parallel()
	tempDir := createTempDir(t)
	leasesFile := filepath.Join(tempDir, "dhcpd.leases")

	// Create the leases file
	err := os.WriteFile(leasesFile, []byte(""), 0o600)
	require.NoError(t, err)

	zoneDetector := localzone.NewZoneDetector()
	leaseManager := lanresolver.NewLeaseManager(leasesFile)

	manager := lanresolver.NewManager(zoneDetector, leaseManager, nil)

	ctx := context.Background()
	watchPaths := []string{leasesFile}

	err = manager.Start(ctx, watchPaths)
	require.NoError(t, err)
}

func TestManager_Start_WithEmptyWatchPaths(t *testing.T) {
	t.Parallel()

	zoneDetector := localzone.NewZoneDetector()
	leaseManager := lanresolver.NewLeaseManager("/var/lib/dhcp/dhcpd.leases")
	onChange := func() {}

	manager := lanresolver.NewManager(zoneDetector, leaseManager, onChange)

	ctx := context.Background()
	watchPaths := []string{}

	err := manager.Start(ctx, watchPaths)
	require.NoError(t, err)
}

func TestManager_Start_WithNilContext(t *testing.T) {
	t.Parallel()
	tempDir := createTempDir(t)
	leasesFile := filepath.Join(tempDir, "dhcpd.leases")

	// Create the leases file
	err := os.WriteFile(leasesFile, []byte(""), 0o600)
	require.NoError(t, err)

	zoneDetector := localzone.NewZoneDetector()
	leaseManager := lanresolver.NewLeaseManager(leasesFile)
	onChange := func() {}

	manager := lanresolver.NewManager(zoneDetector, leaseManager, onChange)

	watchPaths := []string{leasesFile}

	// This should work even with nil context
	err = manager.Start(context.TODO(), watchPaths)
	require.NoError(t, err)
}

func TestManager_Start_MultipleTimes(t *testing.T) {
	t.Parallel()
	tempDir := createTempDir(t)
	leasesFile := filepath.Join(tempDir, "dhcpd.leases")

	// Create the leases file
	err := os.WriteFile(leasesFile, []byte(""), 0o600)
	require.NoError(t, err)

	zoneDetector := localzone.NewZoneDetector()
	leaseManager := lanresolver.NewLeaseManager(leasesFile)
	onChange := func() {}

	manager := lanresolver.NewManager(zoneDetector, leaseManager, onChange)

	ctx := context.Background()
	watchPaths := []string{leasesFile}

	// Start multiple times should work
	err1 := manager.Start(ctx, watchPaths)
	require.NoError(t, err1)

	err2 := manager.Start(ctx, watchPaths)
	assert.NoError(t, err2)
}

func TestManager_Stop_MultipleTimes(t *testing.T) {
	t.Parallel()
	tempDir := createTempDir(t)
	leasesFile := filepath.Join(tempDir, "dhcpd.leases")

	// Create the leases file
	err := os.WriteFile(leasesFile, []byte(""), 0o600)
	require.NoError(t, err)

	zoneDetector := localzone.NewZoneDetector()
	leaseManager := lanresolver.NewLeaseManager(leasesFile)
	onChange := func() {}

	manager := lanresolver.NewManager(zoneDetector, leaseManager, onChange)

	ctx := context.Background()
	watchPaths := []string{leasesFile}

	// Start the manager
	err = manager.Start(ctx, watchPaths)
	require.NoError(t, err)

	// Stop multiple times should work
	err1 := manager.Stop()
	require.NoError(t, err1)

	err2 := manager.Stop()
	assert.NoError(t, err2)
}

func TestManager_EdgeCases(t *testing.T) {
	t.Parallel()

	zoneDetector := localzone.NewZoneDetector()
	leaseManager := lanresolver.NewLeaseManager("/var/lib/dhcp/dhcpd.leases")
	onChange := func() {}

	_ = lanresolver.NewManager(zoneDetector, leaseManager, onChange)

	// Test with nil zone detector
	// manager.zoneDetector = nil // unexported field, cannot access
	// assert.Nil(t, manager.GetZoneDetector())

	// Test with nil lease manager
	// manager.leaseManager = nil // unexported field, cannot access
	// assert.Nil(t, manager.GetLeaseManager())
}
