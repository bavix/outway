import type { Meta, StoryObj } from '@storybook/preact';
import { AccessibleButton } from './AccessibleButton.js';

const meta: Meta<typeof AccessibleButton> = {
  title: 'Components/AccessibleButton',
  component: AccessibleButton,
  parameters: {
    layout: 'centered',
  },
  argTypes: {
    variant: {
      control: 'select',
      options: ['primary', 'secondary', 'danger', 'success', 'warning'],
    },
    size: {
      control: 'select',
      options: ['sm', 'md', 'lg'],
    },
    disabled: {
      control: 'boolean',
    },
    loading: {
      control: 'boolean',
    },
  },
};

export default meta;
type Story = StoryObj<typeof AccessibleButton>;

export const Default: Story = {
  args: {
    children: 'Accessible Button',
  },
};

export const Primary: Story = {
  args: {
    children: 'Primary Button',
    variant: 'primary',
  },
};

export const Secondary: Story = {
  args: {
    children: 'Secondary Button',
    variant: 'secondary',
  },
};

export const Danger: Story = {
  args: {
    children: 'Danger Button',
    variant: 'danger',
  },
};

export const Success: Story = {
  args: {
    children: 'Success Button',
    variant: 'success',
  },
};

export const Warning: Story = {
  args: {
    children: 'Warning Button',
    variant: 'warning',
  },
};

export const Small: Story = {
  args: {
    children: 'Small Button',
    size: 'sm',
  },
};

export const Medium: Story = {
  args: {
    children: 'Medium Button',
    size: 'md',
  },
};

export const Large: Story = {
  args: {
    children: 'Large Button',
    size: 'lg',
  },
};

export const Disabled: Story = {
  args: {
    children: 'Disabled Button',
    disabled: true,
  },
};

export const Loading: Story = {
  args: {
    children: 'Loading Button',
    loading: true,
  },
};

export const WithAriaLabel: Story = {
  args: {
    children: 'Save',
    'aria-label': 'Save the current document',
  },
};

export const WithAriaDescribedBy: Story = {
  args: {
    children: 'Delete',
    'aria-describedby': 'delete-help',
    variant: 'danger',
  },
  render: (args) => (
    <div>
      <AccessibleButton {...args} onClick={() => {}}>Delete</AccessibleButton>
      <p id="delete-help" className="text-sm text-gray-500 mt-2">
        This action cannot be undone
      </p>
    </div>
  ),
};

export const WithTooltip: Story = {
  args: {
    children: 'Hover me',
    title: 'This is a tooltip',
  },
};

export const IconButton: Story = {
  args: {
    children: (
      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
      </svg>
    ),
    'aria-label': 'Add new item',
    variant: 'primary',
  },
};

export const KeyboardNavigation: Story = {
  render: () => (
    <div className="space-y-4">
      <p className="text-sm text-gray-600 dark:text-gray-400">
        Use Tab to navigate between buttons, Enter/Space to activate
      </p>
      <div className="flex gap-2">
        <AccessibleButton variant="primary" onClick={() => {}}>First</AccessibleButton>
        <AccessibleButton variant="secondary" onClick={() => {}}>Second</AccessibleButton>
        <AccessibleButton variant="danger" onClick={() => {}}>Third</AccessibleButton>
      </div>
    </div>
  ),
};

export const FocusStates: Story = {
  render: () => (
    <div className="space-y-4">
      <div className="flex gap-2">
        <AccessibleButton variant="primary" onClick={() => {}}>Normal</AccessibleButton>
        <AccessibleButton variant="primary" className="focus:ring-4 focus:ring-blue-300" onClick={() => {}}>
          Custom Focus
        </AccessibleButton>
      </div>
    </div>
  ),
};

export const AllVariants: Story = {
  render: () => (
    <div className="flex flex-wrap gap-4">
      <AccessibleButton variant="primary" onClick={() => {}}>Primary</AccessibleButton>
      <AccessibleButton variant="secondary" onClick={() => {}}>Secondary</AccessibleButton>
      <AccessibleButton variant="danger" onClick={() => {}}>Danger</AccessibleButton>
      <AccessibleButton variant="outline" onClick={() => {}}>Outline</AccessibleButton>
      <AccessibleButton variant="outline" onClick={() => {}}>Outline</AccessibleButton>
    </div>
  ),
};

export const AllSizes: Story = {
  render: () => (
    <div className="flex items-center gap-4">
      <AccessibleButton size="sm" onClick={() => {}}>Small</AccessibleButton>
      <AccessibleButton size="md" onClick={() => {}}>Medium</AccessibleButton>
      <AccessibleButton size="lg" onClick={() => {}}>Large</AccessibleButton>
    </div>
  ),
};

export const States: Story = {
  render: () => (
    <div className="flex flex-wrap gap-4">
      <AccessibleButton onClick={() => {}}>Normal</AccessibleButton>
      <AccessibleButton disabled onClick={() => {}}>Disabled</AccessibleButton>
      <AccessibleButton loading onClick={() => {}}>Loading</AccessibleButton>
    </div>
  ),
};
