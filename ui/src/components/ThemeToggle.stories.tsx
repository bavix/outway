import type { Meta, StoryObj } from '@storybook/preact';
import { ThemeToggle } from './ThemeToggle';

const meta: Meta<typeof ThemeToggle> = {
  title: 'Components/ThemeToggle',
  component: ThemeToggle,
  parameters: {
    layout: 'centered',
  },
  argTypes: {
    variant: {
      control: 'select',
      options: ['select', 'button'],
    },
    showLabel: {
      control: 'boolean',
    },
  },
};

export default meta;
type Story = StoryObj<typeof ThemeToggle>;

export const Select: Story = {
  args: {
    variant: 'select',
  },
};

export const SelectWithLabel: Story = {
  args: {
    variant: 'select',
    showLabel: true,
  },
};

export const Button: Story = {
  args: {
    variant: 'button',
  },
};

export const ButtonWithLabel: Story = {
  args: {
    variant: 'button',
    showLabel: true,
  },
};

export const AllVariants: Story = {
  render: () => (
    <div className="space-y-4">
      <div>
        <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Select Variant</h3>
        <ThemeToggle variant="select" />
      </div>
      <div>
        <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Select with Label</h3>
        <ThemeToggle variant="select" showLabel />
      </div>
      <div>
        <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Button Variant</h3>
        <ThemeToggle variant="button" />
      </div>
      <div>
        <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Button with Label</h3>
        <ThemeToggle variant="button" showLabel />
      </div>
    </div>
  ),
};
