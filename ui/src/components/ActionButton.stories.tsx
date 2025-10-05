import type { Meta, StoryObj } from '@storybook/preact';
import { ActionButton } from './ActionButton.js';

const meta: Meta<typeof ActionButton> = {
  title: 'Components/ActionButton',
  component: ActionButton,
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
    action: {
      control: 'select',
      options: ['add', 'edit', 'delete', 'save', 'cancel', 'refresh', 'download', 'upload'],
    },
  },
};

export default meta;
type Story = StoryObj<typeof ActionButton>;

export const Add: Story = {
  args: {
    action: 'add',
    children: 'Add Item',
  },
};

export const Edit: Story = {
  args: {
    action: 'edit',
    children: 'Edit',
  },
};

export const Delete: Story = {
  args: {
    action: 'delete',
    children: 'Delete',
  },
};

export const Save: Story = {
  args: {
    action: 'save',
    children: 'Save Changes',
  },
};

export const Cancel: Story = {
  args: {
    action: 'cancel',
    children: 'Cancel',
  },
};

export const Refresh: Story = {
  args: {
    action: 'refresh',
    children: 'Refresh',
  },
};

export const Download: Story = {
  args: {
    action: 'download',
    children: 'Download',
  },
};

export const Upload: Story = {
  args: {
    action: 'upload',
    children: 'Upload File',
  },
};

export const Primary: Story = {
  args: {
    action: 'add',
    children: 'Primary Action',
    variant: 'primary',
  },
};

export const Secondary: Story = {
  args: {
    action: 'edit',
    children: 'Secondary Action',
    variant: 'secondary',
  },
};

export const Danger: Story = {
  args: {
    action: 'delete',
    children: 'Danger Action',
    variant: 'danger',
  },
};

export const Success: Story = {
  args: {
    action: 'save',
    children: 'Success Action',
    variant: 'success',
  },
};

export const Warning: Story = {
  args: {
    action: 'refresh',
    children: 'Warning Action',
    variant: 'warning',
  },
};

export const Small: Story = {
  args: {
    action: 'add',
    children: 'Small',
    size: 'sm',
  },
};

export const Medium: Story = {
  args: {
    action: 'edit',
    children: 'Medium',
    size: 'md',
  },
};

export const Large: Story = {
  args: {
    action: 'save',
    children: 'Large',
    size: 'lg',
  },
};

export const Disabled: Story = {
  args: {
    action: 'add',
    children: 'Disabled',
    disabled: true,
  },
};

export const Loading: Story = {
  args: {
    action: 'save',
    children: 'Saving...',
    loading: true,
  },
};

export const AllActions: Story = {
  render: () => (
    <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
      <ActionButton action="add" onClick={() => {}} />
      <ActionButton action="edit" onClick={() => {}} />
      <ActionButton action="delete" onClick={() => {}} />
      <ActionButton action="save" onClick={() => {}} />
      <ActionButton action="cancel" onClick={() => {}} />
      <ActionButton action="refresh" onClick={() => {}} />
      <ActionButton action="view" onClick={() => {}} />
      <ActionButton action="wake" onClick={() => {}} />
    </div>
  ),
};

export const AllVariants: Story = {
  render: () => (
    <div className="flex flex-wrap gap-4">
      <ActionButton action="add" onClick={() => {}} />
      <ActionButton action="edit" onClick={() => {}} />
      <ActionButton action="delete" onClick={() => {}} />
      <ActionButton action="save" onClick={() => {}} />
      <ActionButton action="refresh" onClick={() => {}} />
    </div>
  ),
};

export const AllSizes: Story = {
  render: () => (
    <div className="flex items-center gap-4">
      <ActionButton action="add" size="sm" onClick={() => {}} />
      <ActionButton action="edit" size="md" onClick={() => {}} />
      <ActionButton action="save" size="lg" onClick={() => {}} />
    </div>
  ),
};