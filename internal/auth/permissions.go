package auth

import "slices"

// Permission represents a specific permission in the system.
type Permission string

const (
	// PermissionViewSystem grants access to view system information.
	PermissionViewSystem Permission = "system:view"
	// PermissionManageSystem grants access to manage system settings.
	PermissionManageSystem Permission = "system:manage"

	// PermissionViewUsers grants access to view users.
	PermissionViewUsers Permission = "users:view"
	// PermissionCreateUsers grants access to create users.
	PermissionCreateUsers Permission = "users:create"
	// PermissionUpdateUsers grants access to update users.
	PermissionUpdateUsers Permission = "users:update"
	// PermissionDeleteUsers grants access to delete users.
	PermissionDeleteUsers Permission = "users:delete"

	// PermissionViewDevices grants access to view devices.
	PermissionViewDevices Permission = "devices:view"
	// PermissionManageDevices grants access to manage devices.
	PermissionManageDevices Permission = "devices:manage"
	// PermissionWakeDevices grants access to wake devices.
	PermissionWakeDevices Permission = "devices:wake"

	// PermissionViewDNS grants access to view DNS configuration.
	PermissionViewDNS Permission = "dns:view"
	// PermissionManageDNS grants access to manage DNS configuration.
	PermissionManageDNS Permission = "dns:manage"

	// PermissionViewConfig grants access to view configuration.
	PermissionViewConfig Permission = "config:view"
	// PermissionManageConfig grants access to manage configuration.
	PermissionManageConfig Permission = "config:manage"

	// PermissionViewUpdates grants access to view updates.
	PermissionViewUpdates Permission = "updates:view"
	// PermissionManageUpdates grants access to manage updates.
	PermissionManageUpdates Permission = "updates:manage"

	// PermissionViewCache grants access to view cache.
	PermissionViewCache Permission = "cache:view"
	// PermissionManageCache grants access to manage cache.
	PermissionManageCache Permission = "cache:manage"

	// PermissionViewHistory grants access to view DNS history.
	PermissionViewHistory Permission = "history:view"

	// PermissionViewStats grants access to view statistics.
	PermissionViewStats Permission = "stats:view"

	// PermissionViewOverview grants access to view overview.
	PermissionViewOverview Permission = "overview:view"

	// PermissionViewInfo grants access to view system info.
	PermissionViewInfo Permission = "info:view"

	// PermissionViewResolve grants access to test DNS resolution.
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
	return slices.Contains(r.Permissions, permission)
}

// HasAnyPermission checks if a role has any of the specified permissions.
func (r *Role) HasAnyPermission(permissions ...Permission) bool {
	return slices.ContainsFunc(permissions, r.HasPermission)
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
