package users

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	MinEmailLength    = 5
	MaxEmailLength    = 100
	Argon2Memory      = 64 * 1024 // 64 MB
	Argon2Iterations  = 3
	Argon2Parallelism = 2
	Argon2SaltLength  = 16
	Argon2KeyLength   = 32
	Argon2PartsCount  = 6
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// isValidEmail validates email format
func isValidEmail(email string) bool {
	return emailRegex.MatchString(email)
}

var (
	ErrInvalidHashFormat        = errors.New("invalid hash format")
	ErrUnsupportedHashAlgorithm = errors.New("unsupported hash algorithm")
	ErrEmailCannotBeEmpty       = errors.New("email cannot be empty")
	ErrEmailTooShort            = errors.New("email must be at least 5 characters")
	ErrEmailTooLong             = errors.New("email must be no more than 100 characters")
	ErrInvalidEmailFormat       = errors.New("invalid email format")
	ErrPasswordCannotBeEmpty    = errors.New("password cannot be empty")
	ErrInvalidRole              = errors.New("invalid role, supported roles: admin, user")
)

// User represents a user in the system.
type User struct {
	Email    string `json:"email"`
	Password string `json:"-"` // Never expose password hash in JSON
	Role     string `json:"role"`
}

// UserRequest represents a user creation/update request.
type UserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// UserResponse represents a user response (without password).
type UserResponse struct {
	Email       string       `json:"email"`
	Role        string       `json:"role"`
	Permissions []Permission `json:"permissions,omitempty"`
}

// Permission represents a user permission
type Permission struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

// Argon2Params holds the parameters for Argon2 hashing.
type Argon2Params struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

// DefaultArgon2Params returns default Argon2 parameters.
func DefaultArgon2Params() *Argon2Params {
	return &Argon2Params{
		Memory:      Argon2Memory,
		Iterations:  Argon2Iterations,
		Parallelism: Argon2Parallelism,
		SaltLength:  Argon2SaltLength,
		KeyLength:   Argon2KeyLength,
	}
}

// HashPassword hashes a password using Argon2.
func HashPassword(password string) (string, error) {
	params := DefaultArgon2Params()

	// Generate a random salt
	salt, err := generateRandomBytes(params.SaltLength)
	if err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// Hash the password
	hash := argon2.IDKey([]byte(password), salt, params.Iterations, params.Memory, params.Parallelism, params.KeyLength)

	// Encode the hash and salt
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	// Return the encoded hash
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, params.Memory, params.Iterations, params.Parallelism, b64Salt, b64Hash), nil
}

// VerifyPassword verifies a password against a hash.
func VerifyPassword(password, hash string) (bool, error) {
	// Parse the hash
	parts := strings.Split(hash, "$")
	if len(parts) != Argon2PartsCount {
		return false, ErrInvalidHashFormat
	}

	if parts[1] != "argon2id" {
		return false, ErrUnsupportedHashAlgorithm
	}

	// Decode parameters
	var (
		version            int
		memory, iterations uint32
		parallelism        uint8
	)

	_, err := fmt.Sscanf(parts[2], "v=%d", &version)
	if err != nil {
		return false, fmt.Errorf("failed to parse version: %w", err)
	}

	_, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism)
	if err != nil {
		return false, fmt.Errorf("failed to parse parameters: %w", err)
	}

	// Decode salt and hash
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("failed to decode salt: %w", err)
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("failed to decode hash: %w", err)
	}

	// Hash the input password
	actualHash := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, uint32(len(expectedHash))) //nolint:gosec

	// Compare hashes using constant time comparison
	return subtle.ConstantTimeCompare(expectedHash, actualHash) == 1, nil
}

// generateRandomBytes generates random bytes of the specified length.
func generateRandomBytes(length uint32) ([]byte, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)

	return bytes, err
}

// ValidateUserRequest validates a user request.
func ValidateUserRequest(req *UserRequest) error {
	if strings.TrimSpace(req.Email) == "" {
		return ErrEmailCannotBeEmpty
	}

	if len(req.Email) < MinEmailLength {
		return ErrEmailTooShort
	}

	if len(req.Email) > MaxEmailLength {
		return ErrEmailTooLong
	}

	// Basic email format validation
	if !isValidEmail(req.Email) {
		return ErrInvalidEmailFormat
	}

	if strings.TrimSpace(req.Password) == "" {
		return ErrPasswordCannotBeEmpty
	}

	if req.Role == "" {
		req.Role = "user" // Default role
	}

	// Validate role
	if req.Role != "admin" && req.Role != "user" {
		return ErrInvalidRole
	}

	return nil
}

