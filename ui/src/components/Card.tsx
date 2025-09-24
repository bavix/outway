import type { JSX } from 'preact';

interface CardProps {
  /** Card content */
  children: JSX.Element | JSX.Element[];
  /** Additional CSS classes */
  className?: string;
  /** Card title */
  title?: string | undefined;
  /** Card subtitle */
  subtitle?: string | undefined;
}

export function Card({ 
  children, 
  className = '', 
  title, 
  subtitle,
  ...props
}: CardProps & JSX.HTMLAttributes<HTMLDivElement>) {
  
  const cardClasses = `card bg-white/80 dark:bg-gray-800/70 backdrop-blur-sm rounded-xl p-6 border border-gray-200/50 dark:border-gray-700/50 shadow-lg hover:shadow-xl transition-all duration-300 ${className}`.trim();

  return (
    <div className={cardClasses} {...props}>
      {(title || subtitle) && (
        <div className="card-header pb-4 mb-6 border-b border-gray-200 dark:border-gray-700">
          {title && <h3 className="card-title text-xl font-semibold text-gray-900 dark:text-gray-100 leading-tight">{title}</h3>}
          {subtitle && <p className="card-subtitle text-sm text-gray-600 dark:text-gray-400 mt-2 leading-relaxed">{subtitle}</p>}
        </div>
      )}
      
      <div className="card-content text-gray-700 dark:text-gray-300 leading-relaxed">
        {children}
      </div>
    </div>
  );
}