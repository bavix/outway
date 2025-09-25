import { useEffect, useState } from 'preact/hooks';
import type { FailoverProvider } from '../providers/failoverProvider.js';
import type { HostOverride } from '../providers/types.js';
import { Button } from '../components/Button.js';
import { Card } from '../components/Card.js';
import { Input } from '../components/Input.js';

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

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Hosts</h1>
        <div className="space-x-2">
          <Button variant="secondary" onClick={addRow}>Add</Button>
          <Button onClick={save}>Save</Button>
        </div>
      </div>
      {error && <div className="text-red-600 text-sm">{error}</div>}
      <Card title="Overrides" subtitle="Static DNS responses with wildcard support">
        {loading ? (
          <div className="text-gray-500">Loading...</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
              <thead className="bg-gray-50 dark:bg-gray-800">
                <tr>
                  <th className="px-3 py-2 text-left text-xs font-medium text-gray-600 dark:text-gray-300 uppercase tracking-wider">Pattern</th>
                  <th className="px-3 py-2 text-left text-xs font-medium text-gray-600 dark:text-gray-300 uppercase tracking-wider">A (comma separated)</th>
                  <th className="px-3 py-2 text-left text-xs font-medium text-gray-600 dark:text-gray-300 uppercase tracking-wider">AAAA (comma separated)</th>
                  <th className="px-3 py-2 text-left text-xs font-medium text-gray-600 dark:text-gray-300 uppercase tracking-wider">TTL</th>
                  <th className="px-3 py-2 text-right text-xs font-medium text-gray-600 dark:text-gray-300 uppercase tracking-wider">Actions</th>
                </tr>
              </thead>
              <tbody className="bg-white dark:bg-gray-900 divide-y divide-gray-200 dark:divide-gray-700">
                {hosts.map((h, idx) => (
                  <tr key={idx}>
                    <td className="px-3 py-2 w-[28%]">
                      <Input value={h.pattern || ''} onInput={(e: any) => update(idx, { pattern: e.currentTarget.value })} placeholder="e.g. *.example.com" />
                    </td>
                    <td className="px-3 py-2 w-[28%]">
                      <Input value={(h.a || []).join(', ')} onInput={(e: any) => update(idx, { a: parseCSV(e.currentTarget.value) })} placeholder="127.0.0.1, 10.0.0.1" />
                    </td>
                    <td className="px-3 py-2 w-[28%]">
                      <Input value={(h.aaaa || []).join(', ')} onInput={(e: any) => update(idx, { aaaa: parseCSV(e.currentTarget.value) })} placeholder="::1, 2001:db8::1" />
                    </td>
                    <td className="px-3 py-2 w-[8%]">
                      <Input type="number" value={String(h.ttl ?? 60)} onInput={(e: any) => update(idx, { ttl: Number(e.currentTarget.value) })} placeholder="60" />
                    </td>
                    <td className="px-3 py-2 text-right w-[8%]">
                      <Button variant="danger" onClick={() => removeRow(idx)}>Del</Button>
                    </td>
                  </tr>
                ))}
                {hosts.length === 0 && (
                  <tr>
                    <td colSpan={5} className="px-3 py-6 text-sm text-gray-500 text-center">No hosts configured. Click Add to create one.</td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        )}
      </Card>
    </div>
  );
}


