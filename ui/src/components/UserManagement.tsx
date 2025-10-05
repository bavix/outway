import { useState, useEffect } from 'preact/hooks';
import { authService } from '../services/authService';
import { UserResponse, UserRequest, Role } from '../providers/types.js';
import { Card } from './Card.js';
import { Button } from './Button.js';
import { Table, THead, TBody, TRow, TH, TD } from './Table.js';
import { Modal } from './Modal.js';
import { Input } from './Input.js';
import { PermissionsDisplay } from './PermissionsDisplay.js';
import { PermissionsModal } from './PermissionsModal.js';
import { UserAvatar } from './UserAvatar.js';

export function UserManagement() {
  const [users, setUsers] = useState<UserResponse[]>([]);
  const [roles, setRoles] = useState<Role[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [editingUser, setEditingUser] = useState<string | null>(null);
  const [showPermissionsModal, setShowPermissionsModal] = useState(false);
  const [selectedRoleForPermissions, setSelectedRoleForPermissions] = useState<string>('');

  const loadUsers = async () => {
    try {
      setIsLoading(true);
      const response = await authService.fetchUsers();
      setUsers(response.users);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load users');
    } finally {
      setIsLoading(false);
    }
  };

  const loadRoles = async () => {
    try {
      const response = await authService.fetchRoles();
      setRoles(response.roles);
    } catch (err) {
      console.error('Failed to load roles:', err);
    }
  };

  useEffect(() => {
    // Only load users if authService is ready and user is authenticated
    if (authService.state.isAuthenticated) {
      loadUsers();
      loadRoles();
    }
  }, [authService.state.isAuthenticated]);

  const handleCreateUser = async (userData: UserRequest) => {
    try {
      await authService.createUser(userData);
      setShowCreateForm(false);
      await loadUsers();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create user');
    }
  };

  const handleUpdateUser = async (email: string, userData: UserRequest) => {
    try {
      await authService.updateUser(email, userData);
      setEditingUser(null);
      await loadUsers();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update user');
    }
  };

  const handleDeleteUser = async (email: string) => {
    if (!confirm(`Are you sure you want to delete user "${email}"?`)) {
      return;
    }

    try {
      await authService.deleteUser(email);
      await loadUsers();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete user');
    }
  };

  const handleViewPermissions = (role: string) => {
    setSelectedRoleForPermissions(role);
    setShowPermissionsModal(true);
  };

  // Check if user is authenticated
  if (!authService.state.isAuthenticated) {
    return (
      <div className="flex justify-center items-center h-64">
        <div className="text-gray-500 dark:text-gray-400">Please log in to manage users.</div>
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="flex justify-center items-center h-64">
        <div className="text-gray-500 dark:text-gray-400">Loading users...</div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">User Management</h1>
          <p className="text-sm text-gray-600 dark:text-gray-400">Manage system users and their permissions</p>
        </div>
        <Button
          onClick={() => setShowCreateForm(true)}
          variant="primary"
        >
          Add User
        </Button>
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

      {showCreateForm && (
        <UserForm
          roles={roles}
          onSubmit={handleCreateUser}
          onCancel={() => setShowCreateForm(false)}
        />
      )}

      {/* Users Table */}
      <Card>
        <Table>
          <THead>
            <TRow>
              <TH>User</TH>
              <TH>Role & Permissions</TH>
              <TH>Actions</TH>
            </TRow>
          </THead>
          <TBody>
            {users.map((user) => (
              <TRow key={user.email}>
                <TD>
                  <div className="flex items-center">
                    <UserAvatar 
                      email={user.email} 
                      role={user.role} 
                      size="md" 
                      className="mr-3"
                    />
                    <div>
                      <div className="text-sm font-medium text-gray-900 dark:text-white">{user.email}</div>
                      <div className="text-xs text-gray-500 dark:text-gray-400">
                        {user.permissions?.length || 0} permissions
                      </div>
                    </div>
                  </div>
                </TD>
                <TD>
                  <PermissionsDisplay 
                    role={user.role} 
                    permissions={user.permissions || []}
                    showDetails={true}
                    onViewPermissions={() => handleViewPermissions(user.role)}
                  />
                </TD>
                <TD>
                  <div className="flex space-x-2">
                    <Button
                      onClick={() => setEditingUser(user.email)}
                      variant="outline"
                      size="sm"
                    >
                      Edit
                    </Button>
                    <Button
                      onClick={() => handleDeleteUser(user.email)}
                      variant="outline"
                      size="sm"
                      className="text-red-600 hover:text-red-700 border-red-300 hover:border-red-400"
                    >
                      Delete
                    </Button>
                  </div>
                </TD>
              </TRow>
            ))}
          </TBody>
        </Table>
      </Card>

      {editingUser && (
        <UserForm
          user={users.find(u => u.email === editingUser) || undefined}
          roles={roles}
          onSubmit={(userData) => handleUpdateUser(editingUser, userData)}
          onCancel={() => setEditingUser(null)}
        />
      )}

      {/* Permissions Modal */}
      <PermissionsModalWrapper
        isOpen={showPermissionsModal}
        onClose={() => setShowPermissionsModal(false)}
        role={selectedRoleForPermissions}
      />
    </div>
  );
}

interface UserFormProps {
  user?: UserResponse | undefined;
  roles?: Role[];
  onSubmit: (userData: UserRequest) => void;
  onCancel: () => void;
}

function UserForm({ user, roles = [], onSubmit, onCancel }: UserFormProps) {
  const [email, setEmail] = useState(user?.email || '');
  const [password, setPassword] = useState('');
  const [role, setRole] = useState(user?.role || 'user');

  // Update role when user changes
  useEffect(() => {
    if (user?.role) {
      setRole(user.role);
    }
  }, [user?.role]);
  const [isLoading, setIsLoading] = useState(false);

  const handleSubmit = async (e: Event) => {
    e.preventDefault();
    setIsLoading(true);

    try {
      await onSubmit({ email, password, role });
    } catch (err) {
      console.error('Form submission error:', err);
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <Modal
      isOpen={true}
      onClose={onCancel}
      title={user ? 'Edit User' : 'Create User'}
    >
      <form onSubmit={handleSubmit} className="space-y-4">
        <Input
          label="Email"
          type="email"
          value={email}
          onInput={(e) => setEmail((e.target as HTMLInputElement).value)}
          placeholder="Enter email"
          required
          disabled={isLoading}
        />
        
        <Input
          label="Password"
          type="password"
          value={password}
          onInput={(e) => setPassword((e.target as HTMLInputElement).value)}
          placeholder={user ? "Leave empty to keep current password" : "Enter password"}
          required={!user}
          disabled={isLoading}
        />
        
        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            Role
          </label>
          <select
            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 dark:bg-gray-700 dark:text-white sm:text-sm"
            value={role}
            onChange={(e) => setRole((e.target as HTMLSelectElement).value)}
            disabled={isLoading}
          >
            {roles.map((roleOption) => (
              <option key={roleOption.name} value={roleOption.name}>
                {roleOption.name.charAt(0).toUpperCase() + roleOption.name.slice(1)} - {roleOption.description}
              </option>
            ))}
          </select>
          {roles.length > 0 && (
            <div className="mt-2">
              <PermissionsDisplay 
                role={role} 
                showDetails={false}
                className="text-xs"
              />
            </div>
          )}
        </div>
        
        <div className="flex justify-end space-x-3 pt-4">
          <Button
            type="button"
            onClick={onCancel}
            variant="outline"
            disabled={isLoading}
          >
            Cancel
          </Button>
          <Button
            type="submit"
            variant="primary"
            loading={isLoading}
            disabled={isLoading}
          >
            {user ? 'Update User' : 'Create User'}
          </Button>
        </div>
      </form>
    </Modal>
  );
}

// Permissions Modal Component
function PermissionsModalWrapper({ 
  isOpen, 
  onClose, 
  role 
}: { 
  isOpen: boolean; 
  onClose: () => void; 
  role: string; 
}) {
  return (
    <PermissionsModal
      isOpen={isOpen}
      onClose={onClose}
      role={role}
    />
  );
}
