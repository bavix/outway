import type { Meta, StoryObj } from '@storybook/preact';
import { Button } from './Button';
import { Input } from './Input';
import { Card } from './Card';
import { Badge } from './Badge';
import { IconBadge } from './IconBadge';
import { Avatar } from './Avatar';
import { StatusCard } from './StatusCard';
import { LoadingState } from './LoadingState';
import { EmptyState } from './EmptyState';
import { ActionButton } from './ActionButton';
import { Notification } from './Notification';
import { RoleBadge } from './RoleBadge';
import { PermissionsCard } from './PermissionsCard';

const meta: Meta = {
  title: 'Design System/Overview',
  parameters: {
    layout: 'padded',
  },
};

export default meta;
type Story = StoryObj;

const StarIcon = () => (
  <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
    <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z" />
  </svg>
);

const systemPermissions = [
  { name: 'system:view', description: 'View system information', category: 'System' },
  { name: 'system:manage', description: 'Manage system settings', category: 'System' },
];

export const CoreComponents: Story = {
  render: () => (
    <div className="space-y-8">
      <section>
        <h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-4">Buttons</h2>
        <div className="flex flex-wrap gap-4">
          <Button variant="primary">Primary</Button>
          <Button variant="secondary">Secondary</Button>
          <Button variant="outline">Outline</Button>
          <Button variant="danger">Danger</Button>
        </div>
      </section>

      <section>
        <h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-4">Inputs</h2>
        <div className="max-w-md space-y-4">
          <Input label="Text Input" placeholder="Enter text" />
          <Input label="Email Input" type="email" placeholder="Enter email" />
          <Input label="Password Input" type="password" placeholder="Enter password" />
        </div>
      </section>

      <section>
        <h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-4">Cards</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Card>
            <div className="p-4">
              <h3 className="font-semibold text-gray-900 dark:text-white mb-2">Basic Card</h3>
              <p className="text-gray-600 dark:text-gray-400">This is a basic card component.</p>
            </div>
          </Card>
          <StatusCard
            title="Status Card"
            description="System is running"
            count={42}
            status="active"
          />
        </div>
      </section>

      <section>
        <h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-4">Badges</h2>
        <div className="flex flex-wrap gap-4">
          <Badge variant="primary">Primary</Badge>
          <Badge variant="secondary">Secondary</Badge>
          <Badge variant="default">Default</Badge>
          <Badge variant="default">Default</Badge>
          <Badge variant="default">Default</Badge>
          <IconBadge icon={<StarIcon />} label="With Icon" />
        </div>
      </section>

      <section>
        <h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-4">Avatars</h2>
        <div className="flex items-center gap-4">
          <Avatar initials="JD" />
          <Avatar initials="AB" variant="secondary" />
          <Avatar initials="CD" variant="danger" />
        </div>
      </section>

      <section>
        <h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-4">Action Buttons</h2>
        <div className="flex flex-wrap gap-4">
          <ActionButton action="add" onClick={() => {}} />
          <ActionButton action="edit" onClick={() => {}} />
          <ActionButton action="delete" onClick={() => {}} />
          <ActionButton action="save" onClick={() => {}} />
        </div>
      </section>

      <section>
        <h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-4">States</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <LoadingState message="Loading data..." />
          <EmptyState title="No data" description="There are no items to display" />
        </div>
      </section>

      <section>
        <h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-4">Notifications</h2>
        <div className="space-y-2">
          <Notification type="info" title="Info" message="This is an info notification" />
          <Notification type="success" title="Success" message="Operation completed successfully" />
          <Notification type="warning" title="Warning" message="Please review your settings" />
          <Notification type="error" title="Error" message="Something went wrong" />
        </div>
      </section>

      <section>
        <h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-4">Domain Components</h2>
        <div className="space-y-4">
          <div className="flex items-center gap-4">
            <RoleBadge role="admin" />
            <RoleBadge role="user" />
            <RoleBadge role="guest" />
          </div>
          <PermissionsCard
            permissions={systemPermissions}
            title="System Permissions"
          />
        </div>
      </section>
    </div>
  ),
};