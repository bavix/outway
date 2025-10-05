package users

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:funlen
func TestValidateUserRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		request     UserRequest
		expectError bool
		errorType   error
	}{
		{
			name: "valid admin request",
			request: UserRequest{
				Email:    "admin@example.com",
				Password: "password123",
				Role:     "admin",
			},
			expectError: false,
		},
		{
			name: "valid user request",
			request: UserRequest{
				Email:    "user@example.com",
				Password: "password123",
				Role:     "user",
			},
			expectError: false,
		},
		{
			name: "empty email",
			request: UserRequest{
				Email:    "",
				Password: "password123",
				Role:     "admin",
			},
			expectError: true,
			errorType:   ErrEmailCannotBeEmpty,
		},
		{
			name: "email too short",
			request: UserRequest{
				Email:    "ab",
				Password: "password123",
				Role:     "admin",
			},
			expectError: true,
			errorType:   ErrEmailTooShort,
		},
		{
			name: "email too long",
			request: UserRequest{
				Email:    "this_is_a_very_long_email_address_that_exceeds_one_hundred_characters_limit_and_should_fail_validation@example.com",
				Password: "password123",
				Role:     "admin",
			},
			expectError: true,
			errorType:   ErrEmailTooLong,
		},
		{
			name: "empty password",
			request: UserRequest{
				Email:    "admin@example.com",
				Password: "",
				Role:     "admin",
			},
			expectError: true,
			errorType:   ErrPasswordCannotBeEmpty,
		},
		{
			name: "invalid role",
			request: UserRequest{
				Email:    "admin@example.com",
				Password: "password123",
				Role:     "invalid",
			},
			expectError: true,
			errorType:   ErrInvalidRole,
		},
		{
			name: "empty role defaults to user",
			request: UserRequest{
				Email:    "user@example.com",
				Password: "password123",
				Role:     "",
			},
			expectError: false,
		},
		{
			name: "whitespace email",
			request: UserRequest{
				Email:    "   ",
				Password: "password123",
				Role:     "admin",
			},
			expectError: true,
			errorType:   ErrEmailCannotBeEmpty,
		},
		{
			name: "whitespace password",
			request: UserRequest{
				Email:    "admin@example.com",
				Password: "   ",
				Role:     "admin",
			},
			expectError: true,
			errorType:   ErrPasswordCannotBeEmpty,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateUserRequest(&tt.request)

			if tt.expectError {
				require.Error(t, err)

				if tt.errorType != nil {
					require.ErrorIs(t, err, tt.errorType)
				}
			} else {
				assert.NoError(t, err)
				// Check that empty role was set to "user"
				if tt.request.Role == "" {
					assert.Equal(t, "user", tt.request.Role)
				}
			}
		})
	}
}

func TestHashPassword(t *testing.T) {
	t.Parallel()

	password := "testpassword123"

	hash, err := HashPassword(password)
	require.NoError(t, err)
	require.NotEmpty(t, hash)

	// Hash should start with argon2id prefix
	assert.Contains(t, hash, "$argon2id$")

	// Hash should contain all required parts
	parts := splitHash(hash)
	assert.Len(t, parts, 6)
	assert.Equal(t, "argon2id", parts[1])
}

func TestVerifyPassword(t *testing.T) {
	t.Parallel()

	password := "testpassword123"
	wrongPassword := "wrongpassword"

	hash, err := HashPassword(password)
	require.NoError(t, err)

	// Test correct password
	valid, err := VerifyPassword(password, hash)
	require.NoError(t, err)
	assert.True(t, valid)

	// Test wrong password
	valid, err = VerifyPassword(wrongPassword, hash)
	require.NoError(t, err)
	assert.False(t, valid)

	// Test with empty password
	valid, err = VerifyPassword("", hash)
	require.NoError(t, err)
	assert.False(t, valid)
}

func TestVerifyPassword_InvalidHash(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		hash        string
		expectError bool
	}{
		{
			name:        "empty hash",
			hash:        "",
			expectError: true,
		},
		{
			name:        "invalid format",
			hash:        "invalid_hash",
			expectError: true,
		},
		{
			name:        "wrong algorithm",
			hash:        "$bcrypt$invalid",
			expectError: true,
		},
		{
			name:        "incomplete hash",
			hash:        "$argon2id$v=19$m=65536,t=3,p=2$salt$hash",
			expectError: false, // This should work
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			valid, err := VerifyPassword("password", tt.hash)

			if tt.expectError {
				require.Error(t, err)
				assert.False(t, valid)
			} else {
				// For valid format but wrong hash, should return false without error
				require.NoError(t, err)
				assert.False(t, valid)
			}
		})
	}
}

func TestUser_ToResponse(t *testing.T) {
	t.Parallel()

	user := &User{
		Email:    "testuser@example.com",
		Password: "hashed_password",
		Role:     "admin",
	}

	response := user.ToResponse()

	assert.Equal(t, user.Email, response.Email)
	assert.Equal(t, user.Role, response.Role)
	assert.NotEmpty(t, response.Permissions)
	// Password should not be in response (it's not included in UserResponse struct)

	// Check that permissions are properly set
	assert.NotEmpty(t, response.Permissions)

	for _, perm := range response.Permissions {
		assert.NotEmpty(t, perm.Name)
		assert.NotEmpty(t, perm.Description)
		assert.NotEmpty(t, perm.Category)
	}
}

func TestGetRolePermissions_Admin(t *testing.T) {
	t.Parallel()

	permissions := getAdminPermissions()

	// Admin should have many permissions
	assert.Greater(t, len(permissions), 15)

	// Check for specific admin permissions
	permissionNames := make(map[string]bool)
	for _, perm := range permissions {
		permissionNames[perm.Name] = true
	}

	// Admin should have user management permissions
	assert.True(t, permissionNames["users:view"])
	assert.True(t, permissionNames["users:create"])
	assert.True(t, permissionNames["users:update"])
	assert.True(t, permissionNames["users:delete"])

	// Admin should have system management permissions
	assert.True(t, permissionNames["system:manage"])
	assert.True(t, permissionNames["config:manage"])
	assert.True(t, permissionNames["dns:manage"])
}

func TestGetRolePermissions_User(t *testing.T) {
	t.Parallel()

	permissions := getUserPermissions()

	// User should have fewer permissions than admin
	assert.Less(t, len(permissions), 15)

	// Check for specific user permissions
	permissionNames := make(map[string]bool)
	for _, perm := range permissions {
		permissionNames[perm.Name] = true
	}

	// User should NOT have user management permissions
	assert.False(t, permissionNames["users:create"])
	assert.False(t, permissionNames["users:update"])
	assert.False(t, permissionNames["users:delete"])

	// User should NOT have system management permissions
	assert.False(t, permissionNames["system:manage"])
	assert.False(t, permissionNames["config:manage"])
	assert.False(t, permissionNames["dns:manage"])

	// User should have view permissions
	assert.True(t, permissionNames["system:view"])
	assert.True(t, permissionNames["devices:view"])
	assert.True(t, permissionNames["dns:view"])
}

func TestDefaultArgon2Params(t *testing.T) {
	t.Parallel()

	params := DefaultArgon2Params()

	assert.Equal(t, uint32(Argon2Memory), params.Memory)
	assert.Equal(t, uint32(Argon2Iterations), params.Iterations)
	assert.Equal(t, uint8(Argon2Parallelism), params.Parallelism)
	assert.Equal(t, uint32(Argon2SaltLength), params.SaltLength)
	assert.Equal(t, uint32(Argon2KeyLength), params.KeyLength)
}

// Helper function to split hash into parts.
func splitHash(hash string) []string {
	parts := make([]string, 0)
	current := ""

	for _, char := range hash {
		if char == '$' {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(char)
		}
	}

	if current != "" {
		parts = append(parts, current)
	}

	return parts
}
