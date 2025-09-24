// no imports needed

interface ProgressProps {
  /** Progress value (0-100) */
  value: number;
  /** Progress size */
  size?: 'sm' | 'md' | 'lg';
  /** Progress color */
  color?: 'primary' | 'secondary' | 'success' | 'warning' | 'error';
  /** Show percentage */
  showPercentage?: boolean;
  /** Additional CSS classes */
  className?: string;
}

export function Progress({ 
  value, 
  size = 'md', 
  color = 'primary',
  showPercentage = false,
  className = ''
}: ProgressProps) {
  const sizeClasses = {
    sm: 'h-2',
    md: 'h-3',
    lg: 'h-4'
  };

  const colorClasses = {
    primary: 'bg-blue-600',
    secondary: 'bg-gray-600',
    success: 'bg-green-600',
    warning: 'bg-yellow-600',
    error: 'bg-red-600'
  };

  const clampedValue = Math.min(Math.max(value, 0), 100);

  return (
    <div className={`w-full ${className}`}>
      <div className={`${sizeClasses[size]} bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden`}>
        <div
          className={`${sizeClasses[size]} ${colorClasses[color]} rounded-full transition-all duration-300 ease-out`}
          style={{ width: `${clampedValue}%` }}
        />
      </div>
      {showPercentage && (
        <div className="mt-1 text-right">
          <span className="text-sm text-gray-600 dark:text-gray-400">
            {clampedValue.toFixed(0)}%
          </span>
        </div>
      )}
    </div>
  );
}

// Circular progress component
interface CircularProgressProps {
  /** Progress value (0-100) */
  value: number;
  /** Progress size */
  size?: number;
  /** Progress color */
  color?: 'primary' | 'secondary' | 'success' | 'warning' | 'error';
  /** Show percentage */
  showPercentage?: boolean;
  /** Additional CSS classes */
  className?: string;
}

export function CircularProgress({ 
  value, 
  size = 64, 
  color = 'primary',
  showPercentage = true,
  className = ''
}: CircularProgressProps) {
  const colorClasses = {
    primary: 'stroke-blue-600',
    secondary: 'stroke-gray-600',
    success: 'stroke-green-600',
    warning: 'stroke-yellow-600',
    error: 'stroke-red-600'
  };

  const clampedValue = Math.min(Math.max(value, 0), 100);
  const radius = (size - 8) / 2;
  const circumference = 2 * Math.PI * radius;
  const strokeDasharray = `${circumference} ${circumference}`;
  const strokeDashoffset = circumference - (clampedValue / 100) * circumference;

  return (
    <div className={`relative ${className}`} style={{ width: size, height: size }}>
      <svg
        width={size}
        height={size}
        className="transform -rotate-90"
      >
        <circle
          cx={size / 2}
          cy={size / 2}
          r={radius}
          stroke="currentColor"
          strokeWidth="4"
          fill="none"
          className="text-gray-200 dark:text-gray-700"
        />
        <circle
          cx={size / 2}
          cy={size / 2}
          r={radius}
          stroke="currentColor"
          strokeWidth="4"
          fill="none"
          strokeDasharray={strokeDasharray}
          strokeDashoffset={strokeDashoffset}
          className={`${colorClasses[color]} transition-all duration-500 ease-out`}
          strokeLinecap="round"
        />
      </svg>
      {showPercentage && (
        <div className="absolute inset-0 flex items-center justify-center">
          <span className="text-sm font-medium text-gray-900 dark:text-gray-100">
            {clampedValue.toFixed(0)}%
          </span>
        </div>
      )}
    </div>
  );
}
