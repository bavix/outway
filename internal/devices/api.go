package devices

import (
	"net/http"

	"github.com/go-chi/render"
	"github.com/gorilla/mux"
)

// APIHandler handles HTTP requests for device management.
type APIHandler struct {
	manager *DeviceManager
}

// NewAPIHandler creates a new API handler.
func NewAPIHandler(manager *DeviceManager) *APIHandler {
	return &APIHandler{
		manager: manager,
	}
}

// RegisterRoutes registers all device API routes.
func (h *APIHandler) RegisterRoutes(api *mux.Router) {
	devicesAPI := api.PathPrefix("/devices").Subrouter()

	// Device filtering (must come before /{id} routes)
	devicesAPI.HandleFunc("/online", h.GetOnlineDevices).Methods("GET")
	devicesAPI.HandleFunc("/wakeable", h.GetWakeableDevices).Methods("GET")
	devicesAPI.HandleFunc("/resolvable", h.GetResolvableDevices).Methods("GET")
	devicesAPI.HandleFunc("/type/{type}", h.GetDevicesByType).Methods("GET")
	devicesAPI.HandleFunc("/scan", h.ScanDevices).Methods("GET")
	devicesAPI.HandleFunc("/stats", h.GetStats).Methods("GET")
	devicesAPI.HandleFunc("/wake-all", h.WakeAllDevices).Methods("POST")

	// Device management
	devicesAPI.HandleFunc("", h.GetDevices).Methods("GET")
	devicesAPI.HandleFunc("", h.AddDevice).Methods("POST")
	devicesAPI.HandleFunc("/{id}", h.GetDevice).Methods("GET")
	devicesAPI.HandleFunc("/{id}", h.UpdateDevice).Methods("PUT")
	devicesAPI.HandleFunc("/{id}", h.DeleteDevice).Methods("DELETE")

	// Device actions
	devicesAPI.HandleFunc("/{id}/wake", h.WakeDevice).Methods("POST")
	devicesAPI.HandleFunc("/{id}/resolve", h.ResolveDevice).Methods("GET")
}

// GetDevices returns all devices.
func (h *APIHandler) GetDevices(w http.ResponseWriter, r *http.Request) {
	if h.manager == nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]interface{}{
			"error": "Device manager not initialized",
		})

		return
	}

	devices := h.manager.GetAllDevices()

	// Convert to API format
	deviceList := make([]map[string]interface{}, 0, len(devices))
	for _, device := range devices {
		deviceList = append(deviceList, map[string]interface{}{
			"id":        device.ID,
			"name":      device.Name,
			"mac":       device.MAC,
			"ip":        device.IP,
			"hostname":  device.Hostname,
			"vendor":    device.Vendor,
			"type":      string(device.Type),
			"status":    device.Status,
			"last_seen": device.LastSeen,
			"source":    device.Source,
		})
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, map[string]interface{}{
		"devices": deviceList,
		"count":   len(deviceList),
		"message": "Devices retrieved successfully",
	})
}

// GetOnlineDevices returns only online devices.
func (h *APIHandler) GetOnlineDevices(w http.ResponseWriter, r *http.Request) {
	h.handleDeviceList(w, r, func() []*Device {
		return h.manager.GetOnlineDevices()
	})
}

// GetWakeableDevices returns devices that can be woken up.
func (h *APIHandler) GetWakeableDevices(w http.ResponseWriter, r *http.Request) {
	h.handleDeviceList(w, r, func() []*Device {
		return h.manager.GetWakeableDevices()
	})
}

// GetResolvableDevices returns devices that can be resolved via DNS.
func (h *APIHandler) GetResolvableDevices(w http.ResponseWriter, r *http.Request) {
	render.Status(r, http.StatusOK)
	render.JSON(w, r, map[string]interface{}{
		"devices": []map[string]interface{}{},
		"count":   0,
	})
}

// GetDevicesByType returns devices filtered by type.
func (h *APIHandler) GetDevicesByType(w http.ResponseWriter, r *http.Request) {
	if h.manager == nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]interface{}{
			"error": "Device manager not initialized",
		})

		return
	}

	vars := mux.Vars(r)
	deviceType := vars["type"]

	devices := h.manager.GetDevicesByType(DeviceType(deviceType))
	if devices == nil {
		devices = []*Device{}
	}

	// Convert to API format
	deviceList := make([]map[string]interface{}, 0, len(devices))
	for _, device := range devices {
		deviceList = append(deviceList, map[string]interface{}{
			"id":        device.ID,
			"name":      device.Name,
			"mac":       device.MAC,
			"ip":        device.IP,
			"hostname":  device.Hostname,
			"vendor":    device.Vendor,
			"type":      string(device.Type),
			"status":    device.Status,
			"last_seen": device.LastSeen,
			"source":    device.Source,
		})
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, map[string]interface{}{
		"devices": deviceList,
		"count":   len(deviceList),
		"type":    deviceType,
	})
}

