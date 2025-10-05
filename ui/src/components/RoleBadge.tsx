import { IconBadge } from './IconBadge.js';

interface RoleBadgeProps {
  role: string;
  size?: 'sm' | 'md' | 'lg';
  showIcon?: boolean;
  className?: string;
}

export function RoleBadge({ 
  role, 
  size = 'md', 
  showIcon = true, 
  className = '' 
}: RoleBadgeProps) {
  const getRoleIcon = (role: string, size: 'sm' | 'md' | 'lg') => {
    const sizeClasses = {
      sm: 'w-4 h-4',
      md: 'w-5 h-5', 
      lg: 'w-6 h-6'
    };

    const iconSize = sizeClasses[size];

    if (role === 'admin') {
      return (
        <svg className={`${iconSize} text-red-600 dark:text-red-400`} fill="currentColor" viewBox="0 0 20 20">
          <path fillRule="evenodd" d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z" clipRule="evenodd" />
        </svg>
      );
    }
    return (
      <svg className={`${iconSize} text-blue-600 dark:text-blue-400`} fill="currentColor" viewBox="0 0 20 20">
        <path fillRule="evenodd" d="M10 9a3 3 0 100-6 3 3 0 000 6zm-7 9a7 7 0 1114 0H3z" clipRule="evenodd" />
      </svg>
    );
  };

  const getRoleVariant = (role: string) => {
    return role === 'admin' ? 'primary' : 'secondary';
  };

  const getRoleClassName = (role: string) => {
    return role === 'admin' 
      ? 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300' 
      : '';
  };

  return (
    <IconBadge
      icon={showIcon ? getRoleIcon(role, size) : <></>}
      label={role.charAt(0).toUpperCase() + role.slice(1)}
      variant={getRoleVariant(role)}
      size={size}
      className={`${getRoleClassName(role)} ${className}`}
    />
  );
}
