package dnsproxy_test

import (
	"context"
	"net"
	"testing"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bavix/outway/internal/config"
	"github.com/bavix/outway/internal/dnsproxy"
)

// MockResolver is a mock implementation of Resolver for testing.
type MockResolver struct {
	resolveFunc func(ctx context.Context, q *dns.Msg) (*dns.Msg, string, error)
}

func (m *MockResolver) Resolve(ctx context.Context, q *dns.Msg) (*dns.Msg, string, error) {
	if m.resolveFunc != nil {
		return m.resolveFunc(ctx, q)
	}
	// Default: return NXDOMAIN
	msg := new(dns.Msg)
	msg.SetRcode(q, dns.RcodeNameError)

	return msg, "mock", nil
}

func TestHostsResolver_StaticHosts(t *testing.T) {
	t.Parallel()

	next := &MockResolver{}
	hosts := []config.HostOverride{
		{
			Pattern: "example.com",
			A:       []string{"1.2.3.4"},
			TTL:     300,
		},
	}

	resolver := &dnsproxy.HostsResolver{
		Next:  next,
		Hosts: hosts,
	}

	ctx := context.Background()
	q := new(dns.Msg)
	q.SetQuestion("example.com.", dns.TypeA)

	msg, src, err := resolver.Resolve(ctx, q)

	require.NoError(t, err)
	assert.Equal(t, "hosts", src)
	assert.NotNil(t, msg)
	assert.Equal(t, dns.RcodeSuccess, msg.Rcode)
	assert.Len(t, msg.Answer, 1)

	if len(msg.Answer) > 0 {
		if a, ok := msg.Answer[0].(*dns.A); ok {
			expectedIP := net.ParseIP("1.2.3.4")
			assert.True(t, expectedIP.Equal(a.A), "IP addresses should match: expected %s, got %s", expectedIP, a.A)
			assert.Equal(t, uint32(300), a.Hdr.Ttl)
		}
	}
}

func TestHostsResolver_DynamicHosts(t *testing.T) {
	t.Parallel()

	next := &MockResolver{}
	cfg := &config.Config{
		Cache: config.CacheConfig{
			Enabled:       true,
			MinTTLSeconds: 0,    // No minimum TTL restriction
			MaxTTLSeconds: 3600, // Allow up to 1 hour
		},
	}

	// Create a mock hosts manager
	hostsManager := &mockHostsManager{
		hosts: []config.HostOverride{
			{
				Pattern: "test.com",
				A:       []string{"5.6.7.8"},
				TTL:     60,
			},
		},
	}

	resolver := &dnsproxy.HostsResolver{
		Next:         next,
		HostsManager: hostsManager,
		Cfg:          cfg,
	}

	ctx := context.Background()
	q := new(dns.Msg)
	q.SetQuestion("test.com.", dns.TypeA)

	// First resolve - should use initial hosts
	msg1, src1, err1 := resolver.Resolve(ctx, q)
	require.NoError(t, err1)
	assert.Equal(t, "hosts", src1)
	assert.Len(t, msg1.Answer, 1)

	// Update hosts in manager
	hostsManager.UpdateHosts([]config.HostOverride{
		{
			Pattern: "test.com",
			A:       []string{"9.10.11.12"},
			TTL:     120,
		},
	})

	// Second resolve - should use updated hosts (dynamic)
	msg2, src2, err2 := resolver.Resolve(ctx, q)
	require.NoError(t, err2)
	assert.Equal(t, "hosts", src2)
	assert.Len(t, msg2.Answer, 1)

	if len(msg2.Answer) > 0 {
		if a, ok := msg2.Answer[0].(*dns.A); ok {
			expectedIP := net.ParseIP("9.10.11.12")
			assert.True(t, expectedIP.Equal(a.A), "IP addresses should match: expected %s, got %s", expectedIP, a.A)
			// TTL should be 120 (from host override) since it's within cache bounds
			assert.Equal(t, uint32(120), a.Hdr.Ttl, "TTL should match host override value")
		}
	}
}

func TestHostsResolver_FallbackToNext(t *testing.T) {
	t.Parallel()

	nextCalled := false
	next := &MockResolver{
		resolveFunc: func(ctx context.Context, q *dns.Msg) (*dns.Msg, string, error) {
			nextCalled = true
			msg := new(dns.Msg)
			msg.SetRcode(q, dns.RcodeNameError)

			return msg, "next", nil
		},
	}

	resolver := &dnsproxy.HostsResolver{
		Next:  next,
		Hosts: []config.HostOverride{},
	}

	ctx := context.Background()
	q := new(dns.Msg)
	q.SetQuestion("unknown.com.", dns.TypeA)

	msg, src, err := resolver.Resolve(ctx, q)

	require.NoError(t, err)
	assert.True(t, nextCalled)
	assert.Equal(t, "next", src)
	assert.NotNil(t, msg)
	assert.Equal(t, dns.RcodeNameError, msg.Rcode)
}

func TestHostsResolver_IPv6(t *testing.T) {
	t.Parallel()

	next := &MockResolver{}
	hosts := []config.HostOverride{
		{
			Pattern: "ipv6.example.com",
			AAAA:    []string{"2001:db8::1"},
			TTL:     300,
		},
	}

	resolver := &dnsproxy.HostsResolver{
		Next:  next,
		Hosts: hosts,
	}

	ctx := context.Background()
	q := new(dns.Msg)
	q.SetQuestion("ipv6.example.com.", dns.TypeAAAA)

	msg, src, err := resolver.Resolve(ctx, q)

	require.NoError(t, err)
	assert.Equal(t, "hosts", src)
	assert.Len(t, msg.Answer, 1)

	if len(msg.Answer) > 0 {
		if aaaa, ok := msg.Answer[0].(*dns.AAAA); ok {
			expectedIP := net.ParseIP("2001:db8::1")
			assert.True(t, expectedIP.Equal(aaaa.AAAA), "IPv6 addresses should match: expected %s, got %s", expectedIP, aaaa.AAAA)
		}
	}
}

func TestHostsResolver_WildcardPattern(t *testing.T) {
	t.Parallel()

	next := &MockResolver{}
	hosts := []config.HostOverride{
		{
			Pattern: "*.example.com",
			A:       []string{"1.1.1.1"},
		},
	}

	resolver := &dnsproxy.HostsResolver{
		Next:  next,
		Hosts: hosts,
	}

	ctx := context.Background()
	q := new(dns.Msg)
	q.SetQuestion("sub.example.com.", dns.TypeA)

	msg, src, err := resolver.Resolve(ctx, q)

	require.NoError(t, err)
	assert.Equal(t, "hosts", src)
	assert.Len(t, msg.Answer, 1)
}

// mockHostsManager is a simple mock implementation of HostsManager for testing.
type mockHostsManager struct {
	hosts []config.HostOverride
}

func (m *mockHostsManager) GetHosts() []config.HostOverride {
	return m.hosts
}

func (m *mockHostsManager) SetHosts(hosts []config.HostOverride) error {
	m.hosts = hosts

	return nil
}

func (m *mockHostsManager) CreateHostsResolver(next dnsproxy.Resolver, cfg *config.Config) *dnsproxy.HostsResolver {
	return &dnsproxy.HostsResolver{
		Next:         next,
		HostsManager: m,
		Cfg:          cfg,
	}
}

func (m *mockHostsManager) UpdateHostsInPlace(hosts []config.HostOverride) error {
	m.hosts = hosts

	return nil
}

func (m *mockHostsManager) UpdateHosts(hosts []config.HostOverride) {
	m.hosts = make([]config.HostOverride, len(hosts))
	copy(m.hosts, hosts)
}
