import type { Meta, StoryObj } from '@storybook/preact';
import { Sidebar } from './Sidebar';

const meta: Meta<typeof Sidebar> = {
  title: 'Components/Sidebar',
  component: Sidebar,
  parameters: {
    layout: 'fullscreen',
    docs: {
      description: {
        component: `
The main navigation sidebar for the admin interface.

## Features
- Collapsible design
- Active tab highlighting
- Responsive behavior
- Clean, minimal navigation
        `,
      },
    },
  },
  argTypes: {
    activeTab: {
      control: 'select',
      options: ['overview', 'rules', 'upstreams', 'history'],
      description: 'Currently active tab',
    },
    collapsed: {
      control: 'boolean',
      description: 'Collapsed state',
    },
  },
};

export default meta;
type Story = StoryObj<typeof Sidebar>;

export const Expanded: Story = {
  args: {
    activeTab: 'overview',
    collapsed: false,
    onTabChange: (tab: string) => console.log('Tab changed:', tab),
  },
};

export const Collapsed: Story = {
  args: {
    activeTab: 'rules',
    collapsed: true,
    onTabChange: (tab: string) => console.log('Tab changed:', tab),
  },
};

export const AllTabs: Story = {
  render: () => (
    <div className="flex h-screen">
      <Sidebar
        activeTab="upstreams"
        collapsed={false}
        onTabChange={(tab: string) => console.log('Tab changed:', tab)}
      />
      <div className="flex-1 p-8 bg-gray-50 dark:bg-gray-900">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white mb-4">
          Main Content Area
        </h1>
        <p className="text-gray-600 dark:text-gray-400">
          This shows how the sidebar looks alongside main content.
        </p>
      </div>
    </div>
  ),
};