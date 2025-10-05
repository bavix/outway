import { useState, useEffect } from 'preact/hooks';
import { authService } from '../services/authService';
import { Permission, RolePermissionsResponse } from '../providers/types.js';
import { Modal } from './Modal.js';
import { RoleBadge } from './RoleBadge.js';
import { LoadingState } from './LoadingState.js';
import { EmptyState } from './EmptyState.js';

interface PermissionsModalProps {
  isOpen: boolean;
  onClose: () => void;
  role: string;
}

export function PermissionsModal({ isOpen, onClose, role }: PermissionsModalProps) {
  const [permissions, setPermissions] = useState<Permission[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (isOpen && role) {
      loadPermissions();
    }
  }, [isOpen, role]);

  const loadPermissions = async () => {
    try {
      setIsLoading(true);
      setError('');
      const response: RolePermissionsResponse = await authService.fetchRolePermissions(role);
      setPermissions(response.permissions);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load permissions');
    } finally {
      setIsLoading(false);
    }
  };


  // Group permissions by category
  const categories = permissions.reduce((acc, perm) => {
    const category = perm.category || 'Other';
    if (!acc[category]) {
      acc[category] = [];
    }
    acc[category].push(perm);
    return acc;
  }, {} as Record<string, Permission[]>);

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={`${role.charAt(0).toUpperCase() + role.slice(1)} Permissions`}
      size="xl"
    >
        <div className="space-y-6">
          {/* Role Summary */}
          <div className="bg-gradient-to-r from-blue-50 to-indigo-50 dark:from-gray-800 dark:to-gray-700 rounded-lg p-6 border border-blue-200 dark:border-gray-600">
            <div className="flex items-center space-x-4">
              <RoleBadge role={role} size="lg" />
              <div>
                <h3 className="text-xl font-bold text-gray-900 dark:text-white capitalize">
                  {role} Role
                </h3>
                <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                  {role === 'admin' 
                    ? 'Full system access with all permissions' 
                    : 'Limited access for regular users'
                  }
                </p>
                <div className="mt-2">
                  <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300">
                    {permissions.length} permissions
                  </span>
                </div>
              </div>
            </div>
          </div>

          {/* Loading State */}
          {isLoading && (
            <LoadingState message="Loading permissions..." />
          )}

          {/* Error State */}
          {error && (
            <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
              <div className="flex items-center">
                <div className="flex-shrink-0">
                  <svg className="h-5 w-5 text-red-400" viewBox="0 0 20 20" fill="currentColor">
                    <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
                  </svg>
                </div>
                <div className="ml-3">
                  <h3 className="text-sm font-medium text-red-800 dark:text-red-200">
                    Error loading permissions
                  </h3>
                  <div className="mt-2 text-sm text-red-700 dark:text-red-300">
                    {error}
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* Permissions by Category */}
          {!isLoading && !error && Object.keys(categories).length > 0 && (
            <div className="space-y-4">
              {Object.entries(categories).map(([category, perms]) => (
                <div key={category} className="border border-gray-200 dark:border-gray-700 rounded-lg shadow-sm hover:shadow-md transition-shadow">
                  <div className="bg-gradient-to-r from-gray-50 to-gray-100 dark:from-gray-800 dark:to-gray-700 px-6 py-4 border-b border-gray-200 dark:border-gray-700">
                    <div className="flex items-center justify-between">
                      <h4 className="text-lg font-semibold text-gray-900 dark:text-white">{category}</h4>
                      <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-200 text-gray-800 dark:bg-gray-600 dark:text-gray-200">
                        {perms.length} permission{perms.length !== 1 ? 's' : ''}
                      </span>
                    </div>
                  </div>
                  <div className="p-6">
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      {perms.map((perm) => (
                        <div key={perm.name} className="flex items-start space-x-3 p-3 bg-gray-50 dark:bg-gray-800 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors">
                          <div className="flex-shrink-0 mt-1">
                            <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                          </div>
                          <div className="flex-1 min-w-0">
                            <p className="text-sm font-medium text-gray-900 dark:text-white">
                              {perm.name}
                            </p>
                            <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                              {perm.description}
                            </p>
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}

          {/* Empty State */}
          {!isLoading && !error && permissions.length === 0 && (
            <EmptyState
              title="No permissions found"
              description="This role doesn't have any permissions assigned."
              icon="shield"
            />
          )}
        </div>
    </Modal>
  );
}
