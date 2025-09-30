import type { JSX } from 'preact';

interface InputProps extends JSX.HTMLAttributes<HTMLInputElement> {
  /** Input label */
  label?: string;
  /** Error message */
  error?: string;
  /** Help text */
  hint?: string;
  /** Required field indicator */
  required?: boolean | undefined;
  /** Controlled value */
  value?: string | undefined;
  /** Placeholder text */
  placeholder?: string | undefined;
  /** Input type */
  type?: string | undefined;
  /** Disabled state */
  disabled?: boolean | undefined;
}

export function Input({ 
  label, 
  error, 
  hint, 
  required = false,
  className = '', 
  id,
  ...props 
}: InputProps) {
  const inputId = id || `input-${Math.random().toString(36).substr(2, 9)}`;
  
  const baseClasses = `input-field block w-full px-3 py-2 h-[42px] text-sm rounded-md border transition-all duration-200 ease-out focus:outline-none focus:ring-2 disabled:opacity-50 disabled:cursor-not-allowed`;
  
  const normalClasses = `bg-white text-gray-900 placeholder-gray-500 border-gray-300 hover:border-gray-400 focus:border-blue-500 focus:ring-blue-500 disabled:bg-gray-50 disabled:border-gray-300`;
  
  const darkClasses = `dark:bg-gray-800 dark:text-gray-100 dark:placeholder-gray-400 dark:border-gray-600 dark:hover:border-gray-500 dark:focus:border-blue-400 dark:focus:ring-blue-400 dark:disabled:bg-gray-700 dark:disabled:border-gray-600`;
  
  const errorClasses = error 
    ? `border-red-500 focus:border-red-500 focus:ring-red-500 dark:border-red-400 dark:focus:border-red-400 dark:focus:ring-red-400`
    : '';
  
  const inputClasses = `${baseClasses} ${normalClasses} ${darkClasses} ${errorClasses} ${className}`.trim();

  return (
    <div className="input-container space-y-1">
      {label && (
        <label htmlFor={inputId} className="input-label block text-sm font-medium text-gray-700 dark:text-gray-300">
          {label}
          {required && <span className="text-red-500 ml-1">*</span>}
        </label>
      )}
      
      <input
        id={inputId}
        className={inputClasses}
        {...props}
      />
      
      {hint && !error && (
        <p className="input-hint text-xs text-gray-500 dark:text-gray-400">{hint}</p>
      )}
      
      {error && (
        <p className="input-error text-xs text-red-600 dark:text-red-400">{error}</p>
      )}
    </div>
  );
}