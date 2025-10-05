import { useState, useEffect } from 'preact/hooks';
import { authService } from '../services/authService';
import { Role, RolePermissionsResponse } from '../providers/types.js';
import { Card } from './Card.js';
import { Button } from './Button.js';
import { StatusCard } from './StatusCard.js';
import { LoadingState } from './LoadingState.js';

export function RolesOverview() {
  const [roles, setRoles] = useState<Role[]>([]);
  const [selectedRole, setSelectedRole] = useState<string | null>(null);
  const [rolePermissions, setRolePermissions] = useState<RolePermissionsResponse | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');

  const loadRoles = async () => {
    try {
      setIsLoading(true);
      const response = await authService.fetchRoles();
      setRoles(response.roles);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load roles');
    } finally {
      setIsLoading(false);
    }
  };

  const loadRolePermissions = async (role: string) => {
    try {
      const response = await authService.fetchRolePermissions(role);
      setRolePermissions(response);
      setSelectedRole(role);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load role permissions');
    }
  };

  useEffect(() => {
    if (authService.state.isAuthenticated) {
      loadRoles();
    }
  }, [authService.state.isAuthenticated]);

  if (!authService.state.isAuthenticated) {
    return (
      <div className="flex justify-center items-center h-64">
        <div className="text-gray-500 dark:text-gray-400">Please log in to view roles.</div>
      </div>
    );
  }

  if (isLoading) {
    return <LoadingState message="Loading roles..." />;
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Roles & Permissions</h1>
          <p className="text-sm text-gray-600 dark:text-gray-400">
            Overview of system roles and their associated permissions
          </p>
        </div>
      </div>

      {/* Error Message */}
      {error && (
        <Card className="border-red-200 bg-red-50 dark:bg-red-900/20 dark:border-red-800">
          <div className="flex items-center">
            <svg className="h-5 w-5 text-red-400 mr-2" viewBox="0 0 20 20" fill="currentColor">
              <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.28 7.22a.75.75 0 00-1.06 1.06L8.94 10l-1.72 1.72a.75.75 0 101.06 1.06L10 11.06l1.72 1.72a.75.75 0 101.06-1.06L11.06 10l1.72-1.72a.75.75 0 00-1.06-1.06L10 8.94l-1.72-1.72z" clipRule="evenodd" />
            </svg>
            <p className="text-sm font-medium text-red-800 dark:text-red-200">{error}</p>
          </div>
        </Card>
      )}

      {/* Roles Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {roles.map((role) => {
          const getRoleIcon = (roleName: string) => {
            if (roleName === 'admin') {
              return (
                <svg className="w-6 h-6 text-red-600 dark:text-red-400" viewBox="0 0 20 20" fill="currentColor">
                  <path fillRule="evenodd" d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z" clipRule="evenodd" />
                </svg>
              );
            }
            return (
              <svg className="w-6 h-6 text-blue-600 dark:text-blue-400" viewBox="0 0 20 20" fill="currentColor">
                <path fillRule="evenodd" d="M10 9a3 3 0 100-6 3 3 0 000 6zm-7 9a7 7 0 1114 0H3z" clipRule="evenodd" />
              </svg>
            );
          };

          return (
            <StatusCard
              key={role.name}
              title={role.name.charAt(0).toUpperCase() + role.name.slice(1)}
              description={role.description}
              count={role.permissions_count}
              icon={getRoleIcon(role.name)}
              status={selectedRole === role.name ? 'active' : 'inactive'}
              onClick={() => loadRolePermissions(role.name)}
            />
          );
        })}
      </div>

      {/* Role Details */}
      {selectedRole && rolePermissions && (
        <Card className="max-w-6xl mx-auto">
          <div className="p-6">
            <div className="flex items-center justify-between mb-6">
              <div className="flex items-center space-x-3">
                <div className={`p-3 rounded-lg ${
                  selectedRole === 'admin' 
                    ? 'bg-red-100 dark:bg-red-900/30' 
                    : 'bg-blue-100 dark:bg-blue-900/30'
                }`}>
                  {selectedRole === 'admin' ? (
                    <svg className="w-8 h-8 text-red-600 dark:text-red-400" viewBox="0 0 20 20" fill="currentColor">
                      <path fillRule="evenodd" d="M9.504 1.132a1 1 0 01.992 0l1.75 1a1 1 0 11-.992 1.736L10 3.152l-1.254.716a1 1 0 11-.992-1.736l1.75-1zM5.618 4.504a1 1 0 01-.372 1.364L5.016 6l.23.132a1 1 0 11-.992 1.736L3.667 8.5l-.23.132a1 1 0 01-.992-1.736L2.5 6.5l-.23-.132a1 1 0 01.992-1.736L3.5 4.5l.23-.132a1 1 0 011.364.372zm8.764 0a1 1 0 011.364-.372l.23.132.23-.132a1 1 0 011.364.372l.132.23-.132.23a1 1 0 01-1.736.992L16.5 6.5l.132.23a1 1 0 01-1.736.992L14.5 6.5l-.132-.23a1 1 0 01.992-1.736L15.5 4.5l-.132-.23a1 1 0 01-.372-1.364zm-7 4a1 1 0 011.364-.372L7.5 9l.23.132a1 1 0 11-.992 1.736L5.667 11.5l-.23.132a1 1 0 01-.992-1.736L4.5 10.5l-.23-.132a1 1 0 01.992-1.736L5.5 8.5l.23-.132a1 1 0 01.364-.372zm7 0a1 1 0 01.364.372l.23.132.23-.132a1 1 0 01.992 1.736L16.5 10.5l.132.23a1 1 0 01-1.736.992L14.5 10.5l-.132-.23a1 1 0 01.992-1.736L15.5 8.5l.132-.23a1 1 0 01.372-.364zM9.504 13.132a1 1 0 01.992 0l1.75 1a1 1 0 11-.992 1.736L10 15.152l-1.254.716a1 1 0 11-.992-1.736l1.75-1z" clipRule="evenodd" />
                    </svg>
                  ) : (
                    <svg className="w-8 h-8 text-blue-600 dark:text-blue-400" viewBox="0 0 20 20" fill="currentColor">
                      <path fillRule="evenodd" d="M10 9a3 3 0 100-6 3 3 0 000 6zm-7 9a7 7 0 1114 0H3z" clipRule="evenodd" />
                    </svg>
                  )}
                </div>
                <div>
                  <h2 className="text-2xl font-bold text-gray-900 dark:text-white capitalize">
                    {selectedRole} Role
                  </h2>
                  <p className="text-gray-600 dark:text-gray-400">
                    {rolePermissions.count} permissions across {Object.keys(rolePermissions.categories).length} categories
                  </p>
                </div>
              </div>
              <Button
                variant="outline"
                onClick={() => setSelectedRole(null)}
              >
                Close
              </Button>
            </div>

            {/* Permissions by Category */}
            <div className="space-y-6">
              {Object.entries(rolePermissions.categories).map(([category, permissions]) => (
                <div key={category} className="border border-gray-200 dark:border-gray-700 rounded-lg">
                  <div className="bg-gray-100 dark:bg-gray-800 px-4 py-3 border-b border-gray-200 dark:border-gray-700">
                    <h3 className="text-lg font-semibold text-gray-900 dark:text-white">{category}</h3>
                    <p className="text-sm text-gray-600 dark:text-gray-400">
                      {permissions.length} permission{permissions.length !== 1 ? 's' : ''}
                    </p>
                  </div>
                  <div className="p-4">
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                      {permissions.map((permission) => (
                        <div key={permission.name} className="flex items-start space-x-3 p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
                          <div className="flex-shrink-0 mt-1">
                            <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                          </div>
                          <div className="flex-1 min-w-0">
                            <p className="text-sm font-medium text-gray-900 dark:text-white">
                              {permission.name}
                            </p>
                            <p className="text-sm text-gray-600 dark:text-gray-400">
                              {permission.description}
                            </p>
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </Card>
      )}
    </div>
  );
}
