import { Permission } from '../providers/types.js';
import { Button } from './Button.js';
import { RoleBadge } from './RoleBadge.js';

interface PermissionsDisplayProps {
  role: string;
  permissions?: Permission[];
  showDetails?: boolean;
  onViewPermissions?: () => void;
  className?: string;
}

export function PermissionsDisplay({ 
  role, 
  permissions = [], 
  showDetails = false, 
  onViewPermissions,
  className = '' 
}: PermissionsDisplayProps) {
  return (
    <div className={`space-y-2 ${className}`}>
      {/* Role Badge */}
      <div className="flex items-center space-x-2">
        <RoleBadge role={role} size="sm" />
        {permissions.length > 0 && (
          <span className="text-xs text-gray-500 dark:text-gray-400">
            {permissions.length} permissions
          </span>
        )}
      </div>

      {/* View Permissions Button */}
      {showDetails && onViewPermissions && (
        <Button
          variant="outline"
          size="sm"
          onClick={onViewPermissions}
          className="w-full"
        >
          View Permissions ({permissions.length})
        </Button>
      )}
    </div>
  );
}