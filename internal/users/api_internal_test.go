package users

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bavix/outway/internal/config"
)

func TestAPIHandler_GetRoles(t *testing.T) {
	t.Parallel()

	cfg := &MockConfig{}
	handler := NewAPIHandler(cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/roles", nil)
	w := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/users/roles", handler.GetRoles).Methods("GET")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}

	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "roles")
	assert.Contains(t, response, "count")

	roles, ok := response["roles"].([]interface{})
	require.True(t, ok)
	assert.Len(t, roles, 2)

	// Check admin role
	adminRole, ok := roles[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "admin", adminRole["name"])
	description, ok := adminRole["description"].(string)
	require.True(t, ok)
	assert.Contains(t, description, "Full system access")

	permissionsCount, ok := adminRole["permissions_count"].(float64)
	require.True(t, ok)
	assert.Positive(t, int(permissionsCount))

	// Check user role
	userRole, ok := roles[1].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "user", userRole["name"])
	userDescription, ok := userRole["description"].(string)
	require.True(t, ok)
	assert.Contains(t, userDescription, "Limited access")

	userPermissionsCount, ok := userRole["permissions_count"].(float64)
	require.True(t, ok)
	assert.Positive(t, int(userPermissionsCount))
}

//nolint:funlen
func TestAPIHandler_GetRolePermissions(t *testing.T) {
	t.Parallel()

	cfg := &MockConfig{}
	handler := NewAPIHandler(cfg)

	tests := []struct {
		name           string
		role           string
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "admin role",
			role:           "admin",
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "user role",
			role:           "user",
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "unknown role",
			role:           "unknown",
			expectedStatus: http.StatusOK, // Should return user permissions as default
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/api/v1/users/roles/"+tt.role+"/permissions", nil)
			w := httptest.NewRecorder()

			router := mux.NewRouter()
			router.HandleFunc("/api/v1/users/roles/{role}/permissions", handler.GetRolePermissions).Methods("GET")
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if !tt.expectError {
				var response map[string]interface{}

				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Contains(t, response, "role")
				assert.Contains(t, response, "permissions")
				assert.Contains(t, response, "categories")
				assert.Contains(t, response, "count")

				assert.Equal(t, tt.role, response["role"])

				permissions, ok := response["permissions"].([]interface{})
				require.True(t, ok)
				assert.NotEmpty(t, permissions)

				categories, ok := response["categories"].(map[string]interface{})
				require.True(t, ok)
				assert.NotEmpty(t, categories)
			}
		})
	}
}

//nolint:funlen
func TestAPIHandler_CreateUser(t *testing.T) {
	t.Parallel()

	cfg := &MockConfig{
		Users: []config.UserConfig{},
	}
	handler := NewAPIHandler(cfg)

	tests := []struct {
		name           string
		userData       UserRequest
		expectedStatus int
		expectError    bool
		errorMessage   string
	}{
		{
			name: "valid admin user",
			userData: UserRequest{
				Email:    "admin@example.com",
				Password: "password123",
				Role:     "admin",
			},
			expectedStatus: http.StatusCreated,
			expectError:    false,
		},
		{
			name: "valid user role",
			userData: UserRequest{
				Email:    "user@example.com",
				Password: "password123",
				Role:     "user",
			},
			expectedStatus: http.StatusCreated,
			expectError:    false,
		},
		{
			name: "invalid role",
			userData: UserRequest{
				Email:    "test@example.com",
				Password: "password123",
				Role:     "invalid",
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
			errorMessage:   "invalid role",
		},
		{
			name: "empty email",
			userData: UserRequest{
				Email:    "",
				Password: "password123",
				Role:     "admin",
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
			errorMessage:   "email cannot be empty",
		},
		{
			name: "duplicate user",
			userData: UserRequest{
				Email:    "admin@example.com",
				Password: "password123",
				Role:     "admin",
			},
			expectedStatus: http.StatusCreated, // First creation should succeed
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Reset users for each test
			cfg.Users = []config.UserConfig{}
			cfg.SaveError = nil

			userDataJSON, err := json.Marshal(tt.userData)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBuffer(userDataJSON))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()

			router := mux.NewRouter()
			router.HandleFunc("/api/v1/users", handler.CreateUser).Methods("POST")
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectError {
				var response map[string]string

				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], tt.errorMessage)
			} else {
				var response UserResponse

				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, tt.userData.Email, response.Email)
				assert.Equal(t, tt.userData.Role, response.Role)
				assert.NotEmpty(t, response.Permissions)
			}
		})
	}
}

