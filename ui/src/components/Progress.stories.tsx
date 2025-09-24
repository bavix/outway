import type { Meta, StoryObj } from '@storybook/preact';
import { Progress, CircularProgress as CircularProgressComponent } from './Progress';

const meta: Meta<typeof Progress> = {
  title: 'Components/Progress',
  component: Progress,
  parameters: {
    layout: 'padded',
    docs: {
      description: {
        component: `
Progress indicators for showing completion status.

## Features
- Linear and circular progress bars
- Multiple sizes and colors
- Percentage display
- Smooth animations
        `,
      },
    },
  },
  argTypes: {
    value: {
      control: { type: 'range', min: 0, max: 100 },
      description: 'Progress value (0-100)',
    },
    size: {
      control: 'select',
      options: ['sm', 'md', 'lg'],
      description: 'Progress size',
    },
    color: {
      control: 'select',
      options: ['primary', 'secondary', 'success', 'warning', 'error'],
      description: 'Progress color',
    },
    showPercentage: {
      control: 'boolean',
      description: 'Show percentage',
    },
  },
};

export default meta;
type Story = StoryObj<typeof Progress>;

export const Default: Story = {
  args: {
    value: 65,
    showPercentage: true,
  },
};

export const Sizes: Story = {
  render: () => (
    <div className="space-y-4">
      <div>
        <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">Small</p>
        <Progress value={45} size="sm" showPercentage />
      </div>
      <div>
        <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">Medium</p>
        <Progress value={65} size="md" showPercentage />
      </div>
      <div>
        <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">Large</p>
        <Progress value={85} size="lg" showPercentage />
      </div>
    </div>
  ),
};

export const Colors: Story = {
  render: () => (
    <div className="space-y-4">
      <div>
        <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">Primary</p>
        <Progress value={30} color="primary" showPercentage />
      </div>
      <div>
        <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">Success</p>
        <Progress value={60} color="success" showPercentage />
      </div>
      <div>
        <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">Warning</p>
        <Progress value={80} color="warning" showPercentage />
      </div>
      <div>
        <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">Error</p>
        <Progress value={95} color="error" showPercentage />
      </div>
    </div>
  ),
};

export const CircularProgress: Story = {
  render: () => (
    <div className="flex space-x-8">
      <div className="text-center">
        <CircularProgressComponent value={25} size={80} />
        <p className="text-sm text-gray-600 dark:text-gray-400 mt-2">25%</p>
      </div>
      <div className="text-center">
        <CircularProgressComponent value={50} size={80} color="success" />
        <p className="text-sm text-gray-600 dark:text-gray-400 mt-2">50%</p>
      </div>
      <div className="text-center">
        <CircularProgressComponent value={75} size={80} color="warning" />
        <p className="text-sm text-gray-600 dark:text-gray-400 mt-2">75%</p>
      </div>
      <div className="text-center">
        <CircularProgressComponent value={100} size={80} color="error" />
        <p className="text-sm text-gray-600 dark:text-gray-400 mt-2">100%</p>
      </div>
    </div>
  ),
};
