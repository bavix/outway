import type { StorybookConfig } from '@storybook/preact';

const config: StorybookConfig = {
  stories: ['../src/**/*.stories.@(js|jsx|ts|tsx)'],
  addons: [
    '@storybook/addon-essentials',
  ],
  framework: {
    name: '@storybook/preact-webpack5',
    options: {},
  },
  docs: {
    autodocs: 'tag',
  },
  typescript: {
    check: false,
    reactDocgen: 'react-docgen-typescript',
  },
  webpackFinal: async (config) => {
    // Ensure TypeScript files are handled properly
    config.module = config.module || {};
    config.module.rules = config.module.rules || [];
    
    // Add TypeScript rule if not present
    const hasTypeScriptRule = config.module.rules.some((rule: any) => 
      rule.test && rule.test.toString().includes('tsx?')
    );
    
    if (!hasTypeScriptRule) {
      config.module.rules.push({
        test: /\.tsx?$/,
        use: [
          {
            loader: 'ts-loader',
            options: {
              transpileOnly: true,
            },
          },
        ],
        exclude: /node_modules/,
      });
    }

          // Add PostCSS support for Tailwind CSS
          const cssRule = config.module.rules.find((rule: any) => 
            rule.test && rule.test.toString().includes('css')
          );

          if (cssRule && cssRule.use) {
            cssRule.use = [
              'style-loader',
              'css-loader',
              {
                loader: 'postcss-loader',
                options: {
                  postcssOptions: {
                    plugins: [
                      require('tailwindcss'),
                      require('autoprefixer'),
                    ],
                  },
                },
              },
            ];
          }
    
    return config;
  },
};

export default config;
