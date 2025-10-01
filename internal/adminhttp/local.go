package adminhttp

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/render"
	"github.com/miekg/dns"

	"github.com/bavix/outway/internal/lanresolver"
)

// handleLocalZones returns the detected local zones.
// GET /api/v1/local/zones
func (s *Server) handleLocalZones(w http.ResponseWriter, r *http.Request) {
	resolver := s.getLANResolver()
	if resolver == nil {
		render.JSON(w, r, map[string]any{
			"zones": []string{},
		})
		return
	}

	zones := resolver.GetZones()
	render.JSON(w, r, map[string]any{
		"zones": zones,
	})
}

// handleLocalLeases returns the current DHCP leases.
// GET /api/v1/local/leases
func (s *Server) handleLocalLeases(w http.ResponseWriter, r *http.Request) {
	resolver := s.getLANResolver()
	if resolver == nil {
		render.JSON(w, r, map[string]any{
			"leases": []lanresolver.Lease{},
		})
		return
	}

	leases := resolver.GetLeases()
	render.JSON(w, r, map[string]any{
		"leases": leases,
	})
}

// handleLocalResolve resolves a hostname using the LAN resolver.
// GET /api/v1/local/resolve?name=host.lan
func (s *Server) handleLocalResolve(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		render.Status(r, defaultBadRequestStatus)
		render.JSON(w, r, map[string]string{"error": "name parameter required"})
		return
	}

	resolver := s.getLANResolver()
	if resolver == nil {
		render.Status(r, defaultInternalServerErrorStatus)
		render.JSON(w, r, map[string]string{"error": "LAN resolver not configured"})
		return
	}

	// Create DNS query
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(name), dns.TypeA)

	// Resolve using LAN resolver
	resp, src, err := resolver.Resolve(r.Context(), m)
	if err != nil {
		render.Status(r, defaultBadGatewayStatus)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	// Build response
	result := map[string]any{
		"name":     name,
		"source":   src,
		"rcode":    resp.Rcode,
		"rcodeStr": dns.RcodeToString[resp.Rcode],
		"answers":  len(resp.Answer),
		"records":  rrToStrings(resp.Answer),
	}

	// Check if NXDOMAIN
	if resp.Rcode == dns.RcodeNameError {
		result["status"] = "NXDOMAIN"
	} else if len(resp.Answer) > 0 {
		result["status"] = "OK"
	} else {
		result["status"] = "NODATA"
	}

	render.JSON(w, r, result)
}

// getLANResolver attempts to find the LAN resolver in the resolver chain.
func (s *Server) getLANResolver() *lanresolver.Resolver {
	resolverAny := s.proxy.ResolverActive()
	if resolverAny == nil {
		return nil
	}

	// Try to find LAN resolver in the chain
	return findLANResolver(resolverAny)
}

// findLANResolver recursively searches for a LAN resolver in the resolver chain.
func findLANResolver(resolver interface{}) *lanresolver.Resolver {
	if resolver == nil {
		return nil
	}

	// Direct match
	if lan, ok := resolver.(*lanresolver.Resolver); ok {
		return lan
	}

	// Check for Next field (common pattern in resolver chain)
	type hasNext interface {
		GetNext() interface{}
	}

	if hn, ok := resolver.(hasNext); ok {
		return findLANResolver(hn.GetNext())
	}

	return nil
}

// broadcastLocalUpdate sends local DNS updates to all WebSocket clients.
func (s *Server) broadcastLocalUpdate() {
	resolver := s.getLANResolver()
	if resolver == nil {
		return
	}

	s.broadcast(map[string]any{
		"type": "local_zones",
		"data": map[string]any{
			"zones":  resolver.GetZones(),
			"leases": resolver.GetLeases(),
		},
	})
}

// handleLocalWatch is the WebSocket endpoint for local DNS updates.
// GET /api/v1/local/watch (upgrade to WebSocket)
func (s *Server) handleLocalWatch(w http.ResponseWriter, r *http.Request) {
	// Upgrade to WebSocket using global upgrader
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	// Register connection
	s.wsMu.Lock()
	s.conns[conn] = struct{}{}
	s.wsMu.Unlock()

	// Send initial state
	resolver := s.getLANResolver()
	if resolver != nil {
		data := map[string]any{
			"type": "local_zones",
			"data": map[string]any{
				"zones":  resolver.GetZones(),
				"leases": resolver.GetLeases(),
			},
		}

		s.wsWriteMu.Lock()
		_ = conn.WriteJSON(data)
		s.wsWriteMu.Unlock()
	}

	// Keep connection alive (cleanup happens in broadcast goroutine)
	defer func() {
		s.wsMu.Lock()
		delete(s.conns, conn)
		s.wsMu.Unlock()
		_ = conn.Close()
	}()

	// Read messages (just to keep connection alive, we don't expect client messages)
	for {
		var msg json.RawMessage
		if err := conn.ReadJSON(&msg); err != nil {
			break
		}
	}
}
