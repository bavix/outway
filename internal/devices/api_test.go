package devices_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bavix/outway/internal/devices"
)

func TestAPIHandler_GetDevices(t *testing.T) {
	t.Parallel()

	manager := devices.NewDeviceManager()
	handler := devices.NewAPIHandler(manager)

	// Add test device
	_, err := manager.AddDevice("Test Device", "aa:bb:cc:dd:ee:ff", "192.168.1.1", "test.local", "Test Vendor")
	require.NoError(t, err)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
	w := httptest.NewRecorder()

	// Create router and register handler
	router := mux.NewRouter()
	api := router.PathPrefix("/api/v1").Subrouter()
	handler.RegisterRoutes(api)

	// Serve request
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}

	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "devices")
	assert.Contains(t, response, "count")
	assert.InEpsilon(t, float64(1), response["count"], 0.01)

	devices, ok := response["devices"].([]interface{})
	require.True(t, ok)
	assert.Len(t, devices, 1)

	device, ok := devices[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Test Device", device["name"])
	assert.Equal(t, "aa:bb:cc:dd:ee:ff", device["mac"])
	assert.Equal(t, "192.168.1.1", device["ip"])
}

func TestAPIHandler_GetDevice(t *testing.T) {
	t.Parallel()

	manager := devices.NewDeviceManager()
	handler := devices.NewAPIHandler(manager)

	// Add test device
	device, err := manager.AddDevice("Test Device", "aa:bb:cc:dd:ee:ff", "192.168.1.1", "test.local", "Test Vendor")
	require.NoError(t, err)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/"+device.ID, nil)
	w := httptest.NewRecorder()

	// Create router and register handler
	router := mux.NewRouter()
	api := router.PathPrefix("/api/v1").Subrouter()
	handler.RegisterRoutes(api)

	// Serve request
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}

	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Test Device", response["name"])
	assert.Equal(t, "aa:bb:cc:dd:ee:ff", response["mac"])
	assert.Equal(t, "192.168.1.1", response["ip"])
}

func TestAPIHandler_GetDevice_NotFound(t *testing.T) {
	t.Parallel()

	manager := devices.NewDeviceManager()
	handler := devices.NewAPIHandler(manager)

	// Create request for non-existent device
	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/non-existent", nil)
	w := httptest.NewRecorder()

	// Create router and register handler
	router := mux.NewRouter()
	api := router.PathPrefix("/api/v1").Subrouter()
	handler.RegisterRoutes(api)

	// Serve request
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}

	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "error")
	assert.Equal(t, "Device not found", response["error"])
}

func TestAPIHandler_AddDevice(t *testing.T) {
	t.Parallel()

	manager := devices.NewDeviceManager()
	handler := devices.NewAPIHandler(manager)

	// Create request body
	requestBody := map[string]string{
		"name":     "New Device",
		"mac":      "aa:bb:cc:dd:ee:ff",
		"ip":       "192.168.1.1",
		"hostname": "new.local",
		"vendor":   "New Vendor",
	}

	jsonBody, err := json.Marshal(requestBody)
	require.NoError(t, err)

	// Create request
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	// Create router and register handler
	router := mux.NewRouter()
	api := router.PathPrefix("/api/v1").Subrouter()
	handler.RegisterRoutes(api)

	// Serve request
	router.ServeHTTP(w, req)

	// Check response - should return 501 since not implemented
	assert.Equal(t, http.StatusNotImplemented, w.Code)

	var response map[string]interface{}

	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "error")
	assert.Equal(t, "Add device not implemented", response["error"])
}

func TestAPIHandler_AddDevice_InvalidJSON(t *testing.T) {
	t.Parallel()

	manager := devices.NewDeviceManager()
	handler := devices.NewAPIHandler(manager)

	// Create request with invalid JSON
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	// Create router and register handler
	router := mux.NewRouter()
	api := router.PathPrefix("/api/v1").Subrouter()
	handler.RegisterRoutes(api)

	// Serve request
	router.ServeHTTP(w, req)

	// Check response - should return 501 since not implemented
	assert.Equal(t, http.StatusNotImplemented, w.Code)

	var response map[string]interface{}

	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "error")
	assert.Equal(t, "Add device not implemented", response["error"])
}

func TestAPIHandler_UpdateDevice(t *testing.T) {
	t.Parallel()

	manager := devices.NewDeviceManager()
	handler := devices.NewAPIHandler(manager)

	// Add test device
	device, err := manager.AddDevice("Test Device", "aa:bb:cc:dd:ee:ff", "192.168.1.1", "test.local", "Test Vendor")
	require.NoError(t, err)

	// Create request body
	requestBody := map[string]string{
		"name":     "Updated Device",
		"mac":      "aa:bb:cc:dd:ee:ff",
		"ip":       "192.168.1.2",
		"hostname": "updated.local",
		"vendor":   "Updated Vendor",
	}

	jsonBody, err := json.Marshal(requestBody)
	require.NoError(t, err)

	// Create request
	req := httptest.NewRequest(http.MethodPut, "/api/v1/devices/"+device.ID, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	// Create router and register handler
	router := mux.NewRouter()
	api := router.PathPrefix("/api/v1").Subrouter()
	handler.RegisterRoutes(api)

	// Serve request
	router.ServeHTTP(w, req)

	// Check response - should return 501 since not implemented
	assert.Equal(t, http.StatusNotImplemented, w.Code)

	var response map[string]interface{}

	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "error")
	assert.Equal(t, "Update device not implemented", response["error"])
	assert.Contains(t, response, "id")
}

