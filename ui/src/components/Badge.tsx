import type { JSX, ComponentChildren } from 'preact';

interface BadgeProps extends JSX.HTMLAttributes<HTMLSpanElement> {
  /** Badge content */
  children: ComponentChildren;
  /** Badge variant */
  variant?: 'default' | 'primary' | 'secondary';
  /** Badge size */
  size?: 'sm' | 'md';
}

export function Badge({ 
  children, 
  variant = 'default',
  size = 'md',
  className = '',
  ...props 
}: BadgeProps) {
  
  const baseClasses = 'badge inline-flex items-center rounded-md font-medium';
  
  const variantClasses = {
    default: 'bg-gray-100 text-gray-900 dark:bg-gray-800 dark:text-gray-100',
    primary: 'bg-blue-600 text-white dark:bg-blue-600 dark:text-white',
    secondary: 'bg-gray-200 text-gray-900 dark:bg-gray-700 dark:text-gray-100'
  };
  
  const sizeClasses = {
    sm: 'px-2 py-0.5 text-xs',
    md: 'px-2.5 py-1 text-sm'
  };
  
  const classes = `${baseClasses} ${variantClasses[variant]} ${sizeClasses[size]} ${className}`.trim();
  
  return (
    <span className={classes} {...props}>
      {children}
    </span>
  );
}
