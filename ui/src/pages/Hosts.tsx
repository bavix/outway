import { useEffect, useState } from 'preact/hooks';
import type { FailoverProvider } from '../providers/failoverProvider.js';
import type { HostOverride } from '../providers/types.js';
import { Card } from '../components/Card.js';
import { DataTable } from '../components/DataTable.js';
import { ActionButton } from '../components/ActionButton.js';
import { FormField } from '../components/FormField.js';
import { LoadingState } from '../components/LoadingState.js';

interface HostsProps { provider: FailoverProvider }

export function Hosts({ provider }: HostsProps) {
  const [hosts, setHosts] = useState<HostOverride[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let off = provider.onHosts((h) => { setHosts(h); setLoading(false); });
    provider.fetchHosts().then(setHosts).catch(e => setError(String(e))).finally(() => setLoading(false));
    return () => { off(); };
  }, [provider]);

  const addRow = () => setHosts(h => [...h, { pattern: '', a: [], aaaa: [], ttl: 60 }]);
  const removeRow = (idx: number) => setHosts(h => h.filter((_, i) => i !== idx));
  const update = (idx: number, patch: Partial<HostOverride>) => setHosts(h => h.map((row, i) => i === idx ? { ...row, ...patch } : row));
  const parseCSV = (v?: string) => v ? v.split(',').map(s => s.trim()).filter(Boolean) : [];

  const save = async () => {
    setError(null);
    try {
      await provider.saveHosts(hosts);
    } catch (e) {
      setError(String(e));
    }
  };

  const columns = [
    {
      key: 'pattern',
      title: 'Pattern',
      width: '28%',
      render: (value: string, _item: HostOverride, index: number) => (
        <FormField
          label=""
          value={value || ''}
          onChange={(val) => update(index, { pattern: val as string })}
          placeholder="e.g. *.example.com"
          className="mb-0"
        />
      )
    },
    {
      key: 'a',
      title: 'A Records',
      width: '28%',
      render: (value: string[], _item: HostOverride, index: number) => (
        <FormField
          label=""
          value={(value || []).join(', ')}
          onChange={(val) => update(index, { a: parseCSV(val as string) })}
          placeholder="127.0.0.1, 10.0.0.1"
          className="mb-0"
        />
      )
    },
    {
      key: 'aaaa',
      title: 'AAAA Records',
      width: '28%',
      render: (value: string[], _item: HostOverride, index: number) => (
        <FormField
          label=""
          value={(value || []).join(', ')}
          onChange={(val) => update(index, { aaaa: parseCSV(val as string) })}
          placeholder="::1, 2001:db8::1"
          className="mb-0"
        />
      )
    },
    {
      key: 'ttl',
      title: 'TTL',
      width: '8%',
      render: (value: number, _item: HostOverride, index: number) => (
        <FormField
          label=""
          type="number"
          value={value ?? 60}
          onChange={(val) => update(index, { ttl: val as number })}
          placeholder="60"
          className="mb-0"
        />
      )
    },
    {
      key: 'actions',
      title: 'Actions',
      width: '8%',
      render: (_value: any, _item: HostOverride, index: number) => (
        <div className="flex justify-end">
          <ActionButton
            action="delete"
            onClick={() => removeRow(index)}
            size="sm"
          />
        </div>
      )
    }
  ];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Hosts</h1>
        <div className="space-x-2">
          <ActionButton action="add" onClick={addRow} />
          <ActionButton action="save" onClick={save} />
        </div>
      </div>
      
      {error && (
        <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
          <div className="flex">
            <div className="flex-shrink-0">
              <svg className="h-5 w-5 text-red-400" viewBox="0 0 20 20" fill="currentColor">
                <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
              </svg>
            </div>
            <div className="ml-3">
              <h3 className="text-sm font-medium text-red-800 dark:text-red-200">
                Error
              </h3>
              <div className="mt-2 text-sm text-red-700 dark:text-red-300">
                {error}
              </div>
            </div>
          </div>
        </div>
      )}

      <Card title="Overrides" subtitle="Static DNS responses with wildcard support">
        {loading ? (
          <LoadingState message="Loading hosts..." />
        ) : (
          <DataTable
            data={hosts}
            columns={columns}
            emptyMessage="No hosts configured"
            emptyDescription="Click Add to create your first host override."
          />
        )}
      </Card>
    </div>
  );
}