// ToResponse converts a User to UserResponse.
func (u *User) ToResponse() *UserResponse {
	return &UserResponse{
		Email:       u.Email,
		Role:        u.Role,
		Permissions: GetRolePermissions(u.Role),
	}
}

// GetRolePermissions returns permissions for a given role
func GetRolePermissions(role string) []Permission {
	switch role {
	case "admin":
		return getAdminPermissions()
	case "user":
		return getUserPermissions()
	default:
		return getUserPermissions() // Default to user permissions
	}
}

// getAdminPermissions returns admin permissions
func getAdminPermissions() []Permission {
	return []Permission{
		// System permissions
		{Name: "system:view", Description: "View system information", Category: "System"},
		{Name: "system:manage", Description: "Manage system settings", Category: "System"},

		// User management permissions
		{Name: "users:view", Description: "View users", Category: "Users"},
		{Name: "users:create", Description: "Create users", Category: "Users"},
		{Name: "users:update", Description: "Update users", Category: "Users"},
		{Name: "users:delete", Description: "Delete users", Category: "Users"},

		// Device management permissions
		{Name: "devices:view", Description: "View devices", Category: "Devices"},
		{Name: "devices:manage", Description: "Manage devices", Category: "Devices"},
		{Name: "devices:wake", Description: "Wake devices", Category: "Devices"},

		// DNS management permissions
		{Name: "dns:view", Description: "View DNS settings", Category: "DNS"},
		{Name: "dns:manage", Description: "Manage DNS settings", Category: "DNS"},

		// Configuration permissions
		{Name: "config:view", Description: "View configuration", Category: "Configuration"},
		{Name: "config:manage", Description: "Manage configuration", Category: "Configuration"},

		// Update permissions
		{Name: "updates:view", Description: "View updates", Category: "Updates"},
		{Name: "updates:manage", Description: "Manage updates", Category: "Updates"},

		// Cache permissions
		{Name: "cache:view", Description: "View cache", Category: "Cache"},
		{Name: "cache:manage", Description: "Manage cache", Category: "Cache"},

		// History permissions
		{Name: "history:view", Description: "View history", Category: "History"},

		// Statistics permissions
		{Name: "stats:view", Description: "View statistics", Category: "Statistics"},

		// Overview permissions
		{Name: "overview:view", Description: "View overview", Category: "Overview"},

		// Info permissions
		{Name: "info:view", Description: "View system info", Category: "Info"},

		// Resolve permissions
		{Name: "resolve:view", Description: "Test DNS resolution", Category: "DNS"},
	}
}

// getUserPermissions returns user permissions
func getUserPermissions() []Permission {
	return []Permission{
		// System permissions (limited)
		{Name: "system:view", Description: "View system information", Category: "System"},

		// Device management permissions
		{Name: "devices:view", Description: "View devices", Category: "Devices"},
		{Name: "devices:manage", Description: "Manage devices (scan, refresh)", Category: "Devices"},
		{Name: "devices:wake", Description: "Wake devices (Wake-on-LAN)", Category: "Devices"},

		// DNS management permissions (limited)
		{Name: "dns:view", Description: "Test DNS resolution", Category: "DNS"},

		// Configuration permissions (limited)
		{Name: "config:view", Description: "View configuration (safe version)", Category: "Configuration"},

		// Update permissions (limited)
		{Name: "updates:view", Description: "View updates", Category: "Updates"},

		// Cache permissions (limited)
		{Name: "cache:view", Description: "View cache", Category: "Cache"},

		// History permissions
		{Name: "history:view", Description: "View DNS history", Category: "History"},

		// Statistics permissions
		{Name: "stats:view", Description: "View statistics", Category: "Statistics"},

		// Overview permissions
		{Name: "overview:view", Description: "View overview", Category: "Overview"},

		// Info permissions
		{Name: "info:view", Description: "View system info", Category: "Info"},

		// Resolve permissions
		{Name: "resolve:view", Description: "Test DNS resolution", Category: "DNS"},
	}
}
