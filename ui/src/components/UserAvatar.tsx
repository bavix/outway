import { GravatarAvatar } from './GravatarAvatar';

interface UserAvatarProps {
  email: string;
  role?: string;
  size?: 'sm' | 'md' | 'lg' | 'xl';
  showRole?: boolean;
  className?: string;
}

export function UserAvatar({
  email,
  role,
  size = 'md',
  showRole = false,
  className = ''
}: UserAvatarProps) {
  const getRoleVariant = (role?: string) => {
    if (role === 'admin') {
      return 'danger';
    }
    return 'primary';
  };

  return (
    <div className={`flex items-center space-x-2 ${className}`}>
      <GravatarAvatar
        email={email}
        size={size}
        variant={getRoleVariant(role)}
      />
      {showRole && role && (
        <div className="text-sm text-gray-600 dark:text-gray-400">
          {role}
        </div>
      )}
    </div>
  );
}
