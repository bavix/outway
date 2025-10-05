import type { Meta, StoryObj } from '@storybook/preact';
import { StatusCard } from './StatusCard.js';

const meta: Meta<typeof StatusCard> = {
  title: 'Components/StatusCard',
  component: StatusCard,
  parameters: {
    layout: 'padded',
  },
  argTypes: {
    status: {
      control: 'select',
      options: ['active', 'inactive', 'warning', 'error'],
    },
  },
};

export default meta;
type Story = StoryObj<typeof StatusCard>;

const ServerIcon = () => (
  <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-2a2 2 0 00-2-2m-2-4h.01M17 16h.01" />
  </svg>
);

const UsersIcon = () => (
  <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197m13.5-9a2.5 2.5 0 11-5 0 2.5 2.5 0 015 0z" />
  </svg>
);

const ShieldIcon = () => (
  <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
  </svg>
);

export const Default: Story = {
  args: {
    title: 'Server Status',
    description: 'All systems operational',
    count: 42,
    icon: <ServerIcon />,
    status: 'active',
  },
};

export const Inactive: Story = {
  args: {
    title: 'Maintenance Mode',
    description: 'System is under maintenance',
    count: 0,
    icon: <ServerIcon />,
    status: 'inactive',
  },
};

export const Warning: Story = {
  args: {
    title: 'High Load',
    description: 'CPU usage is above 80%',
    count: 15,
    icon: <ServerIcon />,
    status: 'warning',
  },
};

export const Error: Story = {
  args: {
    title: 'Service Down',
    description: 'Database connection failed',
    count: 0,
    icon: <ServerIcon />,
    status: 'error',
  },
};

export const WithoutCount: Story = {
  args: {
    title: 'Security Status',
    description: 'All security checks passed',
    icon: <ShieldIcon />,
    status: 'active',
  },
};

export const WithoutDescription: Story = {
  args: {
    title: 'Active Users',
    count: 128,
    icon: <UsersIcon />,
    status: 'active',
  },
};

export const WithoutIcon: Story = {
  args: {
    title: 'System Health',
    description: 'All components are working properly',
    count: 100,
    status: 'active',
  },
};

export const Clickable: Story = {
  args: {
    title: 'View Details',
    description: 'Click to see more information',
    count: 5,
    icon: <ServerIcon />,
    status: 'active',
    onClick: () => alert('Card clicked!'),
  },
};

export const AllStatuses: Story = {
  render: () => (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
      <StatusCard
        title="Active"
        description="All systems operational"
        count={42}
        icon={<ServerIcon />}
        status="active"
      />
      <StatusCard
        title="Inactive"
        description="System is offline"
        count={0}
        icon={<ServerIcon />}
        status="inactive"
      />
      <StatusCard
        title="Warning"
        description="High resource usage"
        count={15}
        icon={<ServerIcon />}
        status="warning"
      />
      <StatusCard
        title="Error"
        description="Service unavailable"
        count={0}
        icon={<ServerIcon />}
        status="error"
      />
    </div>
  ),
};