func TestAPIHandler_GetUsers(t *testing.T) {
	t.Parallel()

	cfg := &MockConfig{
		Users: []config.UserConfig{
			{
				Email:    "admin@example.com",
				Password: "hashed_password",
				Role:     "admin",
			},
			{
				Email:    "user@example.com",
				Password: "hashed_password",
				Role:     "user",
			},
		},
	}
	handler := NewAPIHandler(cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	w := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/users", handler.GetUsers).Methods("GET")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}

	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "users")
	assert.Contains(t, response, "count")
	assert.InEpsilon(t, float64(2), response["count"], 0.01)

	users, ok := response["users"].([]interface{})
	require.True(t, ok)
	assert.Len(t, users, 2)

	// Check that passwords are not exposed
	for _, userInterface := range users {
		user, ok := userInterface.(map[string]interface{})
		require.True(t, ok)
		assert.NotContains(t, user, "password")
		assert.Contains(t, user, "email")
		assert.Contains(t, user, "role")
		assert.Contains(t, user, "permissions")
	}
}

func TestGetRolePermissions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		role               string
		expectedCount      int
		expectedCategories []string
	}{
		{
			name:          "admin role",
			role:          "admin",
			expectedCount: 22, // Should have many permissions
			expectedCategories: []string{
				"System", "Users", "Devices", "DNS", "Configuration",
				"Updates", "Cache", "History", "Statistics", "Overview", "Info",
			},
		},
		{
			name:          "user role",
			role:          "user",
			expectedCount: 13, // Should have fewer permissions
			expectedCategories: []string{
				"System", "Devices", "DNS", "Configuration", "Updates",
				"Cache", "History", "Statistics", "Overview", "Info",
			},
		},
		{
			name:          "unknown role",
			role:          "unknown",
			expectedCount: 13, // Should default to user permissions
			expectedCategories: []string{
				"System", "Devices", "DNS", "Configuration", "Updates",
				"Cache", "History", "Statistics", "Overview", "Info",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			permissions := GetRolePermissions(tt.role)

			assert.Len(t, permissions, tt.expectedCount)

			// Check that all permissions have required fields
			for _, perm := range permissions {
				assert.NotEmpty(t, perm.Name)
				assert.NotEmpty(t, perm.Description)
				assert.NotEmpty(t, perm.Category)
			}

			// Check categories
			categories := make(map[string]bool)
			for _, perm := range permissions {
				categories[perm.Category] = true
			}

			for _, expectedCategory := range tt.expectedCategories {
				assert.True(t, categories[expectedCategory], "Expected category %s not found", expectedCategory)
			}
		})
	}
}

func TestAPIHandler_CreateUser_Duplicate(t *testing.T) {
	t.Parallel()

	cfg := &MockConfig{
		Users: []config.UserConfig{},
	}
	handler := NewAPIHandler(cfg)

	// First, create a user
	userData := UserRequest{
		Email:    "admin@example.com",
		Password: "password123",
		Role:     "admin",
	}

	userDataJSON, err := json.Marshal(userData)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBuffer(userDataJSON))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/users", handler.CreateUser).Methods("POST")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	// Now try to create the same user again
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBuffer(userDataJSON))
	req2.Header.Set("Content-Type", "application/json")

	w2 := httptest.NewRecorder()

	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusConflict, w2.Code)

	var response map[string]string

	err = json.Unmarshal(w2.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Contains(t, response["error"], "user already exists")
}
