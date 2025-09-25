// no imports needed

interface DividerProps {
  /** Divider orientation */
  orientation?: 'horizontal' | 'vertical';
  /** Divider text */
  text?: string;
  /** Divider variant */
  variant?: 'solid' | 'dashed' | 'dotted';
  /** Additional CSS classes */
  className?: string;
}

export function Divider({ 
  orientation = 'horizontal',
  text,
  variant = 'solid',
  className = ''
}: DividerProps) {
  const variantClasses = {
    solid: 'border-solid',
    dashed: 'border-dashed',
    dotted: 'border-dotted'
  };

  if (orientation === 'vertical') {
    return (
      <div className={`border-l border-gray-200 dark:border-gray-700 ${variantClasses[variant]} ${className}`} />
    );
  }

  if (text) {
    return (
      <div className={`relative ${className}`}>
        <div className="absolute inset-0 flex items-center">
          <div className={`w-full border-t border-gray-200 dark:border-gray-700 ${variantClasses[variant]}`} />
        </div>
        <div className="relative flex justify-center text-sm">
          <span className="px-3 bg-white dark:bg-gray-900 text-gray-500 dark:text-gray-400">
            {text}
          </span>
        </div>
      </div>
    );
  }

  return (
    <div className={`border-t border-gray-200 dark:border-gray-700 ${variantClasses[variant]} ${className}`} />
  );
}
