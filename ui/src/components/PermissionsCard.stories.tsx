import type { Meta, StoryObj } from '@storybook/preact';
import { PermissionsCard } from './PermissionsCard.js';
import { Permission } from '../providers/types.js';

const meta: Meta<typeof PermissionsCard> = {
  title: 'Components/PermissionsCard',
  component: PermissionsCard,
  parameters: {
    layout: 'padded',
  },
};

export default meta;
type Story = StoryObj<typeof PermissionsCard>;

const systemPermissions: Permission[] = [
  {
    name: 'system:view',
    description: 'View system information',
    category: 'System',
  },
  {
    name: 'system:manage',
    description: 'Manage system settings',
    category: 'System',
  },
  {
    name: 'system:restart',
    description: 'Restart system services',
    category: 'System',
  },
];

const userPermissions: Permission[] = [
  {
    name: 'users:view',
    description: 'View users',
    category: 'Users',
  },
  {
    name: 'users:create',
    description: 'Create users',
    category: 'Users',
  },
  {
    name: 'users:update',
    description: 'Update users',
    category: 'Users',
  },
  {
    name: 'users:delete',
    description: 'Delete users',
    category: 'Users',
  },
];

const devicePermissions: Permission[] = [
  {
    name: 'devices:view',
    description: 'View devices',
    category: 'Devices',
  },
  {
    name: 'devices:manage',
    description: 'Manage devices',
    category: 'Devices',
  },
  {
    name: 'devices:wake',
    description: 'Wake devices',
    category: 'Devices',
  },
  {
    name: 'devices:scan',
    description: 'Scan for devices',
    category: 'Devices',
  },
];

const dnsPermissions: Permission[] = [
  {
    name: 'dns:view',
    description: 'View DNS settings',
    category: 'DNS',
  },
  {
    name: 'dns:manage',
    description: 'Manage DNS settings',
    category: 'DNS',
  },
];

export const SystemPermissions: Story = {
  args: {
    category: 'System',
    permissions: systemPermissions,
  },
};

export const UserPermissions: Story = {
  args: {
    category: 'Users',
    permissions: userPermissions,
  },
};

export const DevicePermissions: Story = {
  args: {
    category: 'Devices',
    permissions: devicePermissions,
  },
};

export const DnsPermissions: Story = {
  args: {
    category: 'DNS',
    permissions: dnsPermissions,
  },
};

export const EmptyPermissions: Story = {
  args: {
    category: 'Empty Category',
    permissions: [],
  },
};

export const SinglePermission: Story = {
  args: {
    category: 'Single Permission',
    permissions: [
      {
        name: 'single:permission',
        description: 'This is a single permission',
        category: 'Single Permission',
      },
    ],
  },
};

export const ManyPermissions: Story = {
  args: {
    category: 'Many Permissions',
    permissions: [
      ...systemPermissions,
      ...userPermissions,
      ...devicePermissions,
      ...dnsPermissions,
    ],
  },
};

export const WithCustomClass: Story = {
  args: {
    category: 'Custom Styled',
    permissions: userPermissions,
    className: 'ring-2 ring-blue-500',
  },
};

export const AllCategories: Story = {
  render: () => (
    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
      <PermissionsCard
        title="System Permissions"
        permissions={systemPermissions}
      />
      <PermissionsCard
        title="User Permissions"
        permissions={userPermissions}
      />
      <PermissionsCard
        title="Device Permissions"
        permissions={devicePermissions}
      />
      <PermissionsCard
        title="DNS Permissions"
        permissions={dnsPermissions}
      />
    </div>
  ),
};

export const AdminRolePermissions: Story = {
  render: () => (
    <div className="max-w-4xl mx-auto">
      <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-6">
        Administrator Role Permissions
      </h3>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <PermissionsCard
          title="System Management Permissions"
          permissions={systemPermissions}
        />
        <PermissionsCard
          title="User Management Permissions"
          permissions={userPermissions}
        />
        <PermissionsCard
          title="Device Management Permissions"
          permissions={devicePermissions}
        />
        <PermissionsCard
          title="DNS Management Permissions"
          permissions={dnsPermissions}
        />
      </div>
    </div>
  ),
};

export const UserRolePermissions: Story = {
  render: () => (
    <div className="max-w-4xl mx-auto">
      <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-6">
        User Role Permissions
      </h3>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <PermissionsCard
          title="System Permissions"
          permissions={[
            {
              name: 'system:view',
              description: 'View system information',
              category: 'System',
            },
          ]}
        />
        <PermissionsCard
          title="Device Permissions"
          permissions={[
            {
              name: 'devices:view',
              description: 'View devices',
              category: 'Devices',
            },
            {
              name: 'devices:wake',
              description: 'Wake devices',
              category: 'Devices',
            },
          ]}
        />
      </div>
    </div>
  ),
};
