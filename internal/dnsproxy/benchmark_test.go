package dnsproxy_test

import (
	"context"
	"testing"

	"github.com/miekg/dns"

	"github.com/bavix/outway/internal/config"
	"github.com/bavix/outway/internal/dnsproxy"
)

// BenchmarkHostsResolver_StaticHosts benchmarks static hosts resolution.
func BenchmarkHostsResolver_StaticHosts(b *testing.B) {
	hosts := []config.HostOverride{
		{
			Pattern: "example.com",
			A:       []string{"1.2.3.4"},
			TTL:     60,
		},
		{
			Pattern: "test.com",
			A:       []string{"5.6.7.8"},
			TTL:     60,
		},
		{
			Pattern: "*.wildcard.com",
			A:       []string{"9.10.11.12"},
			TTL:     60,
		},
	}

	next := &MockResolver{}
	resolver := &dnsproxy.HostsResolver{
		Next:  next,
		Hosts: hosts,
	}

	ctx := context.Background()
	q := new(dns.Msg)
	q.SetQuestion("example.com.", dns.TypeA)

	for b.Loop() {
		_, _, _ = resolver.Resolve(ctx, q)
	}
}

// BenchmarkHostsResolver_DynamicHosts benchmarks dynamic hosts resolution.
func BenchmarkHostsResolver_DynamicHosts(b *testing.B) {
	hostsManager := &mockHostsManager{
		hosts: []config.HostOverride{
			{
				Pattern: "example.com",
				A:       []string{"1.2.3.4"},
				TTL:     60,
			},
			{
				Pattern: "test.com",
				A:       []string{"5.6.7.8"},
				TTL:     60,
			},
		},
	}

	next := &MockResolver{}
	resolver := &dnsproxy.HostsResolver{
		Next:         next,
		HostsManager: hostsManager,
	}

	ctx := context.Background()
	q := new(dns.Msg)
	q.SetQuestion("example.com.", dns.TypeA)

	for b.Loop() {
		_, _, _ = resolver.Resolve(ctx, q)
	}
}

// BenchmarkHostsResolver_ManyHosts benchmarks resolution with many hosts.
func BenchmarkHostsResolver_ManyHosts(b *testing.B) {
	// Create 100 hosts
	hosts := make([]config.HostOverride, 100)
	for i := range 100 {
		hosts[i] = config.HostOverride{
			Pattern: "example" + string(rune('0'+i%10)) + ".com",
			A:       []string{"1.2.3.4"},
			TTL:     60,
		}
	}

	next := &MockResolver{}
	resolver := &dnsproxy.HostsResolver{
		Next:  next,
		Hosts: hosts,
	}

	ctx := context.Background()
	q := new(dns.Msg)
	q.SetQuestion("example5.com.", dns.TypeA)

	for b.Loop() {
		_, _, _ = resolver.Resolve(ctx, q)
	}
}

// BenchmarkCacheResolver_Resolve benchmarks cache resolver performance.
func BenchmarkCacheResolver_Resolve(b *testing.B) {
	next := &MockResolver{
		resolveFunc: func(ctx context.Context, q *dns.Msg) (*dns.Msg, string, error) {
			msg := new(dns.Msg)
			msg.SetReply(q)
			msg.Answer = append(msg.Answer, &dns.A{
				Hdr: dns.RR_Header{
					Name:   q.Question[0].Name,
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    60,
				},
				A: []byte{1, 2, 3, 4},
			})

			return msg, "upstream", nil
		},
	}

	cache := dnsproxy.NewCachedResolver(next, 1000, 60, 3600)

	ctx := context.Background()
	q := new(dns.Msg)
	q.SetQuestion("example.com.", dns.TypeA)

	// Warm up cache
	_, _, err := cache.Resolve(ctx, q)
	if err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		_, _, err := cache.Resolve(ctx, q)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkHistoryManager_AddEvent benchmarks history manager performance.
// Note: newHistoryManager is not exported, so we test through Proxy.
func BenchmarkHistoryManager_AddEvent(b *testing.B) {
	// Create proxy to get history manager
	cfg := &config.Config{
		History: config.HistoryConfig{
			MaxEntries: 1000,
		},
	}
	mockBackend := &MockFirewallBackend{}
	proxy := dnsproxy.New(cfg, mockBackend)
	// Access history through proxy (if available)
	_ = proxy

	// Note: History manager is internal, so we skip this benchmark
	// or test through Proxy interface if available
	b.Skip("History manager is not exported, test through Proxy")
}

// BenchmarkHistoryManager_GetHistoryPaginated benchmarks paginated history retrieval.
func BenchmarkHistoryManager_GetHistoryPaginated(b *testing.B) {
	// Note: History manager is internal, so we skip this benchmark
	b.Skip("History manager is not exported, test through Proxy interface")
}
