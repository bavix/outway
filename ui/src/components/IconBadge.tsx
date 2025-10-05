import { Badge } from './Badge.js';

interface IconBadgeProps {
  icon: any;
  label: string;
  variant?: 'primary' | 'secondary' | 'default';
  size?: 'sm' | 'md' | 'lg';
  className?: string;
}

export function IconBadge({ 
  icon, 
  label, 
  variant = 'secondary',
  size = 'md',
  className = ''
}: IconBadgeProps) {
  const sizeClasses = {
    sm: 'text-xs',
    md: 'text-sm',
    lg: 'text-base'
  };

  return (
    <Badge 
      variant={variant}
      className={`flex items-center space-x-1 ${sizeClasses[size]} ${className}`}
    >
      {icon}
      <span>{label}</span>
    </Badge>
  );
}
