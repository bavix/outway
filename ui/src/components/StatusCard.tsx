import { Card } from './Card.js';
import { Badge } from './Badge.js';

interface StatusCardProps {
  title: string;
  description?: string;
  count?: number;
  icon?: any;
  status?: 'active' | 'inactive' | 'warning' | 'error';
  onClick?: () => void;
  className?: string;
}

export function StatusCard({ 
  title, 
  description, 
  count,
  icon,
  status = 'active',
  onClick,
  className = ''
}: StatusCardProps) {
  const getStatusClasses = (status: 'active' | 'inactive' | 'warning' | 'error') => {
    const statuses = {
      active: 'hover:shadow-lg ring-2 ring-blue-500 bg-blue-50 dark:bg-blue-900/20',
      inactive: 'hover:bg-gray-50 dark:hover:bg-gray-800',
      warning: 'hover:shadow-lg ring-2 ring-yellow-500 bg-yellow-50 dark:bg-yellow-900/20',
      error: 'hover:shadow-lg ring-2 ring-red-500 bg-red-50 dark:bg-red-900/20'
    };
    return statuses[status];
  };

  const statusClasses = getStatusClasses(status);
  const isClickable = !!onClick;

  return (
    <Card 
      className={`transition-all duration-200 ${isClickable ? 'cursor-pointer' : ''} ${statusClasses} ${className}`}
      onClick={onClick}
    >
      <div className="p-6">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center space-x-3">
            {icon && (
              <div className="p-2 rounded-lg bg-gray-100 dark:bg-gray-800">
                {icon}
              </div>
            )}
            <div>
              <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
                {title}
              </h3>
              {description && (
                <p className="text-sm text-gray-600 dark:text-gray-400">
                  {description}
                </p>
              )}
            </div>
          </div>
          {count !== undefined && (
            <Badge variant="secondary" className="text-xs">
              {count}
            </Badge>
          )}
        </div>
      </div>
    </Card>
  );
}
