import type { Meta, StoryObj } from '@storybook/preact';
import { RoleBadge } from './RoleBadge.js';

const meta: Meta<typeof RoleBadge> = {
  title: 'Components/RoleBadge',
  component: RoleBadge,
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
      options: ['sm', 'md', 'lg'],
    },
    showIcon: {
      control: 'boolean',
    },
  },
};

export default meta;
type Story = StoryObj<typeof RoleBadge>;

export const Admin: Story = {
  args: {
    role: 'admin',
  },
};

export const User: Story = {
  args: {
    role: 'user',
  },
};

export const Guest: Story = {
  args: {
    role: 'guest',
  },
};

export const Moderator: Story = {
  args: {
    role: 'moderator',
  },
};

export const Small: Story = {
  args: {
    role: 'admin',
    size: 'sm',
  },
};

export const Medium: Story = {
  args: {
    role: 'user',
    size: 'md',
  },
};

export const Large: Story = {
  args: {
    role: 'admin',
    size: 'lg',
  },
};

export const WithoutIcon: Story = {
  args: {
    role: 'admin',
    showIcon: false,
  },
};

export const WithCustomClass: Story = {
  args: {
    role: 'admin',
    className: 'ring-2 ring-red-500',
  },
};

export const AllRoles: Story = {
  render: () => (
    <div className="flex flex-wrap gap-4">
      <RoleBadge role="admin" />
      <RoleBadge role="user" />
      <RoleBadge role="guest" />
      <RoleBadge role="moderator" />
    </div>
  ),
};

export const AllSizes: Story = {
  render: () => (
    <div className="flex items-center gap-4">
      <RoleBadge role="admin" size="sm" />
      <RoleBadge role="user" size="md" />
      <RoleBadge role="admin" size="lg" />
    </div>
  ),
};

export const InUserList: Story = {
  render: () => (
    <div className="max-w-md mx-auto space-y-3">
      <div className="flex items-center justify-between p-3 bg-white dark:bg-gray-800 rounded-lg shadow">
        <div className="flex items-center space-x-3">
          <div className="w-8 h-8 bg-red-500 rounded-full flex items-center justify-center text-white text-sm font-medium">
            A
          </div>
          <div>
            <p className="text-sm font-medium text-gray-900 dark:text-white">Admin User</p>
            <p className="text-xs text-gray-500 dark:text-gray-400">admin@example.com</p>
          </div>
        </div>
        <RoleBadge role="admin" size="sm" />
      </div>
      
      <div className="flex items-center justify-between p-3 bg-white dark:bg-gray-800 rounded-lg shadow">
        <div className="flex items-center space-x-3">
          <div className="w-8 h-8 bg-blue-500 rounded-full flex items-center justify-center text-white text-sm font-medium">
            U
          </div>
          <div>
            <p className="text-sm font-medium text-gray-900 dark:text-white">Regular User</p>
            <p className="text-xs text-gray-500 dark:text-gray-400">user@example.com</p>
          </div>
        </div>
        <RoleBadge role="user" size="sm" />
      </div>
      
      <div className="flex items-center justify-between p-3 bg-white dark:bg-gray-800 rounded-lg shadow">
        <div className="flex items-center space-x-3">
          <div className="w-8 h-8 bg-gray-500 rounded-full flex items-center justify-center text-white text-sm font-medium">
            G
          </div>
          <div>
            <p className="text-sm font-medium text-gray-900 dark:text-white">Guest User</p>
            <p className="text-xs text-gray-500 dark:text-gray-400">guest@example.com</p>
          </div>
        </div>
        <RoleBadge role="guest" size="sm" />
      </div>
    </div>
  ),
};
