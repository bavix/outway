import { Permission } from '../providers/types.js';
import { Card } from './Card.js';
import { Badge } from './Badge.js';

interface PermissionsCardProps {
  permissions: Permission[];
  title?: string;
  showCount?: boolean;
  groupByCategory?: boolean;
  className?: string;
}

export function PermissionsCard({ 
  permissions, 
  title = 'Permissions',
  showCount = true,
  groupByCategory = true,
  className = ''
}: PermissionsCardProps) {
  // Group permissions by category
  const categories = groupByCategory 
    ? permissions.reduce((acc, perm) => {
        const category = perm.category || 'Other';
        if (!acc[category]) {
          acc[category] = [];
        }
        acc[category].push(perm);
        return acc;
      }, {} as Record<string, Permission[]>)
    : { 'All': permissions };

  return (
    <Card className={className}>
      <div className="p-6">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
            {title}
          </h3>
          {showCount && (
            <Badge variant="secondary">
              {permissions.length} permission{permissions.length !== 1 ? 's' : ''}
            </Badge>
          )}
        </div>

        {Object.keys(categories).length > 0 ? (
          <div className="space-y-4">
            {Object.entries(categories).map(([category, perms]) => (
              <div key={category} className="border border-gray-200 dark:border-gray-700 rounded-lg shadow-sm hover:shadow-md transition-shadow">
                <div className="bg-gradient-to-r from-gray-50 to-gray-100 dark:from-gray-800 dark:to-gray-700 px-4 py-3 border-b border-gray-200 dark:border-gray-700">
                  <div className="flex items-center justify-between">
                    <h4 className="font-medium text-gray-900 dark:text-white">{category}</h4>
                    <Badge variant="secondary" className="text-xs">
                      {perms.length} permission{perms.length !== 1 ? 's' : ''}
                    </Badge>
                  </div>
                </div>
                <div className="p-4">
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                    {perms.map((perm) => (
                      <div key={perm.name} className="flex items-start space-x-3 p-3 bg-gray-50 dark:bg-gray-800 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors">
                        <div className="flex-shrink-0 mt-1">
                          <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                        </div>
                        <div className="flex-1 min-w-0">
                          <p className="text-sm font-medium text-gray-900 dark:text-white">
                            {perm.name}
                          </p>
                          <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                            {perm.description}
                          </p>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="text-center py-8">
            <div className="text-gray-400 dark:text-gray-500">
              <svg className="mx-auto h-12 w-12 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <p className="text-lg font-medium text-gray-900 dark:text-white">No permissions found</p>
              <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                This role doesn't have any permissions assigned.
              </p>
            </div>
          </div>
        )}
      </div>
    </Card>
  );
}
