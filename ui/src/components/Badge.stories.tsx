import type { Meta, StoryObj } from '@storybook/preact';
import { Badge } from './Badge';

const meta: Meta<typeof Badge> = {
  title: 'Components/Badge',
  component: Badge,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component: `
A simple badge component for displaying counts and labels.

## Usage
\`\`\`tsx
<Badge variant="primary">24</Badge>
\`\`\`

## Features
- Three variants: default, primary, secondary
- Two sizes: sm, md
- Perfect for counters and labels
        `,
      },
    },
  },
  argTypes: {
    variant: {
      control: 'select',
      options: ['default', 'primary', 'secondary'],
      description: 'Badge variant',
    },
    size: {
      control: 'select',
      options: ['sm', 'md'],
      description: 'Badge size',
    },
    children: {
      control: 'text',
      description: 'Badge content',
    },
  },
};

export default meta;
type Story = StoryObj<typeof Badge>;

export const Default: Story = {
  args: {
    children: 'Default',
  },
};

export const Primary: Story = {
  args: {
    variant: 'primary',
    children: 'Primary',
  },
};

export const Secondary: Story = {
  args: {
    variant: 'secondary',
    children: 'Secondary',
  },
};

export const Sizes: Story = {
  render: () => (
    <div className="flex items-center gap-3">
      <Badge size="sm">Small</Badge>
      <Badge size="md">Medium</Badge>
    </div>
  ),
};

export const Counters: Story = {
  render: () => (
    <div className="flex items-center gap-4">
      <span className="text-sm text-gray-600 dark:text-gray-400">Rules:</span>
      <Badge variant="primary">24</Badge>
      <span className="text-sm text-gray-600 dark:text-gray-400">Upstreams:</span>
      <Badge variant="primary">8</Badge>
      <span className="text-sm text-gray-600 dark:text-gray-400">Active:</span>
      <Badge variant="secondary">6</Badge>
    </div>
  ),
};