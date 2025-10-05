import { Input } from './Input.js';
import { Select } from './Select.js';

interface FormFieldProps {
  label: string;
  type?: 'text' | 'email' | 'password' | 'number' | 'tel' | 'url';
  value: string | number;
  onChange: (value: string | number) => void;
  placeholder?: string;
  required?: boolean;
  error?: string;
  helpText?: string;
  className?: string;
  disabled?: boolean;
}

interface SelectFieldProps {
  label: string;
  value: string;
  onChange: (value: string) => void;
  options: Array<{ value: string; label: string }>;
  placeholder?: string;
  required?: boolean;
  error?: string;
  helpText?: string;
  className?: string;
  disabled?: boolean;
}

export function FormField({
  label,
  type = 'text',
  value,
  onChange,
  placeholder,
  required = false,
  error,
  helpText,
  className = '',
  disabled = false
}: FormFieldProps) {
  return (
    <div className={`space-y-2 ${className}`}>
      <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
        {label}
        {required && <span className="text-red-500 ml-1">*</span>}
      </label>
      <Input
        type={type}
        value={String(value)}
        onInput={(e: any) => onChange(type === 'number' ? Number(e.currentTarget.value) : e.currentTarget.value)}
        placeholder={placeholder}
        disabled={disabled}
        className={error ? 'border-red-500 focus:border-red-500 focus:ring-red-500' : ''}
      />
      {error && (
        <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
      )}
      {helpText && !error && (
        <p className="text-sm text-gray-500 dark:text-gray-400">{helpText}</p>
      )}
    </div>
  );
}

export function SelectField({
  label,
  value,
  onChange,
  options,
  placeholder,
  required = false,
  error,
  helpText,
  className = '',
  disabled = false
}: SelectFieldProps) {
  return (
    <div className={`space-y-2 ${className}`}>
      <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
        {label}
        {required && <span className="text-red-500 ml-1">*</span>}
      </label>
      <Select
        value={value}
        onChange={(e: any) => onChange(e.currentTarget.value)}
        disabled={disabled}
        className={error ? 'border-red-500 focus:border-red-500 focus:ring-red-500' : ''}
      >
        {placeholder && <option value="">{placeholder}</option>}
        {options.map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </Select>
      {error && (
        <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
      )}
      {helpText && !error && (
        <p className="text-sm text-gray-500 dark:text-gray-400">{helpText}</p>
      )}
    </div>
  );
}
