package dnsproxy_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bavix/outway/internal/config"
	"github.com/bavix/outway/internal/dnsproxy"
)

// TestMatchDomainPattern is already covered in proxy_test.go

func TestRuleStore(t *testing.T) {
	t.Parallel()
	// Test NewRuleStore
	rules := []config.Rule{
		{Pattern: "*.example.com", Via: "eth0"},
		{Pattern: "test.com", Via: "wlan0"},
	}
	store := dnsproxy.NewRuleStore(rules)
	require.NotNil(t, store)

	// Test List
	listedRules := store.List()
	assert.Len(t, listedRules, 2)
	assert.Equal(t, "*.example.com", listedRules[0].Pattern)
	assert.Equal(t, "test.com", listedRules[1].Pattern)

	// Test Upsert - update existing
	newRule := config.Rule{Pattern: "*.example.com", Via: "eth1"}
	store.Upsert(newRule)
	updatedRules := store.List()
	assert.Len(t, updatedRules, 2)
	assert.Equal(t, "eth1", updatedRules[0].Via)

	// Test Upsert - add new
	newRule2 := config.Rule{Pattern: "new.com", Via: "eth2"}
	store.Upsert(newRule2)
	finalRules := store.List()
	assert.Len(t, finalRules, 3)
}

// TestExtractClientIP is already covered in proxy_test.go

// TestProtocolConstants, TestDefaultConstants, and TestErrorConstants are already covered in proxy_test.go

func TestQueryEvent(t *testing.T) {
	t.Parallel()
	// Test QueryEvent struct
	event := dnsproxy.QueryEvent{
		Name:     "example.com",
		QType:    1, // A record
		Upstream: "8.8.8.8:53",
		Duration: "10ms",
		Status:   "success",
		Time:     time.Now(),
		ClientIP: "192.168.1.1",
	}

	assert.Equal(t, "example.com", event.Name)
	assert.Equal(t, uint16(1), event.QType)
	assert.Equal(t, "8.8.8.8:53", event.Upstream)
	assert.Equal(t, "10ms", event.Duration)
	assert.Equal(t, "success", event.Status)
	assert.Equal(t, "192.168.1.1", event.ClientIP)
	assert.False(t, event.Time.IsZero())
}

func TestRuleStoreConcurrency(t *testing.T) {
	t.Parallel()

	store := dnsproxy.NewRuleStore([]config.Rule{})

	// Test concurrent access
	var wg sync.WaitGroup

	numGoroutines := 10
	numOperations := 100

	for i := range numGoroutines {
		wg.Add(1)

		go func(id int) {
			defer wg.Done()

			for j := range numOperations {
				rule := config.Rule{
					Pattern: fmt.Sprintf("pattern%d.%d", id, j),
					Via:     fmt.Sprintf("eth%d", id),
				}
				store.Upsert(rule)
				store.List()
			}
		}(i)
	}

	wg.Wait()

	// Verify final state
	rules := store.List()
	assert.Len(t, rules, numGoroutines*numOperations)
}

// TestMatchDomainPatternEdgeCases is already covered in proxy_test.go

// mockResponseWriter is defined in managers_internal_test.go
