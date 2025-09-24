import type { Meta, StoryObj } from '@storybook/preact';
import { Input } from './Input';

const meta: Meta<typeof Input> = {
  title: 'Components/Input',
  component: Input,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component: `
A simple, clean input component with validation states.

## Usage
\`\`\`tsx
<Input label="Username" placeholder="Enter username" />
\`\`\`

## Features
- Label support
- Error states
- Help text
- Required field indicator
- Clean, minimal design
        `,
      },
    },
  },
  argTypes: {
    label: {
      control: 'text',
      description: 'Input label',
    },
    placeholder: {
      control: 'text',
      description: 'Placeholder text',
    },
    error: {
      control: 'text',
      description: 'Error message',
    },
    hint: {
      control: 'text',
      description: 'Help text',
    },
    required: {
      control: 'boolean',
      description: 'Required field',
    },
    disabled: {
      control: 'boolean',
      description: 'Disabled state',
    },
  },
};

export default meta;
type Story = StoryObj<typeof Input>;

export const Default: Story = {
  args: {
    placeholder: 'Enter text...',
  },
};

export const WithLabel: Story = {
  args: {
    label: 'Username',
    placeholder: 'Enter username',
  },
};

export const WithHint: Story = {
  args: {
    label: 'Email',
    placeholder: 'Enter email address',
    hint: 'We will never share your email',
  },
};

export const WithError: Story = {
  args: {
    label: 'Password',
    type: 'password',
    placeholder: 'Enter password',
    error: 'Password must be at least 8 characters',
  },
};

export const Required: Story = {
  args: {
    label: 'Full Name',
    placeholder: 'Enter your full name',
    required: true,
  },
};

export const Disabled: Story = {
  args: {
    label: 'Disabled Field',
    placeholder: 'This field is disabled',
    disabled: true,
  },
};

export const AllStates: Story = {
  render: () => (
    <div className="space-y-6 w-80">
      <Input label="Normal" placeholder="Normal input" />
      <Input label="With Hint" placeholder="Input with hint" hint="This is help text" />
      <Input label="With Error" placeholder="Input with error" error="This field has an error" />
      <Input label="Required" placeholder="Required input" required />
      <Input label="Disabled" placeholder="Disabled input" disabled />
    </div>
  ),
};