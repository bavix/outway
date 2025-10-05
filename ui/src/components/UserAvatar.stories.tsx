import type { Meta, StoryObj } from '@storybook/preact';
import { UserAvatar } from './UserAvatar.js';

const meta: Meta<typeof UserAvatar> = {
  title: 'Components/UserAvatar',
  component: UserAvatar,
  parameters: {
    layout: 'centered',
  },
  argTypes: {
    role: {
      control: 'select',
      options: ['admin', 'user', 'guest', 'moderator'],
    },
    size: {
      control: 'select',
      options: ['sm', 'md', 'lg', 'xl'],
    },
    showRole: {
      control: 'boolean',
    },
  },
};

export default meta;
type Story = StoryObj<typeof UserAvatar>;

export const Admin: Story = {
  args: {
    email: 'admin@example.com',
    role: 'admin',
  },
};

export const User: Story = {
  args: {
    email: 'user@example.com',
    role: 'user',
  },
};

export const Guest: Story = {
  args: {
    email: 'guest@example.com',
    role: 'guest',
  },
};

export const WithoutRole: Story = {
  args: {
    email: 'user@example.com',
  },
};

export const Small: Story = {
  args: {
    email: 'admin@example.com',
    role: 'admin',
    size: 'sm',
  },
};

export const Medium: Story = {
  args: {
    email: 'user@example.com',
    role: 'user',
    size: 'md',
  },
};

export const Large: Story = {
  args: {
    email: 'admin@example.com',
    role: 'admin',
    size: 'lg',
  },
};

export const ExtraLarge: Story = {
  args: {
    email: 'user@example.com',
    role: 'user',
    size: 'xl',
  },
};

export const WithRoleLabel: Story = {
  args: {
    email: 'admin@example.com',
    role: 'admin',
    showRole: true,
  },
};

export const WithCustomClass: Story = {
  args: {
    email: 'admin@example.com',
    role: 'admin',
    className: 'ring-2 ring-red-500',
  },
};

export const AllRoles: Story = {
  render: () => (
    <div className="flex flex-wrap gap-4">
      <UserAvatar email="admin@example.com" role="admin" />
      <UserAvatar email="user@example.com" role="user" />
      <UserAvatar email="guest@example.com" role="guest" />
      <UserAvatar email="moderator@example.com" role="moderator" />
    </div>
  ),
};

export const AllSizes: Story = {
  render: () => (
    <div className="flex items-center gap-4">
      <UserAvatar email="user@example.com" size="sm" />
      <UserAvatar email="user@example.com" size="md" />
      <UserAvatar email="user@example.com" size="lg" />
      <UserAvatar email="user@example.com" size="xl" />
    </div>
  ),
};

export const WithRoleLabels: Story = {
  render: () => (
    <div className="space-y-4">
      <UserAvatar email="admin@example.com" role="admin" showRole />
      <UserAvatar email="user@example.com" role="user" showRole />
      <UserAvatar email="guest@example.com" role="guest" showRole />
    </div>
  ),
};

export const InUserList: Story = {
  render: () => (
    <div className="max-w-md mx-auto space-y-3">
      <div className="flex items-center justify-between p-3 bg-white dark:bg-gray-800 rounded-lg shadow">
        <div className="flex items-center space-x-3">
          <UserAvatar email="admin@example.com" role="admin" size="md" />
          <div>
            <p className="text-sm font-medium text-gray-900 dark:text-white">Admin User</p>
            <p className="text-xs text-gray-500 dark:text-gray-400">admin@example.com</p>
          </div>
        </div>
      </div>
      
      <div className="flex items-center justify-between p-3 bg-white dark:bg-gray-800 rounded-lg shadow">
        <div className="flex items-center space-x-3">
          <UserAvatar email="user@example.com" role="user" size="md" />
          <div>
            <p className="text-sm font-medium text-gray-900 dark:text-white">Regular User</p>
            <p className="text-xs text-gray-500 dark:text-gray-400">user@example.com</p>
          </div>
        </div>
      </div>
      
      <div className="flex items-center justify-between p-3 bg-white dark:bg-gray-800 rounded-lg shadow">
        <div className="flex items-center space-x-3">
          <UserAvatar email="guest@example.com" role="guest" size="md" />
          <div>
            <p className="text-sm font-medium text-gray-900 dark:text-white">Guest User</p>
            <p className="text-xs text-gray-500 dark:text-gray-400">guest@example.com</p>
          </div>
        </div>
      </div>
    </div>
  ),
};
