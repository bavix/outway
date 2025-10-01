package adminhttp

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/miekg/dns"

	"github.com/bavix/outway/internal/lanresolver"
	"github.com/bavix/outway/internal/localzone"
)

// LocalZonesHandler handles local zones API endpoints.
type LocalZonesHandler struct {
	zoneDetector *localzone.ZoneDetector
	leaseManager *lanresolver.LeaseManager
	lanResolver  *lanresolver.LANResolver
	manager      *lanresolver.Manager     // Manager for file watching
	clients      map[*websocket.Conn]bool // Connected WebSocket clients
	clientsMu    sync.RWMutex             // Mutex for clients map
}

// NewLocalZonesHandler creates a new local zones handler.
func NewLocalZonesHandler(
	zoneDetector *localzone.ZoneDetector,
	leaseManager *lanresolver.LeaseManager,
	lanResolver *lanresolver.LANResolver,
	manager *lanresolver.Manager,
) *LocalZonesHandler {
	return &LocalZonesHandler{
		zoneDetector: zoneDetector,
		leaseManager: leaseManager,
		lanResolver:  lanResolver,
		manager:      manager,
		clients:      make(map[*websocket.Conn]bool),
	}
}

// RegisterRoutes registers local zones API routes.
func (h *LocalZonesHandler) RegisterRoutes(mux *mux.Router) {
	api := mux.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/local/zones", h.handleZones).Methods("GET")
	api.HandleFunc("/local/leases", h.handleLeases).Methods("GET")
	api.HandleFunc("/local/resolve", h.handleResolve).Methods("GET")
	api.HandleFunc("/local/watch", h.handleWatch)
}

// handleZones returns detected local zones.
//
//nolint:funcorder
func (h *LocalZonesHandler) handleZones(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

		return
	}

	zones, err := h.zoneDetector.DetectZones()
	if err != nil {
		// If detection fails, return empty zones array instead of error
		zones = []string{}
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(map[string]any{
		"zones": zones,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}
}

// handleLeases returns current DHCP leases.
//
//nolint:funcorder
func (h *LocalZonesHandler) handleLeases(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

		return
	}

	leases := h.leaseManager.GetAllLeases()

	// Ensure leases is never nil
	if leases == nil {
		leases = []*lanresolver.Lease{}
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(map[string]any{
		"leases": leases,
		"count":  len(leases),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}
}

// handleResolve tests resolution for a specific hostname.
//
//nolint:funcorder
func (h *LocalZonesHandler) handleResolve(w http.ResponseWriter, r *http.Request) { //nolint:funlen
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

		return
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "name parameter is required", http.StatusBadRequest)

		return
	}

	// Test resolution
	response, err := h.lanResolver.TestResolve(r.Context(), name)
	if err != nil {
		// If resolution fails, return a basic response
		result := map[string]any{
			"hostname":       name,
			"is_local":       false,
			"zone":           "",
			"rcode":          dns.RcodeServerFailure, // SERVFAIL
			"answers":        0,
			"authoritative":  false,
			"answer_details": []map[string]any{},
		}

		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(result); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		return
	}

	// Check if it's a local zone
	isLocal, zone := h.lanResolver.IsLocalZone(name)

	result := map[string]interface{}{
		"hostname":      name,
		"is_local":      isLocal,
		"zone":          zone,
		"rcode":         response.Rcode,
		"answers":       len(response.Answer),
		"authoritative": response.Authoritative,
	}

	// Add answer details
	answers := make([]map[string]interface{}, 0, len(response.Answer))
	for _, rr := range response.Answer {
		answer := map[string]interface{}{
			"name":  rr.Header().Name,
			"type":  rr.Header().Rrtype,
			"class": rr.Header().Class,
			"ttl":   rr.Header().Ttl,
		}

		switch v := rr.(type) {
		case *dns.A:
			answer["data"] = v.A.String()
		case *dns.AAAA:
			answer["data"] = v.AAAA.String()
		}

		answers = append(answers, answer)
	}

	result["answer_details"] = answers

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(result); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}
}

// handleWatch provides WebSocket updates for lease changes.
//
//nolint:funcorder
func (h *LocalZonesHandler) handleWatch(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			// Allow connections from localhost only
			return r.RemoteAddr == "127.0.0.1" || r.RemoteAddr == "::1"
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	// Add client to the list
	h.clientsMu.Lock()
	h.clients[conn] = true
	h.clientsMu.Unlock()

	// Remove client when connection closes
	defer func() {
		h.clientsMu.Lock()
		delete(h.clients, conn)
		h.clientsMu.Unlock()

		_ = conn.Close()
	}()

	// Send initial data
	h.sendUpdate(conn)

	// Keep connection alive and handle ping/pong
	for {
		select {
		case <-r.Context().Done():
			return
		default:
			// Set read deadline
			const readTimeoutSeconds = 60

			_ = conn.SetReadDeadline(time.Now().Add(readTimeoutSeconds * time.Second))

			// Read message (we expect ping/pong)
			_, _, err := conn.ReadMessage()
			if err != nil {
				// Connection closed or error
				return
			}
		}
	}
}

// sendUpdate sends current state via WebSocket.
//
//nolint:funcorder
func (h *LocalZonesHandler) sendUpdate(conn *websocket.Conn) {
	zones, _ := h.zoneDetector.DetectZones()
	leases := h.leaseManager.GetAllLeases()

	update := map[string]interface{}{
		"type":   "update",
		"zones":  zones,
		"leases": leases,
		"count":  len(leases),
		"time":   time.Now().Unix(),
	}

	if err := conn.WriteJSON(update); err != nil {
		// Connection closed or error occurred
		return
	}
}

// BroadcastUpdate sends updates to all connected WebSocket clients.
func (h *LocalZonesHandler) BroadcastUpdate() {
	zones, _ := h.zoneDetector.DetectZones()
	leases := h.leaseManager.GetAllLeases()

	update := map[string]interface{}{
		"type":   "update",
		"zones":  zones,
		"leases": leases,
		"count":  len(leases),
		"time":   time.Now().Unix(),
	}

	h.clientsMu.RLock()

	clients := make([]*websocket.Conn, 0, len(h.clients))
	for conn := range h.clients {
		clients = append(clients, conn)
	}

	h.clientsMu.RUnlock()

	// Send update to all clients
	for _, conn := range clients {
		if err := conn.WriteJSON(update); err != nil {
			// Remove failed client
			h.clientsMu.Lock()
			delete(h.clients, conn)
			h.clientsMu.Unlock()

			_ = conn.Close()
		}
	}
}
