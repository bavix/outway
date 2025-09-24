import type { Meta, StoryObj } from '@storybook/preact';
import { Header } from './Header';

const meta: Meta<typeof Header> = {
  title: 'Components/Header',
  component: Header,
  parameters: {
    layout: 'fullscreen',
    docs: {
      description: {
        component: `
The main header component for the admin interface.

## Features
- Sidebar toggle button
- Theme switcher
- Clean, minimal design
- Responsive behavior
        `,
      },
    },
  },
  argTypes: {
    sidebarCollapsed: {
      control: 'boolean',
      description: 'Sidebar collapsed state',
    },
  },
};

export default meta;
type Story = StoryObj<typeof Header>;

export const Default: Story = {
  render: () => (
    <Header
      sidebarCollapsed={false}
      onSidebarToggle={() => console.log('Sidebar toggle clicked')}
    />
  ),
};

export const WithCollapsedSidebar: Story = {
  render: () => (
    <Header
      sidebarCollapsed={true}
      onSidebarToggle={() => console.log('Sidebar toggle clicked')}
    />
  ),
};

export const InLayout: Story = {
  render: () => (
    <div className="bg-white dark:bg-black">
      <Header
        sidebarCollapsed={false}
        onSidebarToggle={() => console.log('Sidebar toggle clicked')}
      />
      <div className="bg-black dark:bg-white p-8">
        <h1 className="text-white dark:text-black font-bold text-xl">
          Page Content
        </h1>
        <p className="text-gray-300 dark:text-gray-700 mt-2">
          This shows how the header looks in a typical page layout.
        </p>
      </div>
    </div>
  ),
};