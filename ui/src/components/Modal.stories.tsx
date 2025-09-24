import type { Meta, StoryObj } from '@storybook/preact';
import { Modal } from './Modal';
import { useState } from 'preact/hooks';

const meta: Meta<typeof Modal> = {
  title: 'Components/Modal',
  component: Modal,
  parameters: {
    layout: 'fullscreen',
    docs: {
      description: {
        component: `
Modal dialogs for important actions and information.

## Features
- Multiple sizes
- Primary and secondary actions
- Keyboard navigation (ESC to close)
- Loading states
- Custom content
        `,
      },
    },
  },
  argTypes: {
    size: {
      control: 'select',
      options: ['sm', 'md', 'lg', 'xl'],
      description: 'Modal size',
    },
    isOpen: {
      control: 'boolean',
      description: 'Modal visibility',
    },
  },
};

export default meta;
type Story = StoryObj<typeof Modal>;

const ModalWrapper = (args: any) => {
  const [isOpen, setIsOpen] = useState(args.isOpen);
  
  return (
    <div className="p-8">
      <button
        onClick={() => setIsOpen(true)}
        className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
      >
        Open Modal
      </button>
      
      <Modal
        {...args}
        isOpen={isOpen}
        onClose={() => setIsOpen(false)}
        primaryAction={{
          ...args.primaryAction,
          onClick: () => {
            args.primaryAction?.onClick?.();
            setIsOpen(false);
          },
        }}
        secondaryAction={{
          ...args.secondaryAction,
          onClick: () => {
            args.secondaryAction?.onClick?.();
            setIsOpen(false);
          },
        }}
      />
    </div>
  );
};

export const Default: Story = {
  render: ModalWrapper,
  args: {
    isOpen: false,
    title: 'Confirm Action',
    children: (
      <div>
        <p className="text-gray-600 dark:text-gray-400">
          Are you sure you want to delete this item? This action cannot be undone.
        </p>
      </div>
    ),
    primaryAction: {
      label: 'Delete',
      onClick: () => console.log('Deleted'),
      variant: 'primary' as const,
    },
    secondaryAction: {
      label: 'Cancel',
      onClick: () => console.log('Cancelled'),
      variant: 'outline' as const,
    },
  },
};

export const LargeModal: Story = {
  render: ModalWrapper,
  args: {
    isOpen: false,
    title: 'Settings',
    size: 'lg',
    children: (
      <div className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            DNS Server
          </label>
          <input
            type="text"
            defaultValue="8.8.8.8"
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            Timeout (seconds)
          </label>
          <input
            type="number"
            defaultValue="5"
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>
        <div className="flex items-center">
          <input
            type="checkbox"
            id="cache"
            defaultChecked
            className="mr-2"
          />
          <label htmlFor="cache" className="text-sm text-gray-700 dark:text-gray-300">
            Enable caching
          </label>
        </div>
      </div>
    ),
    primaryAction: {
      label: 'Save',
      onClick: () => console.log('Saved'),
      variant: 'primary' as const,
    },
    secondaryAction: {
      label: 'Cancel',
      onClick: () => console.log('Cancelled'),
      variant: 'outline' as const,
    },
  },
};

export const LoadingAction: Story = {
  render: ModalWrapper,
  args: {
    isOpen: false,
    title: 'Processing',
    children: (
      <div>
        <p className="text-gray-600 dark:text-gray-400">
          Please wait while we process your request...
        </p>
      </div>
    ),
    primaryAction: {
      label: 'Processing...',
      onClick: () => console.log('Processing'),
      variant: 'primary' as const,
      loading: true,
      disabled: true,
    },
  },
};
