import type { Meta, StoryObj } from '@storybook/preact';
import { AnimatedCard } from './AnimatedCard.js';

const meta: Meta<typeof AnimatedCard> = {
  title: 'Components/AnimatedCard',
  component: AnimatedCard,
  parameters: {
    layout: 'padded',
  },
  argTypes: {
    animation: {
      control: 'select',
      options: ['fade', 'slide', 'scale', 'bounce', 'pulse'],
    },
    delay: {
      control: { type: 'number', min: 0, max: 2000, step: 100 },
    },
    duration: {
      control: { type: 'number', min: 100, max: 2000, step: 100 },
    },
  },
};

export default meta;
type Story = StoryObj<typeof AnimatedCard>;

export const Fade: Story = {
  args: {
    animation: 'fade',
    children: (
      <div className="p-6">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-2">
          Fade Animation
        </h3>
        <p className="text-gray-600 dark:text-gray-400">
          This card fades in smoothly.
        </p>
      </div>
    ),
  },
};

export const Slide: Story = {
  args: {
    animation: 'slide',
    children: (
      <div className="p-6">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-2">
          Slide Animation
        </h3>
        <p className="text-gray-600 dark:text-gray-400">
          This card slides in from the left.
        </p>
      </div>
    ),
  },
};

export const Scale: Story = {
  args: {
    animation: 'scale',
    children: (
      <div className="p-6">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-2">
          Scale Animation
        </h3>
        <p className="text-gray-600 dark:text-gray-400">
          This card scales up from the center.
        </p>
      </div>
    ),
  },
};

export const Bounce: Story = {
  args: {
    animation: 'bounce',
    children: (
      <div className="p-6">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-2">
          Bounce Animation
        </h3>
        <p className="text-gray-600 dark:text-gray-400">
          This card bounces in with energy.
        </p>
      </div>
    ),
  },
};

export const Pulse: Story = {
  args: {
    animation: 'pulse',
    children: (
      <div className="p-6">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-2">
          Pulse Animation
        </h3>
        <p className="text-gray-600 dark:text-gray-400">
          This card pulses to draw attention.
        </p>
      </div>
    ),
  },
};

export const WithDelay: Story = {
  args: {
    animation: 'fade',
    delay: 500,
    children: (
      <div className="p-6">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-2">
          Delayed Animation
        </h3>
        <p className="text-gray-600 dark:text-gray-400">
          This card appears after 500ms delay.
        </p>
      </div>
    ),
  },
};

export const CustomDuration: Story = {
  args: {
    animation: 'scale',
    duration: 1000,
    children: (
      <div className="p-6">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-2">
          Slow Animation
        </h3>
        <p className="text-gray-600 dark:text-gray-400">
          This card animates over 1 second.
        </p>
      </div>
    ),
  },
};

export const WithCustomClass: Story = {
  args: {
    animation: 'fade',
    className: 'bg-gradient-to-r from-blue-500 to-purple-600 text-white',
    children: (
      <div className="p-6">
        <h3 className="text-lg font-semibold mb-2">
          Custom Styled Card
        </h3>
        <p className="text-blue-100">
          This card has custom styling with animation.
        </p>
      </div>
    ),
  },
};

export const AllAnimations: Story = {
  render: () => (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      <AnimatedCard  delay={0}>
        <div className="p-4">
          <h4 className="font-semibold text-gray-900 dark:text-white mb-2">Fade</h4>
          <p className="text-sm text-gray-600 dark:text-gray-400">Smooth fade in</p>
        </div>
      </AnimatedCard>
      
      <AnimatedCard  delay={100}>
        <div className="p-4">
          <h4 className="font-semibold text-gray-900 dark:text-white mb-2">Slide</h4>
          <p className="text-sm text-gray-600 dark:text-gray-400">Slide from left</p>
        </div>
      </AnimatedCard>
      
      <AnimatedCard  delay={200}>
        <div className="p-4">
          <h4 className="font-semibold text-gray-900 dark:text-white mb-2">Scale</h4>
          <p className="text-sm text-gray-600 dark:text-gray-400">Scale up effect</p>
        </div>
      </AnimatedCard>
      
      <AnimatedCard  delay={300}>
        <div className="p-4">
          <h4 className="font-semibold text-gray-900 dark:text-white mb-2">Bounce</h4>
          <p className="text-sm text-gray-600 dark:text-gray-400">Bouncy entrance</p>
        </div>
      </AnimatedCard>
      
      <AnimatedCard  delay={400}>
        <div className="p-4">
          <h4 className="font-semibold text-gray-900 dark:text-white mb-2">Pulse</h4>
          <p className="text-sm text-gray-600 dark:text-gray-400">Pulsing effect</p>
        </div>
      </AnimatedCard>
    </div>
  ),
};

export const StaggeredAnimation: Story = {
  render: () => (
    <div className="space-y-4">
      <AnimatedCard  delay={0}>
        <div className="p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
          <h4 className="font-semibold text-blue-900 dark:text-blue-100 mb-2">First Card</h4>
          <p className="text-sm text-blue-700 dark:text-blue-300">Appears immediately</p>
        </div>
      </AnimatedCard>
      
      <AnimatedCard  delay={200}>
        <div className="p-4 bg-green-50 dark:bg-green-900/20 rounded-lg">
          <h4 className="font-semibold text-green-900 dark:text-green-100 mb-2">Second Card</h4>
          <p className="text-sm text-green-700 dark:text-green-300">Appears after 200ms</p>
        </div>
      </AnimatedCard>
      
      <AnimatedCard  delay={400}>
        <div className="p-4 bg-purple-50 dark:bg-purple-900/20 rounded-lg">
          <h4 className="font-semibold text-purple-900 dark:text-purple-100 mb-2">Third Card</h4>
          <p className="text-sm text-purple-700 dark:text-purple-300">Appears after 400ms</p>
        </div>
      </AnimatedCard>
    </div>
  ),
};
