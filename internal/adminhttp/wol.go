package adminhttp

import (
	"encoding/json"
	"net/http"

	"github.com/bavix/outway/internal/wol"
)

// WOLHandler handles Wake-on-LAN API endpoints.
type WOLHandler struct {
	client *wol.Client
}

// NewWOLHandler creates a new WOL handler.
func NewWOLHandler(client *wol.Client) *WOLHandler {
	return &WOLHandler{
		client: client,
	}
}

// handleGetInterfaces returns all network interfaces with their broadcast addresses.
func (h *WOLHandler) handleGetInterfaces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	interfaces, err := h.client.GetNetworkInterfaces()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"interfaces": interfaces,
		"count":      len(interfaces),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleGetConfig returns the current WOL configuration.
func (h *WOLHandler) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	config := h.client.GetConfig()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(config); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleUpdateConfig updates the WOL configuration.
func (h *WOLHandler) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var config wol.Config
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate port
	if config.DefaultPort < 1 || config.DefaultPort > 65535 {
		http.Error(w, "Invalid port number (must be 1-65535)", http.StatusBadRequest)
		return
	}

	h.client.SetConfig(&config)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"config":  config,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// SendWOLRequest represents a request to send a WOL packet.
type SendWOLRequest struct {
	MAC       string `json:"mac"`
	Interface string `json:"interface,omitempty"` // Empty means all interfaces
	Port      int    `json:"port,omitempty"`      // 0 means use default
}

// handleSendWOL sends a Wake-on-LAN packet.
func (h *WOLHandler) handleSendWOL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SendWOLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate MAC address
	if req.MAC == "" {
		http.Error(w, "MAC address is required", http.StatusBadRequest)
		return
	}

	// Validate port if specified
	if req.Port != 0 && (req.Port < 1 || req.Port > 65535) {
		http.Error(w, "Invalid port number (must be 1-65535)", http.StatusBadRequest)
		return
	}

	var err error
	var sentTo []string

	if req.Interface == "" || req.Interface == "all" {
		// Send to all interfaces
		err = h.client.SendWOLToAll(req.MAC, req.Port)
		if err == nil {
			interfaces, _ := h.client.GetNetworkInterfaces()
			for _, iface := range interfaces {
				sentTo = append(sentTo, iface.Name)
			}
		}
	} else {
		// Send to specific interface
		interfaces, err := h.client.GetNetworkInterfaces()
		if err != nil {
			http.Error(w, "Failed to get network interfaces: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Find the specified interface
		found := false
		for _, iface := range interfaces {
			if iface.Name == req.Interface {
				found = true
				err = h.client.SendWOL(req.MAC, iface.Broadcast, req.Port)
				if err == nil {
					sentTo = append(sentTo, iface.Name)
				}
				break
			}
		}

		if !found {
			http.Error(w, "Interface not found: "+req.Interface, http.StatusNotFound)
			return
		}
	}

	if err != nil {
		http.Error(w, "Failed to send WOL packet: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"mac":     req.MAC,
		"sent_to": sentTo,
		"port":    req.Port,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
