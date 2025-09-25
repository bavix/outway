import type { Meta, StoryObj } from '@storybook/preact';

const meta: Meta = {
  title: 'Design System/Tokens',
  parameters: {
    layout: 'fullscreen',
    docs: {
      description: {
        component: `
# Design Tokens

Design tokens are basic values that define the appearance of the entire design system. They include colors, typography, spacing, shadows, and other visual elements.

## Colors

Our color palette is based on accessibility and contrast principles.
        `,
      },
    },
  },
};

export default meta;
type Story = StoryObj;

export const Tokens: Story = {
  render: () => (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900 p-8">
      <div className="max-w-6xl mx-auto">
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-gray-900 dark:text-white mb-4">
            Design Tokens
          </h1>
          <p className="text-lg text-gray-600 dark:text-gray-400">
            Design tokens are basic values that define the appearance of the entire design system. They include colors, typography, spacing, shadows, and other visual elements.
          </p>
        </div>

        {/* Colors */}
        <section className="mb-12">
          <h2 className="text-2xl font-bold text-gray-900 dark:text-white mb-6">Color Palette</h2>
          
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
            <div>
              <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">Primary Colors</h3>
              <div className="space-y-3">
                <div className="flex items-center gap-4">
                  <div className="w-12 h-12 bg-black dark:bg-white rounded-lg border border-gray-200 dark:border-gray-700"></div>
                  <div>
                    <div className="font-medium text-gray-900 dark:text-white">Black/White</div>
                    <div className="text-sm text-gray-500 dark:text-gray-400">Primary brand color</div>
                  </div>
                </div>
                <div className="flex items-center gap-4">
                  <div className="w-12 h-12 bg-gray-500 rounded-lg"></div>
                  <div>
                    <div className="font-medium text-gray-900 dark:text-white">Gray</div>
                    <div className="text-sm text-gray-500 dark:text-gray-400">Neutral color</div>
                  </div>
                </div>
              </div>
            </div>

            <div>
              <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">Status Colors</h3>
              <div className="space-y-3">
                <div className="flex items-center gap-4">
                  <div className="w-12 h-12 bg-green-500 rounded-lg"></div>
                  <div>
                    <div className="font-medium text-gray-900 dark:text-white">Success</div>
                    <div className="text-sm text-gray-500 dark:text-gray-400">Positive actions</div>
                  </div>
                </div>
                <div className="flex items-center gap-4">
                  <div className="w-12 h-12 bg-yellow-500 rounded-lg"></div>
                  <div>
                    <div className="font-medium text-gray-900 dark:text-white">Warning</div>
                    <div className="text-sm text-gray-500 dark:text-gray-400">Caution states</div>
                  </div>
                </div>
                <div className="flex items-center gap-4">
                  <div className="w-12 h-12 bg-red-500 rounded-lg"></div>
                  <div>
                    <div className="font-medium text-gray-900 dark:text-white">Error</div>
                    <div className="text-sm text-gray-500 dark:text-gray-400">Error states</div>
                  </div>
                </div>
                <div className="flex items-center gap-4">
                  <div className="w-12 h-12 bg-blue-500 rounded-lg"></div>
                  <div>
                    <div className="font-medium text-gray-900 dark:text-white">Info</div>
                    <div className="text-sm text-gray-500 dark:text-gray-400">Informational content</div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* Typography */}
        <section className="mb-12">
          <h2 className="text-2xl font-bold text-gray-900 dark:text-white mb-6">Typography</h2>
          
          <div className="space-y-6">
            <div>
              <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-3">Font Family</h3>
              <div className="space-y-2">
                <div className="text-2xl font-bold text-gray-900 dark:text-white">Inter Bold</div>
                <div className="text-xl font-semibold text-gray-900 dark:text-white">Inter Semibold</div>
                <div className="text-lg font-medium text-gray-900 dark:text-white">Inter Medium</div>
                <div className="text-base font-normal text-gray-900 dark:text-white">
                  Body text - main text for reading. Lorem ipsum dolor sit amet, consectetur adipiscing elit.
                </div>
              </div>
            </div>

            <div>
              <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-3">Text Sizes</h3>
              <div className="space-y-2">
                <div className="text-sm text-gray-900 dark:text-white">
                  Small text - additional information and captions.
                </div>
                <div className="text-xs text-gray-900 dark:text-white">
                  Extra small text - labels and helper text.
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* Spacing */}
        <section className="mb-12">
          <h2 className="text-2xl font-bold text-gray-900 dark:text-white mb-6">Spacing and Sizes</h2>
          
          <div className="space-y-4">
            <div className="flex items-center gap-4">
              <div className="w-2 h-2 bg-gray-400 rounded-full"></div>
              <span className="text-sm text-gray-600 dark:text-gray-400">2px - xs</span>
            </div>
            <div className="flex items-center gap-4">
              <div className="w-4 h-4 bg-gray-400 rounded"></div>
              <span className="text-sm text-gray-600 dark:text-gray-400">4px - sm</span>
            </div>
            <div className="flex items-center gap-4">
              <div className="w-6 h-6 bg-gray-400 rounded"></div>
              <span className="text-sm text-gray-600 dark:text-gray-400">6px - md</span>
            </div>
            <div className="flex items-center gap-4">
              <div className="w-8 h-8 bg-gray-400 rounded"></div>
              <span className="text-sm text-gray-600 dark:text-gray-400">8px - lg</span>
            </div>
            <div className="flex items-center gap-4">
              <div className="w-12 h-12 bg-gray-400 rounded"></div>
              <span className="text-sm text-gray-600 dark:text-gray-400">12px - xl</span>
            </div>
            <div className="flex items-center gap-4">
              <div className="w-16 h-16 bg-gray-400 rounded"></div>
              <span className="text-sm text-gray-600 dark:text-gray-400">16px - 2xl</span>
            </div>
          </div>
        </section>

        {/* Shadows */}
        <section className="mb-12">
          <h2 className="text-2xl font-bold text-gray-900 dark:text-white mb-6">Shadows</h2>
          
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            <div className="bg-white dark:bg-gray-800 rounded-lg p-6 shadow-sm border border-gray-200 dark:border-gray-700">
              <h4 className="font-semibold text-gray-900 dark:text-white mb-2">Small Shadow</h4>
              <p className="text-sm text-gray-600 dark:text-gray-400">Subtle elevation</p>
            </div>
            <div className="bg-white dark:bg-gray-800 rounded-lg p-6 shadow-md border border-gray-200 dark:border-gray-700">
              <h4 className="font-semibold text-gray-900 dark:text-white mb-2">Medium Shadow</h4>
              <p className="text-sm text-gray-600 dark:text-gray-400">Standard elevation</p>
            </div>
            <div className="bg-white dark:bg-gray-800 rounded-lg p-6 shadow-lg border border-gray-200 dark:border-gray-700">
              <h4 className="font-semibold text-gray-900 dark:text-white mb-2">Large Shadow</h4>
              <p className="text-sm text-gray-600 dark:text-gray-400">High elevation</p>
            </div>
          </div>
        </section>
      </div>
    </div>
  ),
};