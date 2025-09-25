import type { Meta, StoryObj } from '@storybook/preact';
import { StatsCard } from './StatsCard';

const meta: Meta<typeof StatsCard> = {
  title: 'Components/StatsCard',
  component: StatsCard,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component: `
A specialized card component for displaying statistics and metrics.

## Usage
\`\`\`tsx
<StatsCard 
  title="Users" 
  value="1,234" 
  change={{ value: 12.5, type: 'increase' }}
  icon={<UserIcon />}
/>
\`\`\`

## Features
- Title and value display
- Change indicators (increase/decrease)
- Optional icons
- Color variants
- Perfect for dashboards
        `,
      },
    },
  },
  argTypes: {
    title: {
      control: 'text',
      description: 'Stat title',
    },
    value: {
      control: 'text',
      description: 'Stat value',
    },
    color: {
      control: 'select',
      options: ['blue', 'green', 'red', 'yellow', 'purple'],
      description: 'Color variant',
    },
  },
};

export default meta;
type Story = StoryObj<typeof StatsCard>;

export const Simple: Story = {
  args: {
    title: 'Total Users',
    value: '1,234',
  },
};

export const WithIncrease: Story = {
  args: {
    title: 'Revenue',
    value: '$45,678',
    change: { value: 12.5, type: 'increase' },
  },
};

export const WithDecrease: Story = {
  args: {
    title: 'Errors',
    value: '23',
    change: { value: 5.1, type: 'decrease' },
  },
};

export const WithIcon: Story = {
  args: {
    title: 'Users',
    value: '1,234',
    change: { value: 12.5, type: 'increase' },
    icon: (
      <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197m13.5-9a2.5 2.5 0 11-5 0 2.5 2.5 0 015 0z" />
      </svg>
    ),
    color: 'blue',
  },
};

export const Dashboard: Story = {
  render: () => (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 w-full max-w-6xl">
      <StatsCard
        title="Users"
        value="1,234"
        change={{ value: 12.5, type: 'increase' }}
        icon={
          <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197m13.5-9a2.5 2.5 0 11-5 0 2.5 2.5 0 015 0z" />
          </svg>
        }
        color="blue"
      />
      
      <StatsCard
        title="Revenue"
        value="$45,678"
        change={{ value: 8.2, type: 'decrease' }}
        icon={
          <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1" />
          </svg>
        }
        color="green"
      />
      
      <StatsCard
        title="Errors"
        value="23"
        change={{ value: 5.1, type: 'increase' }}
        icon={
          <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.732-.833-2.5 0L4.268 18.5c-.77.833.192 2.5 1.732 2.5z" />
          </svg>
        }
        color="red"
      />
      
      <StatsCard
        title="Performance"
        value="98.5%"
        change={{ value: 2.3, type: 'increase' }}
        icon={
          <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
          </svg>
        }
        color="yellow"
      />
    </div>
  ),
  parameters: {
    layout: 'padded',
  },
};