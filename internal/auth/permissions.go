package auth

// Permission represents a specific permission in the system.
type Permission string

const (
	// System permissions.
	PermissionViewSystem   Permission = "system:view"
	PermissionManageSystem Permission = "system:manage"

	// User management permissions.
	PermissionViewUsers   Permission = "users:view"
	PermissionCreateUsers Permission = "users:create"
	PermissionUpdateUsers Permission = "users:update"
	PermissionDeleteUsers Permission = "users:delete"

	// Device management permissions.
	PermissionViewDevices   Permission = "devices:view"
	PermissionManageDevices Permission = "devices:manage"
	PermissionWakeDevices   Permission = "devices:wake"

	// DNS management permissions.
	PermissionViewDNS   Permission = "dns:view"
	PermissionManageDNS Permission = "dns:manage"

	// Configuration permissions.
	PermissionViewConfig   Permission = "config:view"
	PermissionManageConfig Permission = "config:manage"

	// Update permissions.
	PermissionViewUpdates   Permission = "updates:view"
	PermissionManageUpdates Permission = "updates:manage"

	// Cache permissions.
	PermissionViewCache   Permission = "cache:view"
	PermissionManageCache Permission = "cache:manage"

	// History permissions.
	PermissionViewHistory Permission = "history:view"

	// Statistics permissions.
	PermissionViewStats Permission = "stats:view"

	// Overview permissions.
	PermissionViewOverview Permission = "overview:view"

	// Info permissions.
	PermissionViewInfo Permission = "info:view"

	// Resolve permissions.
	PermissionViewResolve Permission = "resolve:view"
)

// Role represents a user role with associated permissions.
type Role struct {
	Name        string       `json:"name"`
	Permissions []Permission `json:"permissions"`
}

// GetRoleAdmin returns the admin role.
func GetRoleAdmin() Role {
	return Role{
		Name: "admin",
		Permissions: []Permission{
			PermissionViewSystem,
			PermissionManageSystem,
			PermissionViewUsers,
			PermissionCreateUsers,
			PermissionUpdateUsers,
			PermissionDeleteUsers,
			PermissionViewDevices,
			PermissionManageDevices,
			PermissionWakeDevices,
			PermissionViewDNS,
			PermissionManageDNS,
			PermissionViewConfig,
			PermissionManageConfig,
			PermissionViewUpdates,
			PermissionManageUpdates,
			PermissionViewCache,
			PermissionManageCache,
			PermissionViewHistory,
			PermissionViewStats,
			PermissionViewOverview,
			PermissionViewInfo,
			PermissionViewResolve,
		},
	}
}

// GetRoleUser returns the user role.
func GetRoleUser() Role {
	return Role{
		Name: "user",
		Permissions: []Permission{
			PermissionViewSystem,    // View system overview
			PermissionViewDevices,   // View devices
			PermissionManageDevices, // Manage devices (scan, refresh)
			PermissionWakeDevices,   // Wake devices (Wake-on-LAN)
			PermissionViewDNS,       // Test DNS resolution
			PermissionViewConfig,    // View configuration (safe version)
			PermissionViewUpdates,   // View updates
			PermissionViewCache,     // View cache
			PermissionViewHistory,   // View DNS history
			PermissionViewStats,     // View statistics
			PermissionViewOverview,  // View overview
			PermissionViewInfo,      // View system info
			PermissionViewResolve,   // Test DNS resolution
		},
	}
}

// GetRole returns a role by name.
func GetRole(name string) *Role {
	switch name {
	case "admin":
		role := GetRoleAdmin()

		return &role
	case "user":
		role := GetRoleUser()

		return &role
	default:
		role := GetRoleUser() // Default to user role for unknown roles

		return &role
	}
}

// HasPermission checks if a role has a specific permission.
func (r *Role) HasPermission(permission Permission) bool {
	for _, p := range r.Permissions {
		if p == permission {
			return true
		}
	}

	return false
}

// HasAnyPermission checks if a role has any of the specified permissions.
func (r *Role) HasAnyPermission(permissions ...Permission) bool {
	for _, permission := range permissions {
		if r.HasPermission(permission) {
			return true
		}
	}

	return false
}

// HasAllPermissions checks if a role has all of the specified permissions.
func (r *Role) HasAllPermissions(permissions ...Permission) bool {
	for _, permission := range permissions {
		if !r.HasPermission(permission) {
			return false
		}
	}

	return true
}
