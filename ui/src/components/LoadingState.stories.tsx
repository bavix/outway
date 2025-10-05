import type { Meta, StoryObj } from '@storybook/preact';
import { LoadingState } from './LoadingState.js';

const meta: Meta<typeof LoadingState> = {
  title: 'Components/LoadingState',
  component: LoadingState,
  parameters: {
    layout: 'padded',
  },
  argTypes: {
    message: {
      control: 'text',
    },
  },
};

export default meta;
type Story = StoryObj<typeof LoadingState>;

export const Default: Story = {
  args: {
    message: 'Loading...',
  },
};

export const CustomMessage: Story = {
  args: {
    message: 'Fetching data from server...',
  },
};

export const ShortMessage: Story = {
  args: {
    message: 'Loading',
  },
};

export const LongMessage: Story = {
  args: {
    message: 'Please wait while we process your request and fetch all the necessary data from our servers',
  },
};

export const WithCustomClass: Story = {
  args: {
    message: 'Loading with custom styling',
    className: 'bg-blue-50 dark:bg-blue-900/20 rounded-lg',
  },
};

export const InCard: Story = {
  render: () => (
    <div className="max-w-md mx-auto">
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow-lg p-6">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
          User Profile
        </h3>
        <LoadingState message="Loading user data..." />
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
          <LoadingState message="Loading table data..." />
        </div>
      </div>
    </div>
  ),
};
