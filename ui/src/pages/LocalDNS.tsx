import { useEffect, useState } from 'preact/hooks';
import type { FailoverProvider } from '../providers/failoverProvider.js';
import { Button } from '../components/Button.js';
import { Card } from '../components/Card.js';
import { Input } from '../components/Input.js';

interface LocalDNSProps { provider: FailoverProvider }

interface Lease {
  hostname: string;
  ip: string;
  mac: string;
  expires_at: string;
  id?: string;
}

interface LocalZonesData {
  zones: string[];
  leases: Lease[];
}

export function LocalDNS({ provider }: LocalDNSProps) {
  const [zones, setZones] = useState<string[]>([]);
  const [leases, setLeases] = useState<Lease[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [queryName, setQueryName] = useState('');
  const [queryResult, setQueryResult] = useState<any>(null);
  const [queryLoading, setQueryLoading] = useState(false);

  useEffect(() => {
    // Subscribe to local zones updates
    const offZones = provider.onLocalZones?.((data: LocalZonesData) => {
      setZones(data.zones || []);
      setLeases(data.leases || []);
      setLoading(false);
    });

    // Initial fetch
    Promise.all([
      provider.fetchLocalZones?.().then(data => setZones(data.zones || [])),
      provider.fetchLocalLeases?.().then(data => setLeases(data.leases || []))
    ]).catch(e => setError(String(e))).finally(() => setLoading(false));

    return () => { offZones?.(); };
  }, [provider]);

  const handleResolve = async () => {
    if (!queryName.trim()) return;
    
    setQueryLoading(true);
    setQueryResult(null);
    setError(null);
    
    try {
      const result = await provider.resolveLocal?.(queryName);
      setQueryResult(result);
    } catch (e) {
      setError(String(e));
    } finally {
      setQueryLoading(false);
    }
  };

  const formatExpiry = (expiresAt: string) => {
    try {
      const date = new Date(expiresAt);
      const now = new Date();
      const diff = date.getTime() - now.getTime();
      
      if (diff < 0) return 'Expired';
      
      const hours = Math.floor(diff / (1000 * 60 * 60));
      const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));
      
      if (hours > 24) {
        const days = Math.floor(hours / 24);
        return `${days}d ${hours % 24}h`;
      }
      return `${hours}h ${minutes}m`;
    } catch {
      return expiresAt;
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Local DNS</h1>
          <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
            LAN domain resolution from DHCP leases
          </p>
        </div>
      </div>

      {error && (
        <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 text-red-800 dark:text-red-200 px-4 py-3 rounded">
          {error}
        </div>
      )}

      {/* Local Zones */}
      <Card title="Local Zones" subtitle="Detected DNS zones for LAN resolution">
        {loading ? (
          <div className="text-gray-500 dark:text-gray-400">Loading...</div>
        ) : zones.length === 0 ? (
          <div className="text-gray-500 dark:text-gray-400">
            No local zones configured. Enable local_zones in config.yaml
          </div>
        ) : (
          <div className="flex flex-wrap gap-2">
            {zones.map(zone => (
              <span key={zone} className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300">
                .{zone}
              </span>
            ))}
          </div>
        )}
      </Card>

      {/* Query Tester */}
      <Card title="Test Resolution" subtitle="Query local DNS resolution">
        <div className="space-y-4">
          <div className="flex gap-2">
            <Input
              value={queryName}
              onChange={(e) => setQueryName(e.currentTarget.value)}
              placeholder="hostname.lan"
              onKeyDown={(e) => { if (e.key === 'Enter') handleResolve(); }}
            />
            <Button onClick={handleResolve} disabled={queryLoading || !queryName.trim()}>
              {queryLoading ? 'Resolving...' : 'Resolve'}
            </Button>
          </div>

          {queryResult && (
            <div className="bg-gray-50 dark:bg-gray-800 rounded p-4 space-y-2 text-sm font-mono">
              <div><span className="text-gray-600 dark:text-gray-400">Name:</span> <span className="text-gray-900 dark:text-white">{queryResult.name}</span></div>
              <div><span className="text-gray-600 dark:text-gray-400">Status:</span> <span className={queryResult.status === 'OK' ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'}>{queryResult.status}</span></div>
              <div><span className="text-gray-600 dark:text-gray-400">Source:</span> <span className="text-gray-900 dark:text-white">{queryResult.source}</span></div>
              {queryResult.records && queryResult.records.length > 0 && (
                <div>
                  <span className="text-gray-600 dark:text-gray-400">Records:</span>
                  <div className="mt-1 space-y-1">
                    {queryResult.records.map((rec: string, idx: number) => (
                      <div key={idx} className="text-gray-900 dark:text-white pl-4">{rec}</div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
      </Card>

      {/* DHCP Leases */}
      <Card title="DHCP Leases" subtitle={`${leases.length} active lease${leases.length !== 1 ? 's' : ''}`}>
        {loading ? (
          <div className="text-gray-500 dark:text-gray-400">Loading...</div>
        ) : leases.length === 0 ? (
          <div className="text-gray-500 dark:text-gray-400">
            No DHCP leases found. Check /tmp/dhcp.leases on your OpenWrt device.
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
              <thead className="bg-gray-50 dark:bg-gray-800">
                <tr>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Hostname</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">IP Address</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">MAC Address</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Expires</th>
                </tr>
              </thead>
              <tbody className="bg-white dark:bg-gray-900 divide-y divide-gray-200 dark:divide-gray-700">
                {leases.map((lease, idx) => (
                  <tr key={idx} className="hover:bg-gray-50 dark:hover:bg-gray-800">
                    <td className="px-4 py-3 whitespace-nowrap text-sm font-medium text-gray-900 dark:text-white">
                      {lease.hostname}
                      {zones.map(zone => (
                        <span key={zone} className="ml-2 text-gray-500 dark:text-gray-400">.{zone}</span>
                      ))[0]}
                    </td>
                    <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-700 dark:text-gray-300 font-mono">{lease.ip}</td>
                    <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-700 dark:text-gray-300 font-mono">{lease.mac}</td>
                    <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-700 dark:text-gray-300">
                      {formatExpiry(lease.expires_at)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Card>
    </div>
  );
}
