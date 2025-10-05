import type { Meta, StoryObj } from '@storybook/preact';
import { DataTable } from './DataTable.js';

interface User {
  id: number;
  name: string;
  email: string;
  role: string;
  status: 'active' | 'inactive';
  lastLogin: string;
}

const sampleData: User[] = [
  {
    id: 1,
    name: 'John Doe',
    email: 'john@example.com',
    role: 'admin',
    status: 'active',
    lastLogin: '2024-01-15'
  },
  {
    id: 2,
    name: 'Jane Smith',
    email: 'jane@example.com',
    role: 'user',
    status: 'inactive',
    lastLogin: '2024-01-10'
  },
  {
    id: 3,
    name: 'Bob Johnson',
    email: 'bob@example.com',
    role: 'user',
    status: 'active',
    lastLogin: '2024-01-14'
  }
];

const columns = [
  {
    key: 'name',
    title: 'Name',
    render: (value: string, item: User) => (
      <div className="flex items-center space-x-3">
        <div className="w-8 h-8 bg-blue-100 dark:bg-blue-900/30 rounded-full flex items-center justify-center">
          <span className="text-sm font-medium text-blue-700 dark:text-blue-300">
            {value.charAt(0).toUpperCase()}
          </span>
        </div>
        <div>
          <div className="text-sm font-medium text-gray-900 dark:text-white">{value}</div>
          <div className="text-sm text-gray-500 dark:text-gray-400">{item.email}</div>
        </div>
      </div>
    )
  },
  {
    key: 'role',
    title: 'Role',
    render: (value: string) => (
      <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
        value === 'admin' 
          ? 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300'
          : 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300'
      }`}>
        {value}
      </span>
    )
  },
  {
    key: 'status',
    title: 'Status',
    render: (value: string) => (
      <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
        value === 'active'
          ? 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300'
          : 'bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-300'
      }`}>
        {value}
      </span>
    )
  },
  {
    key: 'lastLogin',
    title: 'Last Login'
  },
  {
    key: 'actions',
    title: 'Actions',
    render: () => (
      <div className="flex space-x-2">
        <button className="text-blue-600 hover:text-blue-800 text-sm font-medium">Edit</button>
        <button className="text-red-600 hover:text-red-800 text-sm font-medium">Delete</button>
      </div>
    )
  }
];

const meta: Meta<typeof DataTable> = {
  title: 'Components/DataTable',
  component: DataTable,
  parameters: {
    layout: 'padded',
  },
  argTypes: {
    data: {
      control: 'object',
    },
    loading: {
      control: 'boolean',
    },
    emptyMessage: {
      control: 'text',
    },
    emptyDescription: {
      control: 'text',
    },
  },
};

export default meta;
type Story = StoryObj<typeof DataTable>;

export const Default: Story = {
  args: {
    data: sampleData,
    columns,
  },
};

export const Loading: Story = {
  args: {
    data: [],
    columns,
    loading: true,
  },
};

export const Empty: Story = {
  args: {
    data: [],
    columns,
    emptyMessage: 'No users found',
    emptyDescription: 'There are no users in the system yet.',
  },
};

export const WithRowClick: Story = {
  args: {
    data: sampleData,
    columns,
    onRowClick: (item: User, index: number) => {
      alert(`Clicked on ${item.name} (index: ${index})`);
    },
  },
};
