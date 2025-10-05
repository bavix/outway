import type { Meta, StoryObj } from '@storybook/preact';
import { Avatar } from './Avatar.js';

const meta: Meta<typeof Avatar> = {
  title: 'Components/Avatar',
  component: Avatar,
  parameters: {
    layout: 'centered',
  },
  argTypes: {
    variant: {
      control: 'select',
      options: ['primary', 'secondary', 'danger'],
    },
    size: {
      control: 'select',
      options: ['sm', 'md', 'lg', 'xl'],
    },
  },
};

export default meta;
type Story = StoryObj<typeof Avatar>;

export const Default: Story = {
  args: {
    initials: 'JD',
  },
};

export const Primary: Story = {
  args: {
    initials: 'AB',
    variant: 'primary',
  },
};

export const Secondary: Story = {
  args: {
    initials: 'CD',
    variant: 'secondary',
  },
};

export const Danger: Story = {
  args: {
    initials: 'EF',
    variant: 'danger',
  },
};

export const Small: Story = {
  args: {
    initials: 'GH',
    size: 'sm',
  },
};

export const Medium: Story = {
  args: {
    initials: 'IJ',
    size: 'md',
  },
};

export const Large: Story = {
  args: {
    initials: 'KL',
    size: 'lg',
  },
};

export const ExtraLarge: Story = {
  args: {
    initials: 'MN',
    size: 'xl',
  },
};

export const AllVariants: Story = {
  render: () => (
    <div className="flex items-center gap-4">
      <Avatar initials="A" variant="primary" />
      <Avatar initials="B" variant="secondary" />
      <Avatar initials="C" variant="danger" />
    </div>
  ),
};

export const AllSizes: Story = {
  render: () => (
    <div className="flex items-center gap-4">
      <Avatar initials="A" size="sm" />
      <Avatar initials="B" size="md" />
      <Avatar initials="C" size="lg" />
      <Avatar initials="D" size="xl" />
    </div>
  ),
};

export const WithCustomClass: Story = {
  args: {
    initials: 'XY',
    className: 'ring-2 ring-blue-500',
  },
};
