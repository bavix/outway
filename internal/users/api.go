package users

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"

	"github.com/go-chi/render"
	"github.com/gorilla/mux"

	"github.com/bavix/outway/internal/config"
)

// ConfigInterface defines the interface for configuration operations.
type ConfigInterface interface {
	Save() error
	Load() error
	GetUsers() []config.UserConfig
	SetUsers(users []config.UserConfig)
}

// ConfigWrapper wraps config.Config to implement ConfigInterface.
type ConfigWrapper struct {
	*config.Config
}

func (c *ConfigWrapper) GetUsers() []config.UserConfig {
	return c.Users
}

func (c *ConfigWrapper) SetUsers(users []config.UserConfig) {
	c.Users = users
}

func (c *ConfigWrapper) Load() error {
	// Load is a package function, not a method
	// This is a no-op for the wrapper since config is already loaded
	return nil
}

var (
	ErrInvalidJSON          = errors.New("invalid JSON")
	ErrUserAlreadyExists    = errors.New("user with new email already exists")
	ErrFailedToHashPassword = errors.New("failed to hash password")
	ErrFailedToSaveConfig   = errors.New("failed to save config")
)

// APIHandler handles HTTP requests for user management.
type APIHandler struct {
	config ConfigInterface
	mu     sync.RWMutex // protects user operations
}

// NewAPIHandler creates a new user API handler.
func NewAPIHandler(cfg ConfigInterface) *APIHandler {
	return &APIHandler{
		config: cfg,
	}
}

// RegisterRoutes registers all user API routes.
func (h *APIHandler) RegisterRoutes(api *mux.Router) {
	// Role and permissions (must be before /{email} to avoid conflicts)
	api.HandleFunc("/roles", h.GetRoles).Methods("GET")
	api.HandleFunc("/roles/{role}/permissions", h.GetRolePermissions).Methods("GET")

	// User management
	api.HandleFunc("", h.GetUsers).Methods("GET")
	api.HandleFunc("", h.CreateUser).Methods("POST")
	api.HandleFunc("/{email}", h.GetUser).Methods("GET")
	api.HandleFunc("/{email}", h.UpdateUser).Methods("PUT")
	api.HandleFunc("/{email}", h.DeleteUser).Methods("DELETE")
	api.HandleFunc("/{email}/change-password", h.ChangePassword).Methods("POST")
}

// GetUsers returns all users.
func (h *APIHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	users := make([]*UserResponse, 0, len(h.config.GetUsers()))
	for _, user := range h.config.GetUsers() {
		users = append(users, &UserResponse{
			Email:       user.Email,
			Role:        user.Role,
			Permissions: GetRolePermissions(user.Role),
		})
	}

	render.JSON(w, r, map[string]interface{}{
		"users": users,
		"count": len(users),
	})
}

// GetUser returns a specific user.
func (h *APIHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	email := vars["email"]

	h.mu.RLock()
	defer h.mu.RUnlock()

	user := h.findUser(email)
	if user == nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "user not found"})

		return
	}

	render.JSON(w, r, &UserResponse{
		Email:       user.Email,
		Role:        user.Role,
		Permissions: GetRolePermissions(user.Role),
	})
}

// CreateUser creates a new user.
func (h *APIHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req UserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "invalid JSON"})

		return
	}

	// Validate request
	if err := ValidateUserRequest(&req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": err.Error()})

		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// Check if user already exists
	if h.findUser(req.Email) != nil {
		render.Status(r, http.StatusConflict)
		render.JSON(w, r, map[string]string{"error": "user already exists"})

		return
	}

	// Hash password
	hashedPassword, err := HashPassword(req.Password)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "failed to hash password"})

		return
	}

	// Create user
	user := &config.UserConfig{
		Email:    req.Email,
		Password: hashedPassword,
		Role:     req.Role,
	}

	// Add to config
	users := h.config.GetUsers()
	users = append(users, *user)
	h.config.SetUsers(users)

	// Save config
	if err := h.config.Save(); err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "failed to save config"})

		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, &UserResponse{
		Email:       user.Email,
		Role:        user.Role,
		Permissions: GetRolePermissions(user.Role),
	})
}

// UpdateUser updates an existing user.
func (h *APIHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	email := vars["email"]

	h.mu.Lock()
	defer h.mu.Unlock()

	user := h.findUser(email)
	if user == nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "user not found"})

		return
	}

	req, err := h.decodeUserRequest(r)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": err.Error()})

		return
	}

	if err := h.validateUserUpdate(req, email); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": err.Error()})

		return
	}

	if err := h.updateUser(user, req); err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": err.Error()})

		return
	}

	render.JSON(w, r, &UserResponse{
		Email:       user.Email,
		Role:        user.Role,
		Permissions: GetRolePermissions(user.Role),
	})
}

// DeleteUser deletes a user.
func (h *APIHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	email := vars["email"]

	h.mu.Lock()
	defer h.mu.Unlock()

	// Find user index
	users := h.config.GetUsers()
	index := -1

	for i, user := range users {
		if user.Email == email {
			index = i

			break
		}
	}

	if index == -1 {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "user not found"})

		return
	}

	// Remove user
	users = append(users[:index], users[index+1:]...)
	h.config.SetUsers(users)

	// Save config
	if err := h.config.Save(); err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "failed to save config"})

		return
	}

	render.Status(r, http.StatusNoContent)
}

// ChangePassword changes a user's password.
func (h *APIHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	email := vars["email"]

	h.mu.Lock()
	defer h.mu.Unlock()

	user := h.findUser(email)
	if user == nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "user not found"})

		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "invalid JSON"})

		return
	}

	if strings.TrimSpace(req.Password) == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "password cannot be empty"})

		return
	}

	if len(req.Password) < 6 { //nolint:mnd
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "password must be at least 6 characters"})

		return
	}

	// Hash new password
	hashedPassword, err := HashPassword(req.Password)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "failed to hash password"})

		return
	}

	// Update password
	user.Password = hashedPassword

	// Save config
	if err := h.config.Save(); err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "failed to save config"})

		return
	}

	render.Status(r, http.StatusNoContent)
}

// findUser finds a user by email.
func (h *APIHandler) findUser(email string) *config.UserConfig {
	users := h.config.GetUsers()
	for i := range users {
		if users[i].Email == email {
			return &users[i]
		}
	}

	return nil
}

// decodeUserRequest decodes user request from HTTP request body.
func (h *APIHandler) decodeUserRequest(r *http.Request) (*UserRequest, error) {
	var req UserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, ErrInvalidJSON
	}

	return &req, nil
}

// validateUserUpdate validates user update request.
func (h *APIHandler) validateUserUpdate(req *UserRequest, currentEmail string) error {
	if err := ValidateUserRequest(req); err != nil {
		return err
	}

	// Check if email is being changed and new email already exists
	if req.Email != currentEmail && h.findUser(req.Email) != nil {
		return ErrUserAlreadyExists
	}

	return nil
}

// updateUser updates user with new data.
func (h *APIHandler) updateUser(user *config.UserConfig, req *UserRequest) error {
	// Hash password if provided
	hashedPassword := user.Password
	if req.Password != "" {
		var err error

		hashedPassword, err = HashPassword(req.Password)
		if err != nil {
			return ErrFailedToHashPassword
		}
	}

	// Update user
	user.Email = req.Email
	user.Password = hashedPassword
	user.Role = req.Role

	// Update user in config
	users := h.config.GetUsers()
	for i := range users {
		if users[i].Email == user.Email {
			users[i] = *user

			break
		}
	}

	h.config.SetUsers(users)

	// Save config
	if err := h.config.Save(); err != nil {
		return ErrFailedToSaveConfig
	}

	return nil
}

// GetRoles returns all available roles with their descriptions.
func (h *APIHandler) GetRoles(w http.ResponseWriter, r *http.Request) {
	roles := []map[string]interface{}{
		{
			"name":              "admin",
			"description":       "Full system access with all permissions",
			"permissions_count": len(getAdminPermissions()),
		},
		{
			"name":              "user",
			"description":       "Limited access for regular users",
			"permissions_count": len(getUserPermissions()),
		},
	}

	render.JSON(w, r, map[string]interface{}{
		"roles": roles,
		"count": len(roles),
	})
}

// GetRolePermissions returns permissions for a specific role.
func (h *APIHandler) GetRolePermissions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	role := vars["role"]

	permissions := GetRolePermissions(role)

	// Group permissions by category
	categories := make(map[string][]Permission)
	for _, perm := range permissions {
		categories[perm.Category] = append(categories[perm.Category], perm)
	}

	render.JSON(w, r, map[string]interface{}{
		"role":        role,
		"permissions": permissions,
		"categories":  categories,
		"count":       len(permissions),
	})
}
