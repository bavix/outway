import { Card } from './Card.js';

interface AnimatedCardProps {
  children: any;
  className?: string;
  hover?: boolean;
  clickable?: boolean;
  onClick?: () => void;
  delay?: number;
}

export function AnimatedCard({
  children,
  className = '',
  hover = true,
  clickable = false,
  onClick,
  delay = 0
}: AnimatedCardProps) {
  const baseClasses = 'transition-all duration-300 ease-in-out';
  const hoverClasses = hover ? 'hover:shadow-lg hover:-translate-y-1' : '';
  const clickableClasses = clickable ? 'cursor-pointer active:scale-95' : '';
  const animationDelay = delay > 0 ? `animation-delay-${delay}` : '';

  return (
    <Card
      className={`${baseClasses} ${hoverClasses} ${clickableClasses} ${animationDelay} ${className}`}
      onClick={onClick}
    >
      {children}
    </Card>
  );
}
