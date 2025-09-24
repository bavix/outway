import type { Meta, StoryObj } from '@storybook/preact';
import { Button } from '../components/Button';
import { Input } from '../components/Input';
import { Card } from '../components/Card';
import { Badge } from '../components/Badge';
import { StatsCard } from '../components/StatsCard';

const meta: Meta = {
  title: 'Showcase/Component Gallery',
  parameters: {
    layout: 'fullscreen',
    docs: {
      description: {
        component: `
# Component Gallery

Interactive demonstration of design system components used in the admin interface.

## Features

- Live component examples
- Real-time interaction
- Clean, minimal design
- Black & white theme
        `,
      },
    },
  },
};

export default meta;
type Story = StoryObj;

export const Gallery: Story = {
  render: () => (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900 p-8">
      <div className="max-w-6xl mx-auto">
        <div className="text-center mb-12">
          <h1 className="text-3xl font-bold text-gray-900 dark:text-white mb-4">
            Component Gallery
          </h1>
          <p className="text-lg text-gray-600 dark:text-gray-400">
            Simple, elegant components for the admin interface
          </p>
        </div>

        {/* Stats Cards */}
        <section className="mb-12">
          <h2 className="text-2xl font-bold text-gray-900 dark:text-white mb-6">
            Statistics Cards
          </h2>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
            <StatsCard
              title="Users"
              value="1,234"
              change={{ value: 12.5, type: 'increase' }}
              icon={
                <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197m13.5-9a2.5 2.5 0 11-5 0 2.5 2.5 0 015 0z" />
                </svg>
              }
              color="blue"
            />
            <StatsCard
              title="Revenue"
              value="$45,678"
              change={{ value: 8.2, type: 'decrease' }}
              icon={
                <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1" />
                </svg>
              }
              color="green"
            />
            <StatsCard
              title="Errors"
              value="23"
              change={{ value: 5.1, type: 'increase' }}
              icon={
                <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.732-.833-2.5 0L4.268 18.5c-.77.833.192 2.5 1.732 2.5z" />
                </svg>
              }
              color="red"
            />
            <StatsCard
              title="Performance"
              value="98.5%"
              change={{ value: 2.3, type: 'increase' }}
              icon={
                <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
                </svg>
              }
              color="yellow"
            />
          </div>
        </section>

        {/* Forms */}
        <section className="mb-12">
          <h2 className="text-2xl font-bold text-gray-900 dark:text-white mb-6">
            Forms and Inputs
          </h2>
          
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
            <Card title="Buttons" subtitle="Button variants and sizes">
              <div className="space-y-4">
                <div className="flex flex-wrap gap-3">
                  <Button variant="primary">Primary</Button>
                  <Button variant="secondary">Secondary</Button>
                  <Button variant="outline">Outline</Button>
                </div>
                <div className="flex flex-wrap gap-3">
                  <Button size="sm">Small</Button>
                  <Button size="md">Medium</Button>
                  <Button size="lg">Large</Button>
                </div>
                <div className="flex flex-wrap gap-3">
                  <Button loading>Loading</Button>
                  <Button disabled>Disabled</Button>
                </div>
              </div>
            </Card>

            <Card title="Input Fields" subtitle="Input components with validation">
              <div className="space-y-4">
                <Input
                  label="Username"
                  placeholder="Enter username"
                  hint="Enter your username"
                />
                <Input
                  label="Password"
                  type="password"
                  placeholder="Enter password"
                />
                <Input
                  label="Email"
                  type="email"
                  placeholder="Enter email"
                  error="Please enter a valid email address"
                />
              </div>
            </Card>
          </div>
        </section>

        {/* Badges */}
        <section className="mb-12">
          <h2 className="text-2xl font-bold text-gray-900 dark:text-white mb-6">
            Badges and Counters
          </h2>
          
          <Card title="Badges" subtitle="For displaying counts and labels">
            <div className="space-y-4">
              <div className="flex flex-wrap gap-3 items-center">
                <Badge variant="default">Default</Badge>
                <Badge variant="primary">Primary</Badge>
                <Badge variant="secondary">Secondary</Badge>
              </div>
              <div className="flex flex-wrap gap-3 items-center">
                <Badge size="sm">Small</Badge>
                <Badge size="md">Medium</Badge>
              </div>
              <div className="flex flex-wrap gap-3 items-center">
                <span className="text-sm text-gray-600 dark:text-gray-400">Rules:</span>
                <Badge variant="primary">24</Badge>
                <span className="text-sm text-gray-600 dark:text-gray-400">Upstreams:</span>
                <Badge variant="primary">8</Badge>
                <span className="text-sm text-gray-600 dark:text-gray-400">Active:</span>
                <Badge variant="secondary">6</Badge>
              </div>
            </div>
          </Card>
        </section>

        {/* Cards */}
        <section className="mb-12">
          <h2 className="text-2xl font-bold text-gray-900 dark:text-white mb-6">
            Cards
          </h2>
          
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <Card title="Simple Card" subtitle="Basic card with content">
              <p className="text-gray-600 dark:text-gray-400">
                This is a simple card with title, subtitle, and content.
              </p>
            </Card>

            <Card title="Card with Actions" subtitle="Card with interactive elements">
              <div className="space-y-4">
                <p className="text-gray-600 dark:text-gray-400">
                  Cards can contain various interactive elements.
                </p>
                <div className="flex gap-3">
                  <Button variant="primary" size="sm">Action</Button>
                  <Button variant="outline" size="sm">Cancel</Button>
                </div>
              </div>
            </Card>
          </div>
        </section>
      </div>
    </div>
  ),
};