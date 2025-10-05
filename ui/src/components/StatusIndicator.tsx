import { Badge } from './Badge.js';

interface StatusIndicatorProps {
  status: 'online' | 'offline' | 'warning' | 'error' | 'loading' | 'success' | 'pending';
  label?: string;
  showDot?: boolean;
  size?: 'sm' | 'md' | 'lg';
  className?: string;
}

export function StatusIndicator({
  status,
  label,
  showDot = true,
  size = 'md',
  className = ''
}: StatusIndicatorProps) {
  const getStatusConfig = (status: string) => {
    const configs = {
      online: {
        color: 'text-green-500',
        bgColor: 'bg-green-100 dark:bg-green-900/30',
        textColor: 'text-green-800 dark:text-green-300',
        label: 'Online'
      },
      offline: {
        color: 'text-gray-500',
        bgColor: 'bg-gray-100 dark:bg-gray-900/30',
        textColor: 'text-gray-800 dark:text-gray-300',
        label: 'Offline'
      },
      warning: {
        color: 'text-yellow-500',
        bgColor: 'bg-yellow-100 dark:bg-yellow-900/30',
        textColor: 'text-yellow-800 dark:text-yellow-300',
        label: 'Warning'
      },
      error: {
        color: 'text-red-500',
        bgColor: 'bg-red-100 dark:bg-red-900/30',
        textColor: 'text-red-800 dark:text-red-300',
        label: 'Error'
      },
      loading: {
        color: 'text-blue-500',
        bgColor: 'bg-blue-100 dark:bg-blue-900/30',
        textColor: 'text-blue-800 dark:text-blue-300',
        label: 'Loading'
      },
      success: {
        color: 'text-green-500',
        bgColor: 'bg-green-100 dark:bg-green-900/30',
        textColor: 'text-green-800 dark:text-green-300',
        label: 'Success'
      },
      pending: {
        color: 'text-orange-500',
        bgColor: 'bg-orange-100 dark:bg-orange-900/30',
        textColor: 'text-orange-800 dark:text-orange-300',
        label: 'Pending'
      }
    };
    return configs[status as keyof typeof configs] || configs.offline;
  };

  const config = getStatusConfig(status);
  const displayLabel = label || config.label;

  const getSizeClasses = (size: 'sm' | 'md' | 'lg') => {
    const sizes = {
      sm: 'w-2 h-2',
      md: 'w-3 h-3',
      lg: 'w-4 h-4'
    };
    return sizes[size];
  };

  const dotSize = getSizeClasses(size);

  if (status === 'loading') {
    return (
      <div className={`flex items-center space-x-2 ${className}`}>
        {showDot && (
          <div className={`${dotSize} ${config.color} animate-pulse rounded-full`}></div>
        )}
        <span className={`text-sm ${config.textColor}`}>{displayLabel}</span>
      </div>
    );
  }

  return (
    <div className={`flex items-center space-x-2 ${className}`}>
      {showDot && (
        <div className={`${dotSize} ${config.color} rounded-full`}></div>
      )}
      <Badge
        variant="secondary"
        className={`${config.bgColor} ${config.textColor} text-xs`}
      >
        {displayLabel}
      </Badge>
    </div>
  );
}
