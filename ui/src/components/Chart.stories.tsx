import type { Meta, StoryObj } from '@storybook/preact';
import { Chart } from './Chart';

const meta: Meta<typeof Chart> = {
  title: 'Components/Chart',
  component: Chart,
  parameters: {
    layout: 'padded',
    docs: {
      description: {
        component: `
Charts for displaying data visualizations.

## Features
- Line, bar, and doughnut chart types
- Loading and error states
- Responsive design
- Customizable colors
        `,
      },
    },
  },
  argTypes: {
    type: {
      control: 'select',
      options: ['line', 'bar', 'doughnut'],
      description: 'Chart type',
    },
    height: {
      control: 'number',
      description: 'Chart height in pixels',
    },
    loading: {
      control: 'boolean',
      description: 'Loading state',
    },
    error: {
      control: 'text',
      description: 'Error message',
    },
  },
};

export default meta;
type Story = StoryObj<typeof Chart>;

const sampleData = {
  labels: ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun'],
  datasets: [
    {
      label: 'DNS Queries',
      data: [120, 190, 300, 500, 200, 300],
      borderColor: '#3b82f6',
      backgroundColor: '#3b82f6',
      fill: false,
    },
  ],
};

const doughnutData = {
  labels: ['Success', 'Error', 'Timeout'],
  datasets: [
    {
      label: 'Response Types',
      data: [85, 10, 5],
    },
  ],
};

export const LineChart: Story = {
  args: {
    title: 'DNS Queries Over Time',
    data: sampleData,
    type: 'line',
    height: 300,
  },
};

export const BarChart: Story = {
  args: {
    title: 'Monthly Traffic',
    data: sampleData,
    type: 'bar',
    height: 300,
  },
};

export const DoughnutChart: Story = {
  args: {
    title: 'Response Distribution',
    data: doughnutData,
    type: 'doughnut',
    height: 300,
  },
};

export const Loading: Story = {
  args: {
    title: 'Loading Chart',
    loading: true,
    height: 300,
  },
};

export const Error: Story = {
  args: {
    title: 'Error Chart',
    error: 'Failed to load chart data',
    height: 300,
  },
};
