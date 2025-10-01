package lanresolver

import (
	"context"
	"strings"
	"sync"

	"github.com/miekg/dns"

	"github.com/bavix/outway/internal/localzone"
)

const (
	// Default TTL for local DNS records.
	defaultTTL = 60
	// LAN resolver identifier.
	lanResolverID = "lan"
)

// LocalResolver defines the interface for DNS resolvers in lanresolver package.
type LocalResolver interface {
	Resolve(ctx context.Context, q *dns.Msg) (*dns.Msg, string, error)
}

// LANResolver resolves DNS queries for local zones using DHCP leases.
type LANResolver struct {
	Next interface {
		Resolve(ctx context.Context, q *dns.Msg) (*dns.Msg, string, error)
	}
	ZoneDetector *localzone.ZoneDetector
	LeaseManager *LeaseManager
	mu           sync.RWMutex
}

// NewLANResolver creates a new LAN resolver.
func NewLANResolver(next interface {
	Resolve(ctx context.Context, q *dns.Msg) (*dns.Msg, string, error)
}, zoneDetector *localzone.ZoneDetector, leaseManager *LeaseManager,
) *LANResolver {
	return &LANResolver{
		Next:         next,
		ZoneDetector: zoneDetector,
		LeaseManager: leaseManager,
	}
}

// Resolve resolves DNS queries for local zones.
func (lr *LANResolver) Resolve(ctx context.Context, q *dns.Msg) (*dns.Msg, string, error) {
	if q == nil || len(q.Question) == 0 {
		if lr.Next != nil {
			return lr.Next.Resolve(ctx, q)
		}
		// Return error response if no Next resolver
		response := new(dns.Msg)
		if q != nil {
			response.SetReply(q)
		}
		response.Rcode = dns.RcodeServerFailure
		return response, lanResolverID, nil
	}

	question := q.Question[0]
	// Normalize domain: lowercase and remove trailing dot
	domain := strings.ToLower(strings.TrimSuffix(question.Name, "."))
	qtype := question.Qtype

	// Check if this is a local zone query
	isLocal, zone := lr.ZoneDetector.IsLocalZone(domain)
	
	// If not a local zone, try appending detected zones (e.g., .lan)
	// This handles queries like "ipad" -> "ipad.lan"
	if !isLocal {
		zones, _ := lr.ZoneDetector.DetectZones()
		for _, z := range zones {
			testDomain := domain + "." + z
			isLocal, zone = lr.ZoneDetector.IsLocalZone(testDomain)
			if isLocal {
				domain = testDomain
				break
			}
		}
	}
	
	if !isLocal {
		// Not a local zone, pass to next resolver
		if lr.Next != nil {
			return lr.Next.Resolve(ctx, q)
		}
		// If no Next resolver, return NXDOMAIN
		response := new(dns.Msg)
		response.SetReply(q)
		response.Rcode = dns.RcodeNameError
		return response, lanResolverID, nil
	}

	// This is a local zone query - resolve from DHCP leases
	return lr.resolveLocalQuery(ctx, q, domain, zone, qtype)
}

// resolveLocalQuery resolves a query for a local zone.
//
//nolint:cyclop,funcorder,funlen,unparam
func (lr *LANResolver) resolveLocalQuery(ctx context.Context, q *dns.Msg, domain, zone string, qtype uint16) (*dns.Msg, string, error) {
	// Extract hostname from domain (remove zone suffix)
	hostname := domain
	if zone != "" && strings.HasSuffix(domain, "."+zone) {
		hostname = strings.TrimSuffix(domain, "."+zone)
	}

	// Try to resolve from DHCP leases
	// First try with full domain, then with just hostname
	aRecords, aaaaRecords := lr.LeaseManager.ResolveHostname(domain)
	if len(aRecords) == 0 && len(aaaaRecords) == 0 {
		aRecords, aaaaRecords = lr.LeaseManager.ResolveHostname(hostname)
	}

	// Create response
	response := new(dns.Msg)
	response.SetReply(q)
	response.Authoritative = true

	// If no records found, return NXDOMAIN
	if len(aRecords) == 0 && len(aaaaRecords) == 0 {
		response.Rcode = dns.RcodeNameError

		return response, lanResolverID, nil
	}

	// Add A records if requested
	if qtype == dns.TypeA || qtype == dns.TypeANY {
		for _, ip := range aRecords {
			if ipv4 := ip.To4(); ipv4 != nil {
				rr := &dns.A{
					Hdr: dns.RR_Header{
						Name:   q.Question[0].Name,
						Rrtype: dns.TypeA,
						Class:  dns.ClassINET,
						Ttl:    defaultTTL,
					},
					A: ipv4,
				}
				response.Answer = append(response.Answer, rr)
			}
		}
	}

	// Add AAAA records if requested
	if qtype == dns.TypeAAAA || qtype == dns.TypeANY {
		for _, ip := range aaaaRecords {
			if ip.To16() != nil && ip.To4() == nil {
				rr := &dns.AAAA{
					Hdr: dns.RR_Header{
						Name:   q.Question[0].Name,
						Rrtype: dns.TypeAAAA,
						Class:  dns.ClassINET,
						Ttl:    defaultTTL,
					},
					AAAA: ip,
				}
				response.Answer = append(response.Answer, rr)
			}
		}
	}

	// If we have answers, return them
	if len(response.Answer) > 0 {
		return response, lanResolverID, nil
	}

	// No matching record type, return NXDOMAIN
	response.Rcode = dns.RcodeNameError

	return response, "lan", nil
}

// UpdateZones updates the zone detector configuration.
func (lr *LANResolver) UpdateZones(manualZones []string) {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	lr.ZoneDetector.ManualZones = manualZones
}

// ReloadLeases reloads DHCP leases.
func (lr *LANResolver) ReloadLeases() error {
	return lr.LeaseManager.LoadLeases()
}

// GetZones returns detected local zones.
func (lr *LANResolver) GetZones() ([]string, error) {
	return lr.ZoneDetector.DetectZones()
}

// GetLeases returns all current leases.
func (lr *LANResolver) GetLeases() []*Lease {
	return lr.LeaseManager.GetAllLeases()
}

// TestResolve tests resolution for a specific hostname.
func (lr *LANResolver) TestResolve(ctx context.Context, hostname string) (*dns.Msg, error) {
	// Create a test query
	q := new(dns.Msg)
	q.SetQuestion(dns.Fqdn(hostname), dns.TypeA)
	q.RecursionDesired = true

	// Resolve the query
	response, _, err := lr.Resolve(ctx, q)

	return response, err
}

// IsLocalZone checks if a domain is a local zone.
func (lr *LANResolver) IsLocalZone(domain string) (bool, string) {
	return lr.ZoneDetector.IsLocalZone(domain)
}
