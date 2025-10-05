import type { Meta, StoryObj } from '@storybook/preact';
import { IconBadge } from './IconBadge.js';

const meta: Meta<typeof IconBadge> = {
  title: 'Components/IconBadge',
  component: IconBadge,
  parameters: {
    layout: 'centered',
  },
  argTypes: {
    variant: {
      control: 'select',
      options: ['primary', 'secondary', 'default'],
    },
    size: {
      control: 'select',
      options: ['sm', 'md', 'lg'],
    },
  },
};

export default meta;
type Story = StoryObj<typeof IconBadge>;

const StarIcon = () => (
  <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
    <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z" />
  </svg>
);

const UserIcon = () => (
  <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
    <path fillRule="evenodd" d="M10 9a3 3 0 100-6 3 3 0 000 6zm-7 9a7 7 0 1114 0H3z" clipRule="evenodd" />
  </svg>
);

const ShieldIcon = () => (
  <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
    <path fillRule="evenodd" d="M2.166 4.999A11.954 11.954 0 0010 1.944 11.954 11.954 0 0017.834 5c.11.65.166 1.32.166 2.001 0 5.225-3.34 9.67-8 11.317C5.34 16.67 2 12.225 2 7c0-.682.057-1.35.166-2.001zm11.541 3.708a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
  </svg>
);

export const Default: Story = {
  args: {
    icon: <StarIcon />,
    label: 'Star',
  },
};

export const Primary: Story = {
  args: {
    icon: <UserIcon />,
    label: 'User',
    variant: 'primary',
  },
};

export const Secondary: Story = {
  args: {
    icon: <ShieldIcon />,
    label: 'Security',
    variant: 'secondary',
  },
};

export const Small: Story = {
  args: {
    icon: <StarIcon />,
    label: 'Small',
    size: 'sm',
  },
};

export const Large: Story = {
  args: {
    icon: <StarIcon />,
    label: 'Large',
    size: 'lg',
  },
};

export const WithoutIcon: Story = {
  args: {
    icon: null,
    label: 'No Icon',
  },
};

export const AllVariants: Story = {
  render: () => (
    <div className="flex flex-wrap gap-4">
      <IconBadge icon={<StarIcon />} label="Default" />
      <IconBadge icon={<UserIcon />} label="Primary" variant="primary" />
      <IconBadge icon={<ShieldIcon />} label="Secondary" variant="secondary" />
    </div>
  ),
};

export const AllSizes: Story = {
  render: () => (
    <div className="flex items-center gap-4">
      <IconBadge icon={<StarIcon />} label="Small" size="sm" />
      <IconBadge icon={<StarIcon />} label="Medium" size="md" />
      <IconBadge icon={<StarIcon />} label="Large" size="lg" />
    </div>
  ),
};
