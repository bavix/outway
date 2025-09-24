import { addons } from '@storybook/manager-api';
import { themes } from '@storybook/theming';

addons.setConfig({
  theme: {
    ...themes.light,
    brandTitle: 'Outway Admin UI',
    brandUrl: 'https://github.com/bavix/dns2egress',
    brandImage: undefined,
    colorPrimary: '#000000',
    colorSecondary: '#000000',
    
    // UI
    appBg: '#ffffff',
    appContentBg: '#ffffff',
    appBorderColor: '#e5e7eb',
    appBorderRadius: 8,

    // Text colors
    textColor: '#111827',
    textInverseColor: '#ffffff',
    textMutedColor: '#6b7280',

    // Toolbar default and active colors
    barTextColor: '#6b7280',
    barSelectedColor: '#000000',
    barBg: '#ffffff',

    // Form colors
    inputBg: '#ffffff',
    inputBorder: '#d1d5db',
    inputTextColor: '#111827',
    inputBorderRadius: 6,

    // Button colors
    buttonBg: '#f3f4f6',
    buttonBorder: '#d1d5db',
    booleanBg: '#f3f4f6',
    booleanSelectedBg: '#000000',
  },
  
  panelPosition: 'bottom',
  showNav: true,
  showPanel: true,
  showToolbar: true,
  
  selectedPanel: 'storybook/docs/panel',
  initialActive: 'sidebar',
  
  sidebar: {
    showRoots: true,
    collapsedRoots: ['other'],
  },
  
  toolbar: {
    title: { hidden: false },
    zoom: { hidden: false },
    eject: { hidden: false },
    copy: { hidden: false },
    fullscreen: { hidden: false },
  },
});
