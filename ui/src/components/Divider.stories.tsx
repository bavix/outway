import type { Meta, StoryObj } from '@storybook/preact';
import { Divider } from './Divider';

const meta: Meta<typeof Divider> = {
  title: 'Components/Divider',
  component: Divider,
  parameters: {
    layout: 'padded',
    docs: {
      description: {
        component: `
Dividers for separating content sections.

## Features
- Horizontal and vertical orientations
- Text dividers
- Multiple line styles
- Responsive design
        `,
      },
    },
  },
  argTypes: {
    orientation: {
      control: 'select',
      options: ['horizontal', 'vertical'],
      description: 'Divider orientation',
    },
    text: {
      control: 'text',
      description: 'Divider text',
    },
    variant: {
      control: 'select',
      options: ['solid', 'dashed', 'dotted'],
      description: 'Line style',
    },
  },
};

export default meta;
type Story = StoryObj<typeof Divider>;

export const Horizontal: Story = {
  args: {
    orientation: 'horizontal',
  },
};

export const WithText: Story = {
  args: {
    orientation: 'horizontal',
    text: 'OR',
  },
};

export const Variants: Story = {
  render: () => (
    <div className="space-y-8">
      <div>
        <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">Solid</p>
        <Divider variant="solid" />
      </div>
      <div>
        <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">Dashed</p>
        <Divider variant="dashed" />
      </div>
      <div>
        <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">Dotted</p>
        <Divider variant="dotted" />
      </div>
    </div>
  ),
};

export const Vertical: Story = {
  render: () => (
    <div className="flex items-center h-32 space-x-4">
      <div className="flex-1 text-center">
        <p className="text-gray-600 dark:text-gray-400">Left Content</p>
      </div>
      <Divider orientation="vertical" className="h-16" />
      <div className="flex-1 text-center">
        <p className="text-gray-600 dark:text-gray-400">Right Content</p>
      </div>
    </div>
  ),
};
