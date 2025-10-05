import type { Meta, StoryObj } from '@storybook/preact';
import { FormField, SelectField } from './FormField.js';
import { useState } from 'preact/hooks';

const meta: Meta<typeof FormField> = {
  title: 'Components/FormField',
  component: FormField,
  parameters: {
    layout: 'padded',
  },
  argTypes: {
    type: {
      control: 'select',
      options: ['text', 'email', 'password', 'number', 'tel', 'url'],
    },
    required: {
      control: 'boolean',
    },
    disabled: {
      control: 'boolean',
    },
  },
};

export default meta;
type Story = StoryObj<typeof FormField>;

// Wrapper component for controlled inputs
function FormFieldWrapper(props: any) {
  const [value, setValue] = useState(props.value || '');
  return <FormField {...props} value={value} onChange={setValue} />;
}

export const Default: Story = {
  render: () => <FormFieldWrapper label="Name" placeholder="Enter your name" />,
};

export const Email: Story = {
  render: () => <FormFieldWrapper label="Email" type="email" placeholder="Enter your email" />,
};

export const Password: Story = {
  render: () => <FormFieldWrapper label="Password" type="password" placeholder="Enter your password" />,
};

export const Number: Story = {
  render: () => <FormFieldWrapper label="Age" type="number" placeholder="Enter your age" />,
};

export const Required: Story = {
  render: () => <FormFieldWrapper label="Required Field" placeholder="This field is required" required />,
};

export const WithError: Story = {
  render: () => (
    <FormFieldWrapper
      label="Email"
      type="email"
      placeholder="Enter your email"
      error="Please enter a valid email address"
    />
  ),
};

export const WithHelpText: Story = {
  render: () => (
    <FormFieldWrapper
      label="Password"
      type="password"
      placeholder="Enter your password"
      helpText="Password must be at least 8 characters long"
    />
  ),
};

export const Disabled: Story = {
  render: () => (
    <FormFieldWrapper
      label="Disabled Field"
      placeholder="This field is disabled"
      disabled
    />
  ),
};

export const WithValue: Story = {
  render: () => (
    <FormFieldWrapper
      label="Pre-filled Field"
      placeholder="Enter text"
      value="This field has a value"
    />
  ),
};

// SelectField stories
// const SelectFieldMeta: Meta<typeof SelectField> = {
//   title: 'Components/SelectField',
//   component: SelectField,
//   parameters: {
//     layout: 'padded',
//   },
// };

export const SelectDefault: Story = {
  render: () => (
    <SelectField
      label="Country"
      value=""
      onChange={() => {}}
      options={[
        { value: 'us', label: 'United States' },
        { value: 'ca', label: 'Canada' },
        { value: 'uk', label: 'United Kingdom' },
        { value: 'de', label: 'Germany' },
        { value: 'fr', label: 'France' },
      ]}
      placeholder="Select a country"
    />
  ),
};

export const SelectWithValue: Story = {
  render: () => (
    <SelectField
      label="Role"
      value="admin"
      onChange={() => {}}
      options={[
        { value: 'admin', label: 'Administrator' },
        { value: 'user', label: 'User' },
        { value: 'guest', label: 'Guest' },
      ]}
    />
  ),
};

export const SelectRequired: Story = {
  render: () => (
    <SelectField
      label="Required Selection"
      value=""
      onChange={() => {}}
      options={[
        { value: 'option1', label: 'Option 1' },
        { value: 'option2', label: 'Option 2' },
        { value: 'option3', label: 'Option 3' },
      ]}
      placeholder="Please select an option"
      required
    />
  ),
};

export const SelectWithError: Story = {
  render: () => (
    <SelectField
      label="Status"
      value=""
      onChange={() => {}}
      options={[
        { value: 'active', label: 'Active' },
        { value: 'inactive', label: 'Inactive' },
        { value: 'pending', label: 'Pending' },
      ]}
      placeholder="Select status"
      error="Please select a valid status"
    />
  ),
};

export const SelectDisabled: Story = {
  render: () => (
    <SelectField
      label="Disabled Selection"
      value="option1"
      onChange={() => {}}
      options={[
        { value: 'option1', label: 'Option 1' },
        { value: 'option2', label: 'Option 2' },
      ]}
      disabled
    />
  ),
};

export const FormExample: Story = {
  render: () => (
    <div className="max-w-md mx-auto space-y-4">
      <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
        User Registration
      </h3>
      <FormFieldWrapper
        label="Full Name"
        placeholder="Enter your full name"
        required
      />
      <FormFieldWrapper
        label="Email"
        type="email"
        placeholder="Enter your email"
        required
      />
      <FormFieldWrapper
        label="Password"
        type="password"
        placeholder="Enter your password"
        helpText="Password must be at least 8 characters"
        required
      />
      <SelectField
        label="Country"
        value=""
        onChange={() => {}}
        options={[
          { value: 'us', label: 'United States' },
          { value: 'ca', label: 'Canada' },
          { value: 'uk', label: 'United Kingdom' },
        ]}
        placeholder="Select your country"
        required
      />
    </div>
  ),
};
