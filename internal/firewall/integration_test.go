package firewall_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/bavix/outway/internal/firewall"
)

func TestBackendInterfaceCompliance(t *testing.T) {
	t.Parallel()
	// Test that all backends implement the Backend interface

	// Test route backend
	routeBackend, err := firewall.NewRouteBackend()
	if err != nil {
		t.Fatalf("Failed to create route backend: %v", err)
	}

	var _ firewall.Backend = routeBackend

	// Test iptables backend
	iptablesBackend := firewall.NewIPTablesBackend()
	if iptablesBackend != nil {
		var _ firewall.Backend = iptablesBackend
	}

	// Test pf backend
	pfBackend := firewall.NewPFBackend()
	if pfBackend != nil {
		var _ firewall.Backend = pfBackend
	}
}

func TestTunnelInfoStructure(t *testing.T) {
	t.Parallel()
	// Test that TunnelInfo can be created and accessed
	info := firewall.TunnelInfo{
		Name:     "tun0",
		TableID:  30001,
		FwMark:   30001,
		Priority: 30001,
	}

	// Test field access
	if info.Name == "" {
		t.Error("TunnelInfo.Name should not be empty")
	}

	if info.TableID <= 0 {
		t.Error("TunnelInfo.TableID should be positive")
	}

	if info.FwMark <= 0 {
		t.Error("TunnelInfo.FwMark should be positive")
	}

	if info.Priority <= 0 {
		t.Error("TunnelInfo.Priority should be positive")
	}
}

func TestRouteBackendThreadSafety(t *testing.T) {
	t.Parallel()

	backend, err := firewall.NewRouteBackend()
	if err != nil {
		t.Fatalf("Failed to create route backend: %v", err)
	}

	// Test concurrent access through public methods
	done := make(chan bool, 10)

	for i := range 10 {
		go func(id int) {
			tunnel := fmt.Sprintf("tun%d", id)
			// Test concurrent initialization
			_, err := backend.InitializeTunnels(context.Background(), []string{tunnel})
			if err != nil {
				t.Logf("InitializeTunnels failed: %v", err)
			}

			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for range 10 {
		<-done
	}

	// Test that backend is still functional
	if backend.Name() != "route" {
		t.Errorf("Expected backend name to be 'route', got %s", backend.Name())
	}
}

func TestConstantsValidation(t *testing.T) {
	t.Parallel()
	// Test that constants are within expected ranges
	if firewall.RoutingTableBase < 1000 {
		t.Errorf("routingTableBase (%d) should be >= 1000 to avoid system tables", firewall.RoutingTableBase)
	}

	if firewall.RoutingTableMax <= firewall.RoutingTableBase {
		t.Errorf("routingTableMax (%d) should be > routingTableBase (%d)", firewall.RoutingTableMax, firewall.RoutingTableBase)
	}

	if firewall.RoutingTableMax-firewall.RoutingTableBase < 100 {
		t.Errorf("Table range should be at least 100, got %d", firewall.RoutingTableMax-firewall.RoutingTableBase)
	}

	// Test marker constants
	if firewall.MarkerIPPoolStart == "" {
		t.Error("MarkerIPPoolStart should not be empty")
	}

	if firewall.OutwayMarkerProto == "" {
		t.Error("outwayMarkerProto should not be empty")
	}

	if firewall.OutwayMarkerMetric <= 0 {
		t.Error("outwayMarkerMetric should be positive")
	}
}

func TestErrorHandling(t *testing.T) {
	t.Parallel()
	// Test that error variables are properly defined
	if firewall.ErrInvalidIface == nil {
		t.Error("ErrInvalidIface should not be nil")
	}

	if firewall.ErrInvalidIP == nil {
		t.Error("ErrInvalidIP should not be nil")
	}
}

func TestBackendDetection(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	backend, err := firewall.DetectBackend(ctx)
	if err != nil {
		t.Logf("Backend detection failed: %v", err)
		// This is expected on some systems
		return
	}

	if backend == nil {
		t.Error("Detected backend should not be nil")

		return
	}

	// Test that backend implements all required methods
	name := backend.Name()
	if name == "" {
		t.Error("Backend name should not be empty")
	}

	// Test that we can call methods without panic
	_ = backend.EnsurePolicy(ctx, "lo0")
	_ = backend.MarkIP(ctx, "lo0", "127.0.0.1", 30)
	_ = backend.CleanupAll(ctx)
	_, _ = backend.InitializeTunnels(ctx, []string{"lo0"})
	_ = backend.FlushRuntime(ctx)
	_, _ = backend.GetTunnelInfo(ctx, "lo0")
}
