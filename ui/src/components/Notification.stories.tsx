import type { Meta, StoryObj } from '@storybook/preact';
import { Notification } from './Notification.js';

const meta: Meta<typeof Notification> = {
  title: 'Components/Notification',
  component: Notification,
  parameters: {
    layout: 'padded',
  },
  argTypes: {
    type: {
      control: 'select',
      options: ['info', 'success', 'warning', 'error'],
    },
    variant: {
      control: 'select',
      options: ['default', 'filled', 'outlined'],
    },
    dismissible: {
      control: 'boolean',
    },
  },
};

export default meta;
type Story = StoryObj<typeof Notification>;

export const Info: Story = {
  args: {
    type: 'info',
    title: 'Information',
    message: 'This is an informational notification.',
  },
};

export const Success: Story = {
  args: {
    type: 'success',
    title: 'Success',
    message: 'Your action was completed successfully.',
  },
};

export const Warning: Story = {
  args: {
    type: 'warning',
    title: 'Warning',
    message: 'Please review your input before proceeding.',
  },
};

export const Error: Story = {
  args: {
    type: 'error',
    title: 'Error',
    message: 'Something went wrong. Please try again.',
  },
};

export const Filled: Story = {
  args: {
    type: 'success',
    variant: 'filled',
    title: 'Filled Notification',
    message: 'This notification uses the filled variant.',
  },
};

export const Outlined: Story = {
  args: {
    type: 'warning',
    variant: 'outlined',
    title: 'Outlined Notification',
    message: 'This notification uses the outlined variant.',
  },
};

export const Dismissible: Story = {
  args: {
    type: 'info',
    title: 'Dismissible',
    message: 'You can close this notification by clicking the X button.',
  },
};

export const WithoutTitle: Story = {
  args: {
    type: 'info',
    message: 'This notification has no title, just a message.',
  },
};

export const LongMessage: Story = {
  args: {
    type: 'warning',
    title: 'Long Message',
    message: 'This is a very long notification message that demonstrates how the component handles text that might wrap to multiple lines. The notification should maintain its layout and readability even with longer content.',
  },
};

export const WithAction: Story = {
  args: {
    type: 'info',
    title: 'Action Required',
    message: 'Please complete your profile setup.',
    action: {
      label: 'Complete Setup',
      onClick: () => alert('Action clicked!'),
    },
  },
};

export const WithCustomIcon: Story = {
  args: {
    type: 'info',
    title: 'Custom Icon',
    message: 'This notification has a custom icon.',
    icon: (
      <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
        <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clipRule="evenodd" />
      </svg>
    ),
  },
};

export const AllTypes: Story = {
  render: () => (
    <div className="space-y-4">
      <Notification
        type="info"
        title="Info"
        message="This is an informational notification."
      />
      <Notification
        type="success"
        title="Success"
        message="Operation completed successfully."
      />
      <Notification
        type="warning"
        title="Warning"
        message="Please review your settings."
      />
      <Notification
        type="error"
        title="Error"
        message="An error occurred during processing."
      />
    </div>
  ),
};

export const AllVariants: Story = {
  render: () => (
    <div className="space-y-4">
      <Notification
        type="info"
        title="Default"
        message="This is the default variant."
      />
      <Notification
        type="info"
        title="Filled"
        message="This is the filled variant."
      />
      <Notification
        type="info"
        title="Outlined"
        message="This is the outlined variant."
      />
    </div>
  ),
};

export const NotificationList: Story = {
  render: () => (
    <div className="space-y-3">
      <Notification
        type="success"
        title="Profile Updated"
        message="Your profile has been successfully updated."
      />
      <Notification
        type="warning"
        title="Password Expiring"
        message="Your password will expire in 7 days. Consider changing it soon."
      />
      <Notification
        type="error"
        title="Connection Failed"
        message="Unable to connect to the server. Please check your internet connection."
      />
      <Notification
        type="info"
        title="New Feature"
        message="Check out our new dashboard features in the latest update."
        // action={{
        //   label: 'Learn More',
        //   onClick: () => alert('Learn more clicked!'),
        // }}
        // dismissible
      />
    </div>
  ),
};
