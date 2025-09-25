package dnsproxy

import (
	"crypto/tls"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
)

const (
	dotTimeout    = 5 * time.Second
	dotSplitLimit = 2
	protocolDot   = "dot"
)

// DotStrategy implements DoT (DNS-over-TLS) strategy.
type DotStrategy struct{}

func (DotStrategy) Supports(t string) bool { return t == protocolDot }
func (DotStrategy) NewResolver(t, address string, deps StrategyDeps) Resolver { //nolint:ireturn
	host := address
	// Strip scheme if accidentally present
	if strings.HasPrefix(host, "tls://") || strings.HasPrefix(host, "dot://") {
		host = strings.SplitN(host, "://", dotSplitLimit)[1]
	}

	serverName := host
	if h, _, err := net.SplitHostPort(host); err == nil {
		serverName = h
	}

	client := &dns.Client{
		Net: "tcp-tls",
		TLSConfig: &tls.Config{
			ServerName: serverName,
			MinVersion: tls.VersionTLS13,
		},
		Timeout: dotTimeout,
	}

	return &UpstreamResolver{client: client, network: t, address: host}
}
