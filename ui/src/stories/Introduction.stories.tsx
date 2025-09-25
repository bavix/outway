import type { Meta, StoryObj } from '@storybook/preact';

const meta: Meta = {
  title: 'Introduction',
  parameters: {
    layout: 'fullscreen',
    docs: {
      description: {
        component: `
# Outway Admin UI Design System

Welcome to the Outway Admin UI Design System! This system is created for building modern, professional, and intuitive administrative panel interfaces.

## 🎨 Design Principles

- **Simplicity**: Minimalist design without unnecessary elements
- **Consistency**: Uniform components and patterns
- **Accessibility**: Support for dark/light themes and responsiveness
- **Performance**: Fast loading and responsive interface

## 🌙 Themes

The system supports automatic switching between light and dark themes with user choice persistence.

## 📱 Responsiveness

All components are adapted to work on mobile devices, tablets, and desktops.

## 🚀 Components

Explore available components in the sidebar. Each component has:
- Interactive examples
- Various usage options
- API documentation
- Controls for customization
        `,
      },
    },
  },
};

export default meta;
type Story = StoryObj;

export const Welcome: Story = {
  render: () => (
    <div className="py-8">
      <div className="max-w-4xl mx-auto">
        <div className="mb-10">
          <h1 className="text-3xl font-bold text-gray-900 dark:text-white mb-3">
            Outway Admin UI
          </h1>
          <p className="text-base text-gray-600 dark:text-gray-400">
            Modern design system for administrative panels
          </p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-10">
          <div className="bg-white dark:bg-gray-800 rounded-lg p-5 shadow-sm border border-gray-200 dark:border-gray-700">
            <h3 className="text-base font-semibold text-gray-900 dark:text-white mb-1">Fast</h3>
            <p className="text-sm text-gray-600 dark:text-gray-400">Optimized components for performance</p>
          </div>
          <div className="bg-white dark:bg-gray-800 rounded-lg p-5 shadow-sm border border-gray-200 dark:border-gray-700">
            <h3 className="text-base font-semibold text-gray-900 dark:text-white mb-1">Reliable</h3>
            <p className="text-sm text-gray-600 dark:text-gray-400">Typed and tested components</p>
          </div>
          <div className="bg-white dark:bg-gray-800 rounded-lg p-5 shadow-sm border border-gray-200 dark:border-gray-700">
            <h3 className="text-base font-semibold text-gray-900 dark:text-white mb-1">Simple</h3>
            <p className="text-sm text-gray-600 dark:text-gray-400">Clean, unobtrusive styles</p>
          </div>
        </div>

        <div className="bg-white dark:bg-gray-800 rounded-lg p-6 shadow-sm border border-gray-200 dark:border-gray-700">
          <h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-3">Getting started</h2>
          <p className="text-sm text-gray-600 dark:text-gray-400">
            Explore components in the sidebar. Each story contains interactive examples and concise docs.
          </p>
        </div>
      </div>
    </div>
  ),
};
