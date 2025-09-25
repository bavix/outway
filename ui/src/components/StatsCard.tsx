import type { JSX } from 'preact';

interface StatsCardProps {
  /** Stat title */
  title: string;
  /** Stat value */
  value: string | number;
  /** Change indicator */
  change?: {
    value: number;
    type: 'increase' | 'decrease';
  };
  /** Icon */
  icon?: JSX.Element;
  /** Color variant */
  color?: 'blue' | 'green' | 'red' | 'yellow' | 'purple' | 'indigo' | undefined;
  /** Additional CSS classes */
  className?: string;
  /** Loading state (ignored for now) */
  loading?: boolean | undefined;
}

export function StatsCard({ 
  title, 
  value, 
  change, 
  icon,
  color = 'blue',
  className = '' 
}: StatsCardProps) {
  
  const colorClasses = {
    blue: 'text-blue-600 dark:text-blue-400',
    green: 'text-green-600 dark:text-green-400',
    red: 'text-red-600 dark:text-red-400',
    yellow: 'text-yellow-600 dark:text-yellow-400',
    purple: 'text-purple-600 dark:text-purple-400',
    indigo: 'text-indigo-600 dark:text-indigo-400'
  };
  
  const statsClasses = `stats-card bg-white rounded-md p-6 shadow-sm dark:bg-black dark:shadow-none ${className}`.trim();

  return (
    <div className={statsClasses}>
      <div className="flex items-center justify-between">
        <div>
          <p className="text-sm text-gray-600 dark:text-gray-400">{title}</p>
          <p className="text-2xl font-bold text-black dark:text-white mt-1">{value}</p>
          {change && (
            <p className={`text-xs font-medium mt-2 ${change.type === 'increase' ? 'text-green-600' : 'text-red-600'}`}>
              {change.type === 'increase' ? '↗' : '↘'} {Math.abs(change.value)}%
            </p>
          )}
        </div>
        {icon && (
          <div className={`flex-shrink-0 w-6 h-6 flex items-center justify-center rounded-full ${colorClasses[color]}`}>
            {icon}
          </div>
        )}
      </div>
    </div>
  );
}