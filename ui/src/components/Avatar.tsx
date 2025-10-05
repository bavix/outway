interface AvatarProps {
  initials: string;
  size?: 'sm' | 'md' | 'lg' | 'xl';
  variant?: 'primary' | 'secondary' | 'success' | 'warning' | 'danger';
  className?: string;
}

export function Avatar({ 
  initials, 
  size = 'md', 
  variant = 'primary',
  className = ''
}: AvatarProps) {
  const getSizeClasses = (size: 'sm' | 'md' | 'lg' | 'xl') => {
    const sizes = {
      sm: 'w-6 h-6 text-xs',
      md: 'w-8 h-8 text-sm',
      lg: 'w-12 h-12 text-lg',
      xl: 'w-16 h-16 text-xl'
    };
    return sizes[size];
  };

  const getVariantClasses = (variant: 'primary' | 'secondary' | 'success' | 'warning' | 'danger') => {
    const variants = {
      primary: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300',
      secondary: 'bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-300',
      success: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300',
      warning: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-300',
      danger: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300'
    };
    return variants[variant];
  };

  const sizeClasses = getSizeClasses(size);
  const variantClasses = getVariantClasses(variant);

  return (
    <div className={`${sizeClasses} ${variantClasses} rounded-full flex items-center justify-center font-medium ${className}`}>
      {initials}
    </div>
  );
}