func TestAPIHandler_DeleteDevice(t *testing.T) {
	t.Parallel()

	manager := devices.NewDeviceManager()
	handler := devices.NewAPIHandler(manager)

	// Add test device
	device, err := manager.AddDevice("Test Device", "aa:bb:cc:dd:ee:ff", "192.168.1.1", "test.local", "Test Vendor")
	require.NoError(t, err)

	// Create request
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/devices/"+device.ID, nil)
	w := httptest.NewRecorder()

	// Create router and register handler
	router := mux.NewRouter()
	api := router.PathPrefix("/api/v1").Subrouter()
	handler.RegisterRoutes(api)

	// Serve request
	router.ServeHTTP(w, req)

	// Check response - should return 501 since not implemented
	assert.Equal(t, http.StatusNotImplemented, w.Code)

	var response map[string]interface{}

	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "error")
	assert.Equal(t, "Delete device not implemented", response["error"])
	assert.Contains(t, response, "id")
}

func TestAPIHandler_GetStats(t *testing.T) {
	t.Parallel()

	manager := devices.NewDeviceManager()
	handler := devices.NewAPIHandler(manager)

	// Add some devices
	_, err := manager.AddDevice("Device 1", "aa:bb:cc:dd:ee:ff", "192.168.1.1", "device1.local", "Vendor 1")
	require.NoError(t, err)

	_, err = manager.AddDevice("Device 2", "bb:cc:dd:ee:ff:aa", "192.168.1.2", "device2.local", "Vendor 2")
	require.NoError(t, err)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/stats", nil)
	w := httptest.NewRecorder()

	// Create router and register handler
	router := mux.NewRouter()
	api := router.PathPrefix("/api/v1").Subrouter()
	handler.RegisterRoutes(api)

	// Serve request
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}

	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.InEpsilon(t, float64(2), response["total_devices"], 0.01)
	assert.Contains(t, response, "online_devices")
	assert.Contains(t, response, "wakeable_devices")
	assert.Contains(t, response, "resolvable_devices")
	assert.Contains(t, response, "device_types")
}

func TestAPIHandler_GetOnlineDevices(t *testing.T) {
	t.Parallel()

	manager := devices.NewDeviceManager()
	handler := devices.NewAPIHandler(manager)

	// Add online device
	_, err := manager.AddDevice("Online Device", "aa:bb:cc:dd:ee:ff", "192.168.1.1", "online.local", "Vendor")
	require.NoError(t, err)

	// Add offline device
	offlineDevice, err := manager.AddDevice("Offline Device", "bb:cc:dd:ee:ff:aa", "192.168.1.2", "offline.local", "Vendor")
	require.NoError(t, err)
	manager.UpdateDeviceStatus(offlineDevice.ID, "offline")

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/online", nil)
	w := httptest.NewRecorder()

	// Create router and register handler
	router := mux.NewRouter()
	api := router.PathPrefix("/api/v1").Subrouter()
	handler.RegisterRoutes(api)

	// Serve request
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}

	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.InEpsilon(t, float64(1), response["count"], 0.01)

	devices, ok := response["devices"].([]interface{})
	require.True(t, ok)
	assert.Len(t, devices, 1)

	device, ok := devices[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Online Device", device["name"])
}

func TestAPIHandler_GetWakeableDevices(t *testing.T) {
	t.Parallel()

	manager := devices.NewDeviceManager()
	handler := devices.NewAPIHandler(manager)

	// Add wakeable device
	wakeableDevice, err := manager.AddDevice("Wakeable Device", "aa:bb:cc:dd:ee:ff", "192.168.1.1", "wakeable.local", "Vendor")
	require.NoError(t, err)

	// Set device to offline to make it wakeable
	manager.UpdateDeviceStatus(wakeableDevice.ID, "offline")

	// Add non-wakeable device (invalid MAC)
	_, err = manager.AddDevice("Non-wakeable Device", "invalid-mac", "192.168.1.2", "non-wakeable.local", "Vendor")
	require.NoError(t, err)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/wakeable", nil)
	w := httptest.NewRecorder()

	// Create router and register handler
	router := mux.NewRouter()
	api := router.PathPrefix("/api/v1").Subrouter()
	handler.RegisterRoutes(api)

	// Serve request
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}

	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.InEpsilon(t, float64(1), response["count"], 0.01)

	devices, ok := response["devices"].([]interface{})
	require.True(t, ok)
	assert.Len(t, devices, 1)

	device, ok := devices[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Wakeable Device", device["name"])
}

func TestAPIHandler_GetDevicesByType(t *testing.T) {
	t.Parallel()

	manager := devices.NewDeviceManager()
	handler := devices.NewAPIHandler(manager)

	// Add devices of different types
	_, err := manager.AddDevice("MacBook", "aa:bb:cc:dd:ee:ff", "192.168.1.1", "macbook.local", "Apple MacBook")
	require.NoError(t, err)

	_, err = manager.AddDevice("iPhone", "bb:cc:dd:ee:ff:aa", "192.168.1.2", "iphone.local", "Apple iPhone")
	require.NoError(t, err)

	// Create request for computers
	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/type/computer", nil)
	w := httptest.NewRecorder()

	// Create router and register handler
	router := mux.NewRouter()
	api := router.PathPrefix("/api/v1").Subrouter()
	handler.RegisterRoutes(api)

	// Serve request
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}

	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.InEpsilon(t, float64(1), response["count"], 0.01)
	assert.Equal(t, "computer", response["type"])

	devices, ok := response["devices"].([]interface{})
	require.True(t, ok)
	assert.Len(t, devices, 1)

	device, ok := devices[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "MacBook", device["name"])
}
