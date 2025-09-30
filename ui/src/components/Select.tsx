import type { ComponentChildren } from 'preact';

type SelectProps = {
  label?: string;
  value: string;
  onChange: (e: Event) => void;
  children: ComponentChildren;
  className?: string;
  name?: string;
  disabled?: boolean;
  required?: boolean;
  id?: string;
} & Record<string, any>;

export function Select({ label, value, onChange, children, className = '', ...props }: SelectProps) {
  return (
    <div>
      {label && (
        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{label}</label>
      )}
      <select
        value={value}
        onChange={onChange as any}
        className={`w-full px-3 py-2 h-[42px] border border-gray-300 rounded-md bg-white text-gray-900 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-gray-800 dark:text-gray-100 dark:border-gray-600 dark:focus:border-blue-400 dark:focus:ring-blue-400 ${className}`}
        {...props}
      >
        {children}
      </select>
    </div>
  );
}


