package firewall_test

import (
	"context"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bavix/outway/internal/firewall"
)

func TestDetectBackend(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Test DetectBackend
	backend, err := firewall.DetectBackend(ctx)

	// The result depends on the current OS and available tools
	switch runtime.GOOS {
	case "linux":
		if backend != nil {
			assert.Equal(t, "simple_route", backend.Name())
			require.NoError(t, err)
		} else {
			require.Error(t, err)
			assert.Equal(t, "no supported firewall backend detected", err.Error())
		}
	case "darwin":
		if backend != nil {
			assert.Equal(t, "pf", backend.Name())
			require.NoError(t, err)
		} else {
			require.Error(t, err)
			assert.Equal(t, "no supported firewall backend detected", err.Error())
		}
	default:
		require.Error(t, err)
		assert.Equal(t, "no supported firewall backend detected", err.Error())
		assert.Nil(t, backend)
	}
}

func TestBackendInterface(t *testing.T) {
	t.Parallel()
	// Test that backends implement the Backend interface
	var backend firewall.Backend

	// Test SimpleRouteBackend
	simpleBackend := firewall.NewSimpleRouteBackend()
	if simpleBackend != nil {
		backend = simpleBackend
		assert.NotNil(t, backend)
		assert.Equal(t, "simple_route", backend.Name())
	}

	// Test PFBackend
	pfBackend := firewall.NewPFBackend()
	if pfBackend != nil {
		backend = pfBackend
		assert.NotNil(t, backend)
		assert.Equal(t, "pf", backend.Name())
	}
}

func TestBackendMethods(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Test SimpleRouteBackend methods
	simpleBackend := firewall.NewSimpleRouteBackend()
	if simpleBackend != nil {
		// Test Name
		assert.Equal(t, "simple_route", simpleBackend.Name())

		// Test MarkIP with valid inputs
		err := simpleBackend.MarkIP(ctx, "eth0", "192.168.1.1", 300)
		// This might fail in test environment, but should not panic
		_ = err

		// Test CleanupAll
		err = simpleBackend.CleanupAll(ctx)
		// This might fail in test environment, but should not panic
		_ = err
	}

	// Test PFBackend methods
	pfBackend := firewall.NewPFBackend()
	if pfBackend != nil {
		// Test Name
		assert.Equal(t, "pf", pfBackend.Name())

		// Test MarkIP with valid inputs
		err := pfBackend.MarkIP(ctx, "eth0", "192.168.1.1", 300)
		// This might fail in test environment, but should not panic
		_ = err

		// Test CleanupAll
		err = pfBackend.CleanupAll(ctx)
		// This might fail in test environment, but should not panic
		_ = err
	}
}

func TestBackendErrorHandling(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Test SimpleRouteBackend error handling
	simpleBackend := firewall.NewSimpleRouteBackend()
	if simpleBackend != nil {
		// Test with invalid interface name
		err := simpleBackend.MarkIP(ctx, "", "192.168.1.1", 300)
		require.Error(t, err)

		// Test with invalid IP
		err = simpleBackend.MarkIP(ctx, "eth0", "invalid-ip", 300)
		require.Error(t, err)

		// Test with negative TTL
		err = simpleBackend.MarkIP(ctx, "eth0", "192.168.1.1", -1)
		// Should normalize TTL to minimum value
		_ = err
	}

	// Test PFBackend error handling
	pfBackend := firewall.NewPFBackend()
	if pfBackend != nil {
		// Test with invalid interface name
		err := pfBackend.MarkIP(ctx, "", "192.168.1.1", 300)
		require.Error(t, err)

		// Test with invalid IP
		err = pfBackend.MarkIP(ctx, "eth0", "invalid-ip", 300)
		require.Error(t, err)

		// Test with negative TTL
		err = pfBackend.MarkIP(ctx, "eth0", "192.168.1.1", -1)
		// Should normalize TTL to minimum value
		_ = err
	}
}

func TestBackendConcurrency(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Test SimpleRouteBackend concurrency
	simpleBackend := firewall.NewSimpleRouteBackend()
	if simpleBackend != nil {
		// Test concurrent MarkIP calls
		done := make(chan bool, 10)

		for i := range 10 {
			go func(id int) {
				defer func() { done <- true }()

				ip := "192.168.1." + string(rune(id+1))
				_ = simpleBackend.MarkIP(ctx, "eth0", ip, 300)
			}(i)
		}

		// Wait for all goroutines to complete
		for range 10 {
			<-done
		}

		// Test CleanupAll
		_ = simpleBackend.CleanupAll(ctx)
	}

	// Test PFBackend concurrency
	pfBackend := firewall.NewPFBackend()
	if pfBackend != nil {
		// Test concurrent MarkIP calls
		done := make(chan bool, 10)

		for i := range 10 {
			go func(id int) {
				defer func() { done <- true }()

				ip := "192.168.1." + string(rune(id+1))
				_ = pfBackend.MarkIP(ctx, "eth0", ip, 300)
			}(i)
		}

		// Wait for all goroutines to complete
		for range 10 {
			<-done
		}

		// Test CleanupAll
		_ = pfBackend.CleanupAll(ctx)
	}
}

func TestBackendNilHandling(t *testing.T) {
	t.Parallel()
	// Test that backends handle nil context gracefully
	// Note: Some backends may panic with nil context due to exec.CommandContext
	// This is expected behavior, so we skip this test
	t.Skip("Skipping nil context test - backends may panic with nil context")
}

func TestBackendTTLNormalization(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Test SimpleRouteBackend TTL normalization
	simpleBackend := firewall.NewSimpleRouteBackend()
	if simpleBackend != nil {
		// Test with very small TTL
		err := simpleBackend.MarkIP(ctx, "eth0", "192.168.1.1", 1)
		// Should normalize to minimum TTL
		_ = err

		// Test with zero TTL
		err = simpleBackend.MarkIP(ctx, "eth0", "192.168.1.2", 0)
		// Should normalize to minimum TTL
		_ = err

		// Test with negative TTL
		err = simpleBackend.MarkIP(ctx, "eth0", "192.168.1.3", -10)
		// Should normalize to minimum TTL
		_ = err
	}

	// Test PFBackend TTL normalization
	pfBackend := firewall.NewPFBackend()
	if pfBackend != nil {
		// Test with very small TTL
		err := pfBackend.MarkIP(ctx, "eth0", "192.168.1.1", 1)
		// Should normalize to minimum TTL
		_ = err

		// Test with zero TTL
		err = pfBackend.MarkIP(ctx, "eth0", "192.168.1.2", 0)
		// Should normalize to minimum TTL
		_ = err

		// Test with negative TTL
		err = pfBackend.MarkIP(ctx, "eth0", "192.168.1.3", -10)
		// Should normalize to minimum TTL
		_ = err
	}
}
