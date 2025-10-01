import { useState, useEffect } from 'preact/hooks';
import { FailoverProvider } from '../providers/failoverProvider.js';

interface LocalDNSProps {
  provider: FailoverProvider;
}

interface Lease {
  expire: string;
  mac: string;
  ip: string;
  hostname: string;
  id: string;
}

interface Zone {
  name: string;
  detected: boolean;
}

interface ResolveResult {
  hostname: string;
  is_local: boolean;
  zone: string;
  rcode: number;
  answers: number;
  authoritative: boolean;
  answer_details: Array<{
    name: string;
    type: number;
    class: number;
    ttl: number;
    data: string;
  }>;
}

export function LocalDNS({ provider }: LocalDNSProps) {
  const [zones, setZones] = useState<Zone[]>([]);
  const [leases, setLeases] = useState<Lease[]>([]);
  const [testHostname, setTestHostname] = useState('');
  const [resolveResult, setResolveResult] = useState<ResolveResult | null>(null);
  const [loading, setLoading] = useState(true);
  const [resolving, setResolving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Load initial data and set up polling
  useEffect(() => {
    const loadData = async () => {
      try {
        setLoading(true);
        setError(null);

        // Load zones and leases using provider methods
        const [zonesData, leasesData] = await Promise.all([
          provider.fetchLocalZones?.() || Promise.resolve([]),
          provider.fetchLocalLeases?.() || Promise.resolve([])
        ]);

        setZones(zonesData.map((zone: string) => ({ name: zone, detected: true })));
        setLeases(leasesData);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load data');
      } finally {
        setLoading(false);
      }
    };

    // Load initial data
    loadData();

    // Set up polling every 30 seconds
    const interval = setInterval(loadData, 30000);

    return () => {
      clearInterval(interval);
    };
  }, [provider]);


  const testResolve = async () => {
    if (!testHostname.trim()) return;

    try {
      setResolving(true);
      setError(null);
      setResolveResult(null); // Clear previous result
      
      const result = await provider.testLocalResolve(testHostname);
      setResolveResult(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to resolve hostname');
    } finally {
      setResolving(false);
    }
  };

  const formatExpiry = (expireStr: string) => {
    try {
      const expire = new Date(expireStr);
      const now = new Date();
      const diff = expire.getTime() - now.getTime();
      
      if (diff < 0) {
        return 'Expired';
      }
      
      const hours = Math.floor(diff / (1000 * 60 * 60));
      const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));
      
      if (hours > 0) {
        return `${hours}h ${minutes}m`;
      } else {
        return `${minutes}m`;
      }
    } catch {
      return 'Unknown';
    }
  };

  const getRCodeName = (rcode: number) => {
    switch (rcode) {
      case 0: return 'NOERROR';
      case 1: return 'FORMERR';
      case 2: return 'SERVFAIL';
      case 3: return 'NXDOMAIN';
      case 4: return 'NOTIMP';
      case 5: return 'REFUSED';
      default: return `RCODE_${rcode}`;
    }
  };

  const getRecordTypeName = (type: number) => {
    switch (type) {
      case 1: return 'A';
      case 28: return 'AAAA';
      case 5: return 'CNAME';
      case 15: return 'MX';
      case 2: return 'NS';
      case 16: return 'TXT';
      case 33: return 'SRV';
      case 12: return 'PTR';
      default: return `TYPE_${type}`;
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
        <span className="ml-2 text-gray-600">Loading local DNS data...</span>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Local DNS</h1>
      </div>

      {error && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-4">
          <div className="flex">
            <div className="flex-shrink-0">
              <svg className="h-5 w-5 text-red-400" viewBox="0 0 20 20" fill="currentColor">
                <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
              </svg>
            </div>
            <div className="ml-3">
              <h3 className="text-sm font-medium text-red-800">Error</h3>
              <div className="mt-2 text-sm text-red-700">{error}</div>
            </div>
          </div>
        </div>
      )}

      {/* Detected Zones */}
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
        <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">Detected Local Zones</h2>
        <div className="flex flex-wrap gap-2">
          {zones.map((zone) => (
            <span
              key={zone.name}
              className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200"
            >
              {zone.name}
              {zone.detected && (
                <span className="ml-1 text-xs">(auto)</span>
              )}
            </span>
          ))}
          {zones.length === 0 && (
            <p className="text-gray-500 dark:text-gray-400">No local zones detected</p>
          )}
        </div>
      </div>

      {/* DHCP Leases */}
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
        <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
          DHCP Leases ({leases.length})
        </h2>
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
            <thead className="bg-gray-50 dark:bg-gray-700">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                  Hostname
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                  IP Address
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                  MAC Address
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                  Expires
                </th>
              </tr>
            </thead>
            <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
              {leases.map((lease, index) => (
                <tr key={index} className="hover:bg-gray-50 dark:hover:bg-gray-700">
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900 dark:text-white">
                    {lease.hostname}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-300">
                    {lease.ip}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-300 font-mono">
                    {lease.mac}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-300">
                    {formatExpiry(lease.expire)}
                  </td>
                </tr>
              ))}
              {leases.length === 0 && (
                <tr>
                  <td colSpan={4} className="px-6 py-4 text-center text-gray-500 dark:text-gray-400">
                    No DHCP leases found
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Test Resolution */}
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
        <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">Test Resolution</h2>
        <div className="flex gap-4 mb-4">
          <input
            type="text"
            value={testHostname}
            onChange={(e) => setTestHostname((e.target as HTMLInputElement).value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && testHostname.trim() && !resolving) {
                testResolve();
              }
            }}
            placeholder="Enter hostname (e.g., myhost.lan)"
            className="flex-1 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-white"
          />
          <button
            onClick={testResolve}
            disabled={!testHostname.trim() || resolving}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors flex items-center gap-2"
          >
            {resolving && (
              <svg className="animate-spin h-4 w-4" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
              </svg>
            )}
            {resolving ? 'Resolving...' : 'Resolve'}
          </button>
        </div>

        {resolveResult && (
          <div className="bg-gray-50 dark:bg-gray-700 rounded-lg p-4">
            <h3 className="font-medium text-gray-900 dark:text-white mb-2">Resolution Result</h3>
            <div className="space-y-2 text-sm">
              <div className="flex">
                <span className="font-medium text-gray-600 dark:text-gray-400 w-24">Hostname:</span>
                <span className="text-gray-900 dark:text-white">{resolveResult.hostname}</span>
              </div>
              <div className="flex">
                <span className="font-medium text-gray-600 dark:text-gray-400 w-24">Local Zone:</span>
                <span className={`px-2 py-1 rounded text-xs ${resolveResult.is_local ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'}`}>
                  {resolveResult.is_local ? 'Yes' : 'No'}
                </span>
              </div>
              {resolveResult.is_local && (
                <div className="flex">
                  <span className="font-medium text-gray-600 dark:text-gray-400 w-24">Zone:</span>
                  <span className="text-gray-900 dark:text-white">{resolveResult.zone}</span>
                </div>
              )}
              <div className="flex">
                <span className="font-medium text-gray-600 dark:text-gray-400 w-24">RCode:</span>
                <span className={`px-2 py-1 rounded text-xs ${
                  resolveResult.rcode === 0 ? 'bg-green-100 text-green-800' : 
                  resolveResult.rcode === 3 ? 'bg-yellow-100 text-yellow-800' : 
                  'bg-red-100 text-red-800'
                }`}>
                  {getRCodeName(resolveResult.rcode)}
                </span>
              </div>
              <div className="flex">
                <span className="font-medium text-gray-600 dark:text-gray-400 w-24">Answers:</span>
                <span className="text-gray-900 dark:text-white">{resolveResult.answers}</span>
              </div>
              <div className="flex">
                <span className="font-medium text-gray-600 dark:text-gray-400 w-24">Authoritative:</span>
                <span className="text-gray-900 dark:text-white">{resolveResult.authoritative ? 'Yes' : 'No'}</span>
              </div>

              {resolveResult.answers === 0 ? (
                <div className="mt-4 p-3 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg">
                  <div className="flex items-center gap-2">
                    <svg className="w-5 h-5 text-yellow-600 dark:text-yellow-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.732 16.5c-.77.833.192 2.5 1.732 2.5z" />
                    </svg>
                    <span className="text-yellow-800 dark:text-yellow-200 font-medium">No DNS records found</span>
                  </div>
                  <p className="text-yellow-700 dark:text-yellow-300 text-sm mt-1">
                    {resolveResult.rcode === 3 ? 'Hostname does not exist (NXDOMAIN)' : 'No answers returned'}
                  </p>
                </div>
              ) : resolveResult.answer_details.length > 0 ? (
                <div className="mt-4">
                  <h4 className="font-medium text-gray-900 dark:text-white mb-2">Answer Details</h4>
                  <div className="space-y-1">
                    {resolveResult.answer_details.map((answer, index) => (
                      <div key={index} className="flex items-center gap-2 text-xs">
                        <span className="font-mono bg-gray-200 dark:bg-gray-600 px-2 py-1 rounded">
                          {getRecordTypeName(answer.type)}
                        </span>
                        <span className="text-gray-900 dark:text-white">{answer.name}</span>
                        <span className="text-gray-500 dark:text-gray-400">→</span>
                        <span className="text-gray-900 dark:text-white">{answer.data}</span>
                        <span className="text-gray-500 dark:text-gray-400">TTL: {answer.ttl}</span>
                      </div>
                    ))}
                  </div>
                </div>
              ) : null}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