// ScanDevices performs a network scan.
func (h *APIHandler) ScanDevices(w http.ResponseWriter, r *http.Request) {
	if h.manager == nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]interface{}{
			"error": "Device manager not initialized",
		})

		return
	}

	// Perform actual network scan
	devices, err := h.manager.ScanNetwork(r.Context())
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]interface{}{
			"error":   "Scan failed",
			"message": err.Error(),
		})

		return
	}

	// Convert to API format
	deviceList := make([]map[string]interface{}, 0, len(devices))
	for _, device := range devices {
		deviceList = append(deviceList, map[string]interface{}{
			"id":        device.ID,
			"name":      device.Name,
			"mac":       device.MAC,
			"ip":        device.IP,
			"hostname":  device.Hostname,
			"vendor":    device.Vendor,
			"type":      string(device.Type),
			"status":    device.Status,
			"last_seen": device.LastSeen,
			"source":    device.Source,
		})
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, map[string]interface{}{
		"devices": deviceList,
		"count":   len(deviceList),
		"message": "Scan completed successfully",
	})
}

// GetStats returns device statistics.
func (h *APIHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	if h.manager == nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]interface{}{
			"error": "Device manager not initialized",
		})

		return
	}

	stats := h.manager.GetStats()

	render.Status(r, http.StatusOK)
	render.JSON(w, r, stats)
}

// GetDevice returns a specific device by ID.
func (h *APIHandler) GetDevice(w http.ResponseWriter, r *http.Request) {
	if h.manager == nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]interface{}{
			"error": "Device manager not initialized",
		})

		return
	}

	vars := mux.Vars(r)
	deviceID := vars["id"]

	device, exists := h.manager.GetDeviceByID(deviceID)
	if !exists {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{
			"error": "Device not found",
			"id":    deviceID,
		})

		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, map[string]interface{}{
		"id":        device.ID,
		"name":      device.Name,
		"mac":       device.MAC,
		"ip":        device.IP,
		"hostname":  device.Hostname,
		"vendor":    device.Vendor,
		"type":      string(device.Type),
		"status":    device.Status,
		"last_seen": device.LastSeen,
		"source":    device.Source,
	})
}

// AddDevice adds a new device.
func (h *APIHandler) AddDevice(w http.ResponseWriter, r *http.Request) {
	render.Status(r, http.StatusNotImplemented)
	render.JSON(w, r, map[string]string{
		"error": "Add device not implemented",
	})
}

// UpdateDevice updates an existing device.
func (h *APIHandler) UpdateDevice(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deviceID := vars["id"]

	render.Status(r, http.StatusNotImplemented)
	render.JSON(w, r, map[string]string{
		"error": "Update device not implemented",
		"id":    deviceID,
	})
}

// DeleteDevice deletes a device.
func (h *APIHandler) DeleteDevice(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deviceID := vars["id"]

	render.Status(r, http.StatusNotImplemented)
	render.JSON(w, r, map[string]string{
		"error": "Delete device not implemented",
		"id":    deviceID,
	})
}

// WakeDevice wakes up a device.
func (h *APIHandler) WakeDevice(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deviceID := vars["id"]

	render.Status(r, http.StatusNotImplemented)
	render.JSON(w, r, map[string]string{
		"error": "Wake device not implemented",
		"id":    deviceID,
	})
}

// WakeAllDevices wakes up all wakeable devices.
func (h *APIHandler) WakeAllDevices(w http.ResponseWriter, r *http.Request) {
	render.Status(r, http.StatusNotImplemented)
	render.JSON(w, r, map[string]string{
		"error": "Wake all devices not implemented",
	})
}

// ResolveDevice resolves a device via DNS.
func (h *APIHandler) ResolveDevice(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deviceID := vars["id"]

	render.Status(r, http.StatusNotImplemented)
	render.JSON(w, r, map[string]string{
		"error": "Resolve device not implemented",
		"id":    deviceID,
	})
}

// handleDeviceList is a common handler for device list endpoints.
func (h *APIHandler) handleDeviceList(w http.ResponseWriter, r *http.Request, getDevices func() []*Device) {
	if h.manager == nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]interface{}{
			"error": "Device manager not initialized",
		})

		return
	}

	devices := getDevices()
	if devices == nil {
		devices = []*Device{}
	}

	// Convert to API format
	deviceList := h.convertDevicesToAPI(devices)

	render.Status(r, http.StatusOK)
	render.JSON(w, r, map[string]interface{}{
		"devices": deviceList,
		"count":   len(deviceList),
	})
}

// convertDevicesToAPI converts device slice to API format.
func (h *APIHandler) convertDevicesToAPI(devices []*Device) []map[string]interface{} {
	deviceList := make([]map[string]interface{}, 0, len(devices))
	for _, device := range devices {
		deviceList = append(deviceList, map[string]interface{}{
			"id":        device.ID,
			"name":      device.Name,
			"mac":       device.MAC,
			"ip":        device.IP,
			"hostname":  device.Hostname,
			"vendor":    device.Vendor,
			"type":      string(device.Type),
			"status":    device.Status,
			"last_seen": device.LastSeen,
			"source":    device.Source,
		})
	}

	return deviceList
}
