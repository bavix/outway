import type { Meta, StoryObj } from '@storybook/preact';
import { GravatarAvatar } from './GravatarAvatar.js';

const meta: Meta<typeof GravatarAvatar> = {
  title: 'Components/GravatarAvatar',
  component: GravatarAvatar,
  parameters: {
    layout: 'centered',
  },
  argTypes: {
    variant: {
      control: 'select',
      options: ['primary', 'secondary', 'danger'],
    },
    size: {
      control: 'select',
      options: ['sm', 'md', 'lg', 'xl'],
    },
    fallbackToInitials: {
      control: 'boolean',
    },
  },
};

export default meta;
type Story = StoryObj<typeof GravatarAvatar>;

export const Default: Story = {
  args: {
    email: 'user@example.com',
  },
};

export const Admin: Story = {
  args: {
    email: 'admin@example.com',
    variant: 'danger',
  },
};

export const User: Story = {
  args: {
    email: 'user@example.com',
    variant: 'primary',
  },
};

export const Small: Story = {
  args: {
    email: 'user@example.com',
    size: 'sm',
  },
};

export const Medium: Story = {
  args: {
    email: 'user@example.com',
    size: 'md',
  },
};

export const Large: Story = {
  args: {
    email: 'user@example.com',
    size: 'lg',
  },
};

export const ExtraLarge: Story = {
  args: {
    email: 'user@example.com',
    size: 'xl',
  },
};

export const WithoutFallback: Story = {
  args: {
    email: 'user@example.com',
    fallbackToInitials: false,
  },
};

export const WithCustomClass: Story = {
  args: {
    email: 'admin@example.com',
    className: 'ring-2 ring-red-500',
  },
};

export const AllSizes: Story = {
  render: () => (
    <div className="flex items-center gap-4">
      <GravatarAvatar email="user@example.com" size="sm" />
      <GravatarAvatar email="user@example.com" size="md" />
      <GravatarAvatar email="user@example.com" size="lg" />
      <GravatarAvatar email="user@example.com" size="xl" />
    </div>
  ),
};

export const AllVariants: Story = {
  render: () => (
    <div className="flex items-center gap-4">
      <GravatarAvatar email="user@example.com" variant="primary" />
      <GravatarAvatar email="user@example.com" variant="secondary" />
      <GravatarAvatar email="user@example.com" variant="danger" />
    </div>
  ),
};

export const DifferentEmails: Story = {
  render: () => (
    <div className="flex items-center gap-4">
      <GravatarAvatar email="john.doe@example.com" />
      <GravatarAvatar email="jane.smith@example.com" />
      <GravatarAvatar email="bob.johnson@example.com" />
      <GravatarAvatar email="alice.brown@example.com" />
    </div>
  ),
};

export const InUserList: Story = {
  render: () => (
    <div className="max-w-md mx-auto space-y-3">
      <div className="flex items-center justify-between p-3 bg-white dark:bg-gray-800 rounded-lg shadow">
        <div className="flex items-center space-x-3">
          <GravatarAvatar email="admin@example.com" size="md" variant="danger" />
          <div>
            <p className="text-sm font-medium text-gray-900 dark:text-white">Admin User</p>
            <p className="text-xs text-gray-500 dark:text-gray-400">admin@example.com</p>
          </div>
        </div>
      </div>
      
      <div className="flex items-center justify-between p-3 bg-white dark:bg-gray-800 rounded-lg shadow">
        <div className="flex items-center space-x-3">
          <GravatarAvatar email="user@example.com" size="md" variant="primary" />
          <div>
            <p className="text-sm font-medium text-gray-900 dark:text-white">Regular User</p>
            <p className="text-xs text-gray-500 dark:text-gray-400">user@example.com</p>
          </div>
        </div>
      </div>
      
      <div className="flex items-center justify-between p-3 bg-white dark:bg-gray-800 rounded-lg shadow">
        <div className="flex items-center space-x-3">
          <GravatarAvatar email="guest@example.com" size="md" variant="secondary" />
          <div>
            <p className="text-sm font-medium text-gray-900 dark:text-white">Guest User</p>
            <p className="text-xs text-gray-500 dark:text-gray-400">guest@example.com</p>
          </div>
        </div>
      </div>
    </div>
  ),
};

export const FallbackBehavior: Story = {
  render: () => (
    <div className="space-y-4">
      <div>
        <h4 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
          With Gravatar Fallback
        </h4>
        <GravatarAvatar email="user@example.com" fallbackToInitials />
      </div>
      
      <div>
        <h4 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
          Without Gravatar Fallback
        </h4>
        <GravatarAvatar email="user@example.com" fallbackToInitials={false} />
      </div>
    </div>
  ),
};
