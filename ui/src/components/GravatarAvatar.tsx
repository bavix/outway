import { useState } from 'preact/hooks';
import { Avatar } from './Avatar';

interface GravatarAvatarProps {
  email: string;
  size?: 'sm' | 'md' | 'lg' | 'xl';
  variant?: 'primary' | 'secondary' | 'danger';
  className?: string;
  fallbackToInitials?: boolean;
}

export function GravatarAvatar({
  email,
  size = 'md',
  variant = 'primary',
  className = '',
  fallbackToInitials = true
}: GravatarAvatarProps) {
  const [imageError, setImageError] = useState(false);
  const [imageLoading, setImageLoading] = useState(true);

  // Generate MD5 hash for Gravatar (simplified version)
  const generateMD5 = (str: string): string => {
    // This is a simplified MD5 implementation for demo purposes
    // In production, you should use a proper MD5 library
    let hash = 0;
    for (let i = 0; i < str.length; i++) {
      const char = str.charCodeAt(i);
      hash = ((hash << 5) - hash) + char;
      hash = hash & hash; // Convert to 32-bit integer
    }
    return Math.abs(hash).toString(16);
  };

  const getGravatarUrl = (email: string, size: number): string => {
    const hash = generateMD5(email.toLowerCase().trim());
    return `https://www.gravatar.com/avatar/${hash}?s=${size}&d=404&r=g`;
  };

  const getSizeInPixels = (size: 'sm' | 'md' | 'lg' | 'xl'): number => {
    const sizes = {
      sm: 32,
      md: 40,
      lg: 48,
      xl: 56
    };
    return sizes[size];
  };

  const getInitials = (email: string): string => {
    const name = email.split('@')[0];
    return name ? name.charAt(0).toUpperCase() : '?';
  };

  const sizeInPixels = getSizeInPixels(size);
  const gravatarUrl = getGravatarUrl(email, sizeInPixels);

  if (imageError || !fallbackToInitials) {
    return (
      <Avatar
        initials={getInitials(email)}
        size={size}
        variant={variant}
        className={className}
      />
    );
  }

  return (
    <div className={`relative ${className}`}>
      {imageLoading && (
        <Avatar
          initials={getInitials(email)}
          size={size}
          variant={variant}
          className="absolute inset-0"
        />
      )}
      <img
        src={gravatarUrl}
        alt={`Avatar for ${email}`}
        className={`rounded-full ${getSizeClasses(size)} object-cover`}
        onLoad={() => setImageLoading(false)}
        onError={() => setImageError(true)}
        style={{ display: imageLoading ? 'none' : 'block' }}
      />
    </div>
  );
}

function getSizeClasses(size: 'sm' | 'md' | 'lg' | 'xl'): string {
  const sizeClasses = {
    sm: 'w-8 h-8',
    md: 'w-10 h-10',
    lg: 'w-12 h-12',
    xl: 'w-14 h-14',
  };
  return sizeClasses[size];
}
