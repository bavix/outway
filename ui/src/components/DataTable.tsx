import { Table, THead, TBody, TRow, TH, TD } from './Table.js';
import { LoadingState } from './LoadingState.js';
import { EmptyState } from './EmptyState.js';

interface Column<T> {
  key: keyof T | string;
  title: string;
  render?: (value: any, item: T, index: number) => any;
  width?: string;
  className?: string;
}

interface DataTableProps<T> {
  data: T[];
  columns: Column<T>[];
  loading?: boolean;
  emptyMessage?: string;
  emptyDescription?: string;
  className?: string;
}

export function DataTable<T>({
  data,
  columns,
  loading = false,
  emptyMessage = 'No data available',
  emptyDescription = 'There are no items to display.',
  className = ''
}: DataTableProps<T>) {
  if (loading) {
    return <LoadingState message="Loading data..." />;
  }

  if (data.length === 0) {
    return (
      <EmptyState
        title={emptyMessage}
        description={emptyDescription}
        icon="box"
      />
    );
  }

  const getValue = (item: T, key: keyof T | string) => {
    if (typeof key === 'string' && key.includes('.')) {
      return key.split('.').reduce((obj: any, k) => obj?.[k], item);
    }
    return item[key as keyof T];
  };

  return (
    <div className={`overflow-x-auto ${className}`}>
      <Table>
        <THead>
          <TRow>
            {columns.map((column) => (
              <TH
                key={String(column.key)}
              >
                {column.title}
              </TH>
            ))}
          </TRow>
        </THead>
        <TBody>
          {data.map((item, index) => (
            <TRow
              key={index}
            >
              {columns.map((column) => (
                <TD
                  key={String(column.key)}
                >
                  {column.render
                    ? column.render(getValue(item, column.key), item, index)
                    : getValue(item, column.key)
                  }
                </TD>
              ))}
            </TRow>
          ))}
        </TBody>
      </Table>
    </div>
  );
}
