interface LoadingStateProps {
  message?: string;
  size?: 'sm' | 'md' | 'lg';
  className?: string;
}

export function LoadingState({ 
  message = 'Loading...', 
  size = 'md',
  className = ''
}: LoadingStateProps) {
  const getSizeClasses = (size: 'sm' | 'md' | 'lg') => {
    const sizes = {
      sm: 'h-4 w-4',
      md: 'h-8 w-8',
      lg: 'h-12 w-12'
    };
    return sizes[size];
  };

  const sizeClasses = getSizeClasses(size);

  return (
    <div className={`flex items-center justify-center py-8 ${className}`}>
      <div className="flex flex-col items-center space-y-3">
        <div className={`animate-spin rounded-full border-b-2 border-blue-600 ${sizeClasses}`}></div>
        <span className="text-gray-600 dark:text-gray-400 text-sm">{message}</span>
      </div>
    </div>
  );
}
