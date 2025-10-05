import type { Meta, StoryObj } from '@storybook/preact';
import { StatusIndicator } from './StatusIndicator.js';

const meta: Meta<typeof StatusIndicator> = {
  title: 'Components/StatusIndicator',
  component: StatusIndicator,
  parameters: {
    layout: 'centered',
  },
  argTypes: {
    status: {
      control: 'select',
      options: ['online', 'offline', 'busy', 'away', 'error', 'warning', 'success'],
    },
    size: {
      control: 'select',
      options: ['sm', 'md', 'lg'],
    },
    animated: {
      control: 'boolean',
    },
  },
};

export default meta;
type Story = StoryObj<typeof StatusIndicator>;

export const Online: Story = {
  args: {
    status: 'online',
  },
};

export const Offline: Story = {
  args: {
    status: 'offline',
  },
};

export const Busy: Story = {
  args: {
    status: 'busy',
  },
};

export const Away: Story = {
  args: {
    status: 'away',
  },
};

export const Error: Story = {
  args: {
    status: 'error',
  },
};

export const Warning: Story = {
  args: {
    status: 'warning',
  },
};

export const Success: Story = {
  args: {
    status: 'success',
  },
};

export const Small: Story = {
  args: {
    status: 'online',
    size: 'sm',
  },
};

export const Medium: Story = {
  args: {
    status: 'online',
    size: 'md',
  },
};

export const Large: Story = {
  args: {
    status: 'online',
    size: 'lg',
  },
};

export const Animated: Story = {
  args: {
    status: 'online',
    animated: true,
  },
};

export const WithLabel: Story = {
  args: {
    status: 'online',
    label: 'Online',
  },
};

export const WithCustomClass: Story = {
  args: {
    status: 'online',
    className: 'ring-2 ring-blue-500',
  },
};

export const AllStatuses: Story = {
  render: () => (
    <div className="flex flex-wrap gap-4">
      <StatusIndicator status="online" label="Online" />
      <StatusIndicator status="offline" label="Offline" />
      <StatusIndicator status="loading" label="Busy" />
      <StatusIndicator status="offline" label="Away" />
      <StatusIndicator status="error" label="Error" />
      <StatusIndicator status="warning" label="Warning" />
      <StatusIndicator status="success" label="Success" />
    </div>
  ),
};

export const AllSizes: Story = {
  render: () => (
    <div className="flex items-center gap-4">
      <StatusIndicator status="online" size="sm" label="Small" />
      <StatusIndicator status="online" size="md" label="Medium" />
      <StatusIndicator status="online" size="lg" label="Large" />
    </div>
  ),
};

export const InUserList: Story = {
  render: () => (
    <div className="max-w-md mx-auto space-y-3">
      <div className="flex items-center justify-between p-3 bg-white dark:bg-gray-800 rounded-lg shadow">
        <div className="flex items-center space-x-3">
          <div className="w-8 h-8 bg-blue-500 rounded-full flex items-center justify-center text-white text-sm font-medium">
            JD
          </div>
          <div>
            <p className="text-sm font-medium text-gray-900 dark:text-white">John Doe</p>
            <p className="text-xs text-gray-500 dark:text-gray-400">john@example.com</p>
          </div>
        </div>
        <StatusIndicator status="online" size="sm" />
      </div>
      
      <div className="flex items-center justify-between p-3 bg-white dark:bg-gray-800 rounded-lg shadow">
        <div className="flex items-center space-x-3">
          <div className="w-8 h-8 bg-green-500 rounded-full flex items-center justify-center text-white text-sm font-medium">
            JS
          </div>
          <div>
            <p className="text-sm font-medium text-gray-900 dark:text-white">Jane Smith</p>
            <p className="text-xs text-gray-500 dark:text-gray-400">jane@example.com</p>
          </div>
        </div>
        <StatusIndicator status="offline" size="sm" />
      </div>
      
      <div className="flex items-center justify-between p-3 bg-white dark:bg-gray-800 rounded-lg shadow">
        <div className="flex items-center space-x-3">
          <div className="w-8 h-8 bg-red-500 rounded-full flex items-center justify-center text-white text-sm font-medium">
            BJ
          </div>
          <div>
            <p className="text-sm font-medium text-gray-900 dark:text-white">Bob Johnson</p>
            <p className="text-xs text-gray-500 dark:text-gray-400">bob@example.com</p>
          </div>
        </div>
        <StatusIndicator status="offline" size="sm" />
      </div>
    </div>
  ),
};