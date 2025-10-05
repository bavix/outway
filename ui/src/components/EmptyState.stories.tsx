import type { Meta, StoryObj } from '@storybook/preact';
import { EmptyState } from './EmptyState.js';

const meta: Meta<typeof EmptyState> = {
  title: 'Components/EmptyState',
  component: EmptyState,
  parameters: {
    layout: 'padded',
  },
  argTypes: {
    icon: {
      control: 'select',
      options: ['shield', 'users', 'roles', 'default'],
    },
  },
};

export default meta;
type Story = StoryObj<typeof EmptyState>;

export const Default: Story = {
  args: {
    title: 'No data available',
    description: 'There are no items to display at the moment.',
  },
};

export const Shield: Story = {
  args: {
    title: 'No permissions found',
    description: 'This user does not have any permissions assigned.',
    icon: 'shield',
  },
};

export const Users: Story = {
  args: {
    title: 'No users found',
    description: 'There are no users in the system yet. Create your first user to get started.',
    icon: 'users',
  },
};

export const Roles: Story = {
  args: {
    title: 'No roles configured',
    description: 'Set up roles and permissions to control user access.',
    icon: 'roles',
  },
};

export const LongDescription: Story = {
  args: {
    title: 'No results found',
    description: 'We couldn\'t find any items matching your search criteria. Try adjusting your filters or search terms to see more results.',
    icon: 'default',
  },
};

export const ShortTitle: Story = {
  args: {
    title: 'Empty',
    description: 'Nothing here yet.',
  },
};

export const InCard: Story = {
  render: () => (
    <div className="max-w-md mx-auto">
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow-lg p-6">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
          User Management
        </h3>
        <EmptyState
          title="No users found"
          description="Create your first user to get started with user management."
          icon="user"
        />
      </div>
    </div>
  ),
};

export const InTable: Story = {
  render: () => (
    <div className="max-w-4xl mx-auto">
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow-lg overflow-hidden">
        <div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
            Data Table
          </h3>
        </div>
        <div className="p-6">
          <EmptyState
            title="No data available"
            description="There are no records to display. Add some data to get started."
            icon="box"
          />
        </div>
      </div>
    </div>
  ),
};

export const AllIcons: Story = {
  render: () => (
    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
      <EmptyState
        title="No permissions"
        description="No permissions found"
        icon="shield"
      />
      <EmptyState
        title="No users"
        description="No users found"
        icon="user"
      />
      <EmptyState
        title="No roles"
        description="No roles configured"
        icon="settings"
      />
      <EmptyState
        title="No data"
        description="No data available"
        icon="box"
      />
    </div>
  ),
};
