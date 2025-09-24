import type { Meta, StoryObj } from '@storybook/preact';
import { Table } from './Table';
import { Badge } from './Badge';

interface SampleData {
  id: number;
  name: string;
  status: 'active' | 'inactive';
  requests: number;
  lastSeen: string;
}

const meta: Meta<typeof Table> = {
  title: 'Components/Table',
  component: Table,
  parameters: {
    layout: 'padded',
    docs: {
      description: {
        component: `
Tables for displaying structured data.

## Features
- Sortable columns
- Loading state
- Empty state
- Row click handlers
- Custom cell rendering
- Responsive design
        `,
      },
    },
  },
  argTypes: {
    loading: {
      control: 'boolean',
      description: 'Loading state',
    },
    emptyMessage: {
      control: 'text',
      description: 'Empty state message',
    },
  },
};

export default meta;
type Story = StoryObj<typeof Table>;

const sampleData: SampleData[] = [
  { id: 1, name: 'google.com', status: 'active', requests: 1250, lastSeen: '2 min ago' },
  { id: 2, name: 'github.com', status: 'active', requests: 890, lastSeen: '5 min ago' },
  { id: 3, name: 'stackoverflow.com', status: 'inactive', requests: 340, lastSeen: '1 hour ago' },
  { id: 4, name: 'reddit.com', status: 'active', requests: 2100, lastSeen: '1 min ago' },
  { id: 5, name: 'youtube.com', status: 'active', requests: 1800, lastSeen: '3 min ago' },
];

const columns = [
  {
    key: 'name',
    title: 'Domain',
    render: (value: string) => (
      <span className="font-medium text-gray-900 dark:text-gray-100">{value}</span>
    ),
  },
  {
    key: 'status',
    title: 'Status',
    render: (value: string) => (
      <Badge variant={value === 'active' ? 'primary' : 'secondary'}>
        {value}
      </Badge>
    ),
  },
  {
    key: 'requests',
    title: 'Requests',
    render: (value: number) => value.toLocaleString(),
  },
  {
    key: 'lastSeen',
    title: 'Last Seen',
  },
];

export const Default: Story = {
  args: {
    columns,
    data: sampleData,
  },
};

export const Loading: Story = {
  args: {
    columns,
    data: [],
    loading: true,
  },
};

export const Empty: Story = {
  args: {
    columns,
    data: [],
    emptyMessage: 'No domains found',
  },
};

export const WithRowClick: Story = {
  args: {
    columns,
    data: sampleData,
    onRowClick: (item: SampleData) => {
      alert(`Clicked on ${item.name}`);
    },
  },
};
