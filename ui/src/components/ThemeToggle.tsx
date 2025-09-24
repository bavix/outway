import { JSX } from 'preact';
import { useTheme, type Theme } from '../hooks/useTheme';

interface ThemeToggleProps extends JSX.HTMLAttributes<HTMLDivElement> {
  className?: string;
  showLabel?: boolean;
  variant?: 'select' | 'button';
}

export function ThemeToggle({ className = '', showLabel = false, variant = 'select', ...props }: ThemeToggleProps) {
  const { theme, resolvedTheme, setTheme } = useTheme();

  const handleThemeChange = (newTheme: Theme): void => {
    setTheme(newTheme);
  };

  const cycleTheme = () => {
    const themes: Theme[] = ['light', 'dark', 'auto'];
    const currentIndex = themes.indexOf(theme);
    const nextIndex = (currentIndex + 1) % themes.length;
    setTheme(themes[nextIndex]!);
  };

  const getThemeIcon = () => {
    switch (theme) {
      case 'light':
        return (
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z" />
          </svg>
        );
      case 'dark':
        return (
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z" />
          </svg>
        );
      case 'auto':
        return (
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
          </svg>
        );
    }
  };

  const getThemeLabel = () => {
    switch (theme) {
      case 'light':
        return 'Light theme';
      case 'dark':
        return 'Dark theme';
      case 'auto':
        return `Auto theme (${resolvedTheme})`;
    }
  };

  const options = [
    { value: 'light', label: 'Light', icon: '☀️' },
    { value: 'dark', label: 'Dark', icon: '🌙' },
    { value: 'auto', label: 'Auto', icon: '🔄' }
  ] as const;

  if (variant === 'button') {
    return (
      <div className={`flex items-center gap-2 ${className}`} {...props}>
        <button
          onClick={cycleTheme}
          className="p-2 rounded-md hover:bg-gray-100 dark:hover:bg-gray-800 text-gray-500 dark:text-gray-400 transition-colors duration-200"
          aria-label={getThemeLabel()}
          title={getThemeLabel()}
        >
          {getThemeIcon()}
        </button>
        {showLabel && (
          <span className="text-sm text-gray-600 dark:text-gray-300">
            {theme === 'auto' ? `Auto (${resolvedTheme})` : theme}
          </span>
        )}
      </div>
    );
  }

  return (
    <div className={`theme-toggle ${className}`} {...props}>
      <select
        value={theme}
        onChange={(e) => handleThemeChange(e.currentTarget.value as Theme)}
        className="theme-select"
      >
        {options.map(option => (
          <option key={option.value} value={option.value}>
            {option.icon} {option.label}
          </option>
        ))}
      </select>
    </div>
  );
}