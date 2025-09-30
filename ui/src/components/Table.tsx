import type { ComponentChildren } from 'preact';

export function Table({ children, className = '' }: { children: ComponentChildren; className?: string }) {
  return (
    <div className={`overflow-auto border border-gray-200 dark:border-gray-700 rounded-lg ${className}`}>
      <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">{children}</table>
    </div>
  );
}

export function THead({ children }: { children: ComponentChildren }) {
  return <thead className="bg-gray-50 dark:bg-gray-900">{children}</thead>;
}

export function TRow({ children }: { children: ComponentChildren }) {
  return <tr>{children}</tr>;
}

export function TH({ children, sortable = false, active = false, order = 'asc', onClick }: { children?: ComponentChildren; sortable?: boolean; active?: boolean; order?: 'asc' | 'desc'; onClick?: () => void }) {
  return (
    <th
      onClick={onClick}
      className={`px-4 py-2 text-left text-xs font-medium uppercase tracking-wider ${sortable ? 'cursor-pointer select-none' : ''} text-gray-500 dark:text-gray-400`}
    >
      <span className="inline-flex items-center gap-1">
        {children}
        {sortable && active && (
          <svg className="w-3 h-3" viewBox="0 0 20 20" fill="currentColor"><path d={order==='asc'? 'M5 12l5-5 5 5H5z' : 'M5 8l5 5 5-5H5z'} /></svg>
        )}
      </span>
    </th>
  );
}

export function TBody({ children }: { children: ComponentChildren }) {
  return <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">{children}</tbody>;
}

export function TD({ children, align = 'left', colSpan, title }: { children: ComponentChildren; align?: 'left' | 'right'; colSpan?: number; title?: string }) {
  return <td className={`px-4 py-2 text-sm ${align === 'right' ? 'text-right' : ''}`} colSpan={colSpan} title={title}>{children}</td>;
}

// Generic data table intentionally omitted to avoid name conflicts; use lightweight primitives above.
