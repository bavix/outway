import type { Meta, StoryObj } from '@storybook/preact';
import { Card } from './Card';
import { Button } from './Button';

const meta: Meta<typeof Card> = {
  title: 'Components/Card',
  component: Card,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component: `
A simple card component for grouping content.

## Usage
\`\`\`tsx
<Card title="Card Title" subtitle="Card subtitle">
  Card content goes here
</Card>
\`\`\`

## Features
- Optional title and subtitle
- Clean, minimal design
- Consistent spacing
- Dark theme support
        `,
      },
    },
  },
  argTypes: {
    title: {
      control: 'text',
      description: 'Card title',
    },
    subtitle: {
      control: 'text',
      description: 'Card subtitle',
    },
  },
};

export default meta;
type Story = StoryObj<typeof Card>;

export const Simple: Story = {
  args: {
    children: 'This is a simple card with just content.',
  },
};

export const WithTitle: Story = {
  args: {
    title: 'Card Title',
    children: 'This card has a title.',
  },
};

export const WithTitleAndSubtitle: Story = {
  args: {
    title: 'Card Title',
    subtitle: 'This is a subtitle that provides more context',
    children: 'This card has both a title and subtitle.',
  },
};

export const WithActions: Story = {
  render: () => (
    <Card title="Card with Actions" subtitle="A card containing interactive elements">
      <div className="space-y-4">
        <p className="text-gray-600 dark:text-gray-400">
          This card contains some content and action buttons below.
        </p>
        <div className="flex gap-3">
          <Button variant="primary" size="sm">Primary Action</Button>
          <Button variant="outline" size="sm">Secondary Action</Button>
        </div>
      </div>
    </Card>
  ),
};

export const MultipleCards: Story = {
  render: () => (
    <div className="grid grid-cols-1 md:grid-cols-2 gap-6 w-full max-w-4xl">
      <Card title="Statistics" subtitle="Key metrics">
        <div className="space-y-2">
          <div className="flex justify-between">
            <span className="text-sm text-gray-600 dark:text-gray-400">Users</span>
            <span className="font-medium">1,234</span>
          </div>
          <div className="flex justify-between">
            <span className="text-sm text-gray-600 dark:text-gray-400">Revenue</span>
            <span className="font-medium">$45,678</span>
          </div>
        </div>
      </Card>
      
      <Card title="Recent Activity" subtitle="Latest updates">
        <div className="space-y-2">
          <div className="text-sm text-gray-600 dark:text-gray-400">
            User John Doe logged in
          </div>
          <div className="text-sm text-gray-600 dark:text-gray-400">
            New order #1234 created
          </div>
        </div>
      </Card>
    </div>
  ),
  parameters: {
    layout: 'padded',
  },
};