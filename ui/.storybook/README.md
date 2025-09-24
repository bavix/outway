# Storybook Configuration

This Storybook is configured for demonstrating and developing Outway Admin UI components.

## 🚀 Getting Started

```bash
npm run storybook
```

Storybook will be available at: http://localhost:6006

## 📁 Structure

```
.storybook/
├── main.ts          # Storybook configuration
├── preview.ts       # Global settings and decorators
├── manager.js       # Storybook UI settings
└── README.md        # This file

src/stories/
├── Introduction.stories.tsx     # Design system introduction
├── ComponentShowcase.stories.tsx # Interactive component gallery
└── DesignTokens.stories.tsx    # Design tokens and styles
```

## 🎨 Features

### Themes
- Automatic switching between light and dark themes
- User choice persistence in localStorage
- System theme detection

### Responsiveness
- Built-in viewports for testing on different devices
- Responsive components
- Mobile-first approach

### Documentation
- Automatic documentation generation
- Interactive examples
- Best practices usage

## 🛠 Development

### Adding New Stories

1. Create a `ComponentName.stories.tsx` file in the component folder
2. Use TypeScript for typing
3. Add documentation in `docs.description.component`
4. Configure controls in `argTypes`

### Example Story

```typescript
import type { Meta, StoryObj } from '@storybook/preact';
import { MyComponent } from './MyComponent';

const meta: Meta<typeof MyComponent> = {
  title: 'Components/MyComponent',
  component: MyComponent,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component: 'Component description and usage',
      },
    },
  },
  argTypes: {
    variant: {
      control: { type: 'select' },
      options: ['primary', 'secondary'],
      description: 'Style variant',
    },
  },
};

export default meta;
type Story = StoryObj<typeof MyComponent>;

export const Default: Story = {
  args: {
    variant: 'primary',
  },
};
```

## 📚 Best Practices

### Organization
- Group components by categories
- Use descriptive names for stories
- Create interactive examples

### Documentation
- Describe the purpose of each component
- Specify usage variants
- Add code examples

### Testing
- Test components in different states
- Check responsiveness
- Test accessibility

## 🎯 Goals

1. **Documentation** - Complete documentation of all components
2. **Development** - Convenient environment for developing new components
3. **Testing** - Component testing in isolation
4. **Design** - Visual design testing
5. **Quality** - Ensuring quality and consistency

## 🎨 Design System

This Storybook showcases our design system built with:
- **Tailwind CSS** - Utility-first CSS framework
- **Custom CSS Variables** - Design tokens for consistency
- **Dark Mode** - Full dark/light theme support
- **Typography** - Inter font family
- **Components** - Reusable, accessible components

## 🔧 Technical Stack

- **Framework**: Preact + TypeScript
- **Styling**: Tailwind CSS + Custom CSS
- **Build Tool**: Vite
- **Storybook**: Latest version with modern configuration
- **Icons**: Heroicons (via SVG)