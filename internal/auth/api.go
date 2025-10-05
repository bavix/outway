package auth

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/render"
	"github.com/gorilla/mux"
)

// APIHandler handles HTTP requests for authentication.
type APIHandler struct {
	authService *Service
}

// NewAPIHandler creates a new authentication API handler.
func NewAPIHandler(authService *Service) *APIHandler {
	return &APIHandler{
		authService: authService,
	}
}

// RegisterRoutes registers all authentication API routes.
func (h *APIHandler) RegisterRoutes(mux *mux.Router) {
	api := mux.PathPrefix("/api/v1/auth").Subrouter()

	// Authentication endpoints (no auth required)
	api.HandleFunc("/status", h.GetAuthStatus).Methods("GET")
	api.HandleFunc("/login", h.Login).Methods("POST")
	api.HandleFunc("/first-user", h.CreateFirstUser).Methods("POST")
	api.HandleFunc("/refresh", h.Refresh).Methods("POST")
}

// GetAuthStatus returns the authentication status (whether users exist).
func (h *APIHandler) GetAuthStatus(w http.ResponseWriter, r *http.Request) {
	status, err := h.authService.GetAuthStatus()
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "failed to get auth status"})

		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, status)
}

// Login handles user authentication and returns a JWT token.
func (h *APIHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "invalid JSON"})

		return
	}

	// Validate request
	if req.Email == "" || req.Password == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "email and password are required"})

		return
	}

	// Authenticate user
	response, err := h.authService.Login(&req)
	if err != nil {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "invalid credentials"})

		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, response)
}

// CreateFirstUser creates the first user if no users exist.
func (h *APIHandler) CreateFirstUser(w http.ResponseWriter, r *http.Request) {
	var req FirstUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "invalid JSON"})

		return
	}

	// Validate request
	if req.Email == "" || req.Password == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "email and password are required"})

		return
	}

	// Create first user
	response, err := h.authService.CreateFirstUser(&req)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": err.Error()})

		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, response)
}

// Refresh handles refresh token requests and returns new access and refresh tokens.
func (h *APIHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "invalid JSON"})

		return
	}

	// Validate request
	if req.RefreshToken == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "refresh token is required"})

		return
	}

	// Refresh tokens
	response, err := h.authService.Refresh(&req)
	if err != nil {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "invalid refresh token"})

		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, response)
}
