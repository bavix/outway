import type { JSX, ComponentChildren } from 'preact';

interface ButtonProps extends JSX.HTMLAttributes<HTMLButtonElement> {
  /** Button variant */
  variant?: 'primary' | 'secondary' | 'outline' | 'danger' | undefined;
  /** Button size */
  size?: 'sm' | 'md' | 'lg';
  /** Button content */
  children: ComponentChildren;
  /** Disabled state */
  disabled?: boolean | undefined;
  /** Loading state */
  loading?: boolean | undefined;
  /** Full width button */
  fullWidth?: boolean | undefined;
  /** Button type */
  type?: 'button' | 'submit' | 'reset' | undefined;
}

export function Button({ 
  variant = 'primary', 
  size = 'md', 
  children, 
  disabled = false,
  loading = false,
  fullWidth = false,
  className = '',
  ...props 
}: ButtonProps) {
  
  const baseClasses = 'btn inline-flex items-center justify-center font-medium rounded-md disabled:opacity-50 disabled:cursor-not-allowed transition-all duration-200 ease-out';
  
  const variantClasses = {
    primary: 'btn-primary bg-blue-600 text-white hover:bg-blue-700 shadow-lg hover:shadow-xl shadow-blue-600/25 hover:shadow-blue-600/40 dark:bg-blue-600 dark:text-white dark:hover:bg-blue-700',
    secondary: 'btn-secondary bg-gray-100 text-gray-900 hover:bg-gray-200 shadow-md hover:shadow-lg dark:bg-gray-700 dark:text-gray-100 dark:hover:bg-gray-600',
    outline: 'btn-outline bg-white/80 dark:bg-gray-800/80 text-gray-900 dark:text-gray-100 border border-gray-300 dark:border-gray-600 hover:bg-gray-50 dark:hover:bg-gray-700 backdrop-blur-sm',
    danger: 'btn-danger bg-red-600 text-white hover:bg-red-700 shadow-lg hover:shadow-xl shadow-red-600/25 hover:shadow-red-600/40 dark:bg-red-600 dark:text-white dark:hover:bg-red-700'
  };
  
  const sizeClasses = {
    sm: 'btn-sm px-3 py-2 text-sm rounded-lg',
    md: 'btn-md px-4 py-2.5 text-sm rounded-lg',
    lg: 'btn-lg px-6 py-3 text-base rounded-lg'
  };
  
  const widthClass = fullWidth ? 'w-full' : '';
  const loadingClass = loading ? 'btn-loading' : '';
  
  const classes = `${baseClasses} ${variantClasses[variant]} ${sizeClasses[size]} ${widthClass} ${loadingClass} ${className}`.trim();
  
  return (
    <button 
      className={classes} 
      disabled={disabled || loading}
      {...props}
    >
      {loading && (
        <svg className="btn-spinner animate-spin -ml-1 mr-2 h-4 w-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
        </svg>
      )}
      {children}
    </button>
  );
}