import { Button } from './Button.js';

interface AccessibleButtonProps {
  children: any;
  onClick: () => void;
  variant?: 'primary' | 'secondary' | 'outline' | 'danger';
  size?: 'sm' | 'md' | 'lg';
  disabled?: boolean;
  loading?: boolean;
  className?: string;
  ariaLabel?: string;
  ariaDescribedBy?: string;
  tabIndex?: number;
}

export function AccessibleButton({
  children,
  onClick,
  variant = 'primary',
  size = 'md',
  disabled = false,
  loading = false,
  className = '',
  ariaLabel,
  ariaDescribedBy,
  tabIndex
}: AccessibleButtonProps) {
  return (
    <Button
      variant={variant}
      size={size}
      onClick={onClick}
      disabled={disabled || loading}
      className={className}
      aria-label={ariaLabel}
      aria-describedby={ariaDescribedBy}
      tabIndex={tabIndex}
      aria-disabled={disabled || loading}
    >
      {loading ? (
        <>
          <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-current mr-2" aria-hidden="true"></div>
          <span className="sr-only">Loading...</span>
        </>
      ) : (
        children
      )}
    </Button>
  );
}
