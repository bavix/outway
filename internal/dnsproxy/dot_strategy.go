package dnsproxy

import (
	"crypto/tls"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
)

// DoT (DNS-over-TLS) strategy
type DotStrategy struct{}

func (DotStrategy) Supports(t string) bool { return t == "dot" }
func (DotStrategy) NewResolver(t, address string, deps StrategyDeps) Resolver {
	host := address
	// Strip scheme if accidentally present
	if strings.HasPrefix(host, "tls://") || strings.HasPrefix(host, "dot://") {
		host = strings.SplitN(host, "://", 2)[1]
	}
	serverName := host
	if h, _, err := net.SplitHostPort(host); err == nil {
		serverName = h
	}
	client := &dns.Client{Net: "tcp-tls", TLSConfig: &tls.Config{ServerName: serverName, MinVersion: tls.VersionTLS13}, Timeout: 5 * time.Second}
	return &UpstreamResolver{client: client, network: t, address: host}
}
