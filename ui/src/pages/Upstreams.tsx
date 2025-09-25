import { useState, useEffect } from 'preact/hooks';
import { Card, Button, Input, Badge } from '../components/index.js';
import { useUpstreams, useUpstreamsActions } from '../store/store.js';
import type { UpstreamItem } from '../providers/types.js';
import { FailoverProvider } from '../providers/failoverProvider.js';
import { validateUpstream } from '../utils/format.js';

interface UpstreamsProps {
  provider: FailoverProvider;
}

export function Upstreams({ provider }: UpstreamsProps) {
  const upstreams = useUpstreams();
  const { setUpstreams, addUpstream, removeUpstream, setError } = useUpstreamsActions();
  
  const [newAddress, setNewAddress] = useState('');
  const [newName, setNewName] = useState('');
  const [newWeight, setNewWeight] = useState<number | ''>('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null);
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('asc');
  const [editing, setEditing] = useState<Record<string, Partial<UpstreamItem>>>({});

  // Subscribe to upstreams updates
  useEffect(() => {
    const unsubscribe = provider.onUpstreams((newUpstreams) => {
      setUpstreams(newUpstreams);
    });

    return unsubscribe;
  }, [provider, setUpstreams, setError]);

  // Normalize and sort upstreams for display
  const sortedUpstreams = [...(upstreams || [])]
    .sort((a, b) => {
      const comparison = (a.name || a.address).localeCompare(b.name || b.address);
      return sortOrder === 'asc' ? comparison : -comparison;
    });

  const handleSubmit = async (e: Event) => {
    e.preventDefault();
    
    const upstreamValidation = validateUpstream(newAddress);
    if (!upstreamValidation.valid) {
      setError(upstreamValidation.error!);
      return;
    }

    setIsSubmitting(true);
    setError(null);

    try {
      // Ensure URL form for address and non-empty name
      let address = newAddress.trim();
      try {
        const u = new URL(address);
        const scheme = u.protocol.replace(':', '').toLowerCase();
        if (scheme !== 'https') {
          const canonical = scheme === 'tls' ? 'dot' : scheme === 'quic' ? 'doq' : scheme;
          address = `${canonical}://${u.host}`;
        }
      } catch {
        // If no scheme, assume udp
        address = `udp://${address.replace(/^(udp|tcp|tls|dot|quic|doq):\/\//i, '')}`;
      }
      const item: UpstreamItem = {
        name: (newName || new URL(address).hostname).trim(),
        address,
      } as UpstreamItem;
      if (newWeight !== '') {
        (item as any).weight = Number(newWeight);
      }
      addUpstream(item);
      const updatedUpstreams = [...upstreams, item];
      await provider.saveUpstreams(updatedUpstreams);
      setNewAddress('');
      setNewName('');
      setNewWeight('');
    } catch (error) {
      console.error('Failed to save upstream:', error);
      setError(error instanceof Error ? error.message : 'Failed to save upstream');
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleDelete = async (addressToDelete: string) => {
    try {
      // Optimistic update
      removeUpstream(addressToDelete);
      
      // Send to server
      const updatedUpstreams = upstreams.filter(u => u.address !== addressToDelete);
      await provider.saveUpstreams(updatedUpstreams);
      
      setDeleteConfirm(null);
    } catch (error) {
      console.error('Failed to delete upstream:', error);
      setError(error instanceof Error ? error.message : 'Failed to delete upstream');
    }
  };

  const startEdit = (address: string) => {
    const item = upstreams.find(u => u.address === address);
    if (item) setEditing(prev => ({ ...prev, [address]: { ...item } }));
  };

  const cancelEdit = (address: string) => {
    setEditing(prev => { const n = { ...prev }; delete n[address]; return n; });
  };

  const applyEdit = async (address: string) => {
    const patch = editing[address];
    if (!patch) return;
    const current = upstreams.find(u => u.address === address);
    if (!current) return;
    const updatedItem: UpstreamItem = {
      name: (patch.name ?? current.name ?? '').trim(),
      address: (patch.address ?? current.address).trim(),
    } as UpstreamItem;
    const nextWeight = (patch.weight === undefined ? current.weight : Number(patch.weight)) as number | undefined;
    if (nextWeight !== undefined) {
      (updatedItem as any).weight = nextWeight;
    }
    const updatedList = upstreams.map(u => u.address === address ? updatedItem : u);
    try {
      setUpstreams(updatedList);
      await provider.saveUpstreams(updatedList);
      cancelEdit(address);
    } catch (err) {
      console.error('Failed to save edit:', err);
      setError(err instanceof Error ? err.message : 'Failed to save');
    }
  };

  const updateEditField = (address: string, field: keyof UpstreamItem, value: string) => {
    setEditing(prev => ({
      ...prev,
      [address]: { ...prev[address], [field]: field === 'weight' ? (value === '' ? undefined : Number(value)) : value }
    }));
  };

  const toggleSort = () => {
    setSortOrder(prev => prev === 'asc' ? 'desc' : 'asc');
  };

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Upstreams</h1>
          <p className="text-gray-600 dark:text-gray-400 mt-1">
            Manage DNS upstream servers and their configuration
          </p>
        </div>
      </div>

      {/* Add Upstream Form */}
      <Card title="Add New Upstream" subtitle="Add a new DNS upstream server">
        <form onSubmit={handleSubmit} className="space-y-6">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <Input label="Name" value={newName} onInput={(e) => setNewName((e.target as HTMLInputElement).value)} placeholder="Human-friendly name" />
            <Input label="Address" value={newAddress} onInput={(e) => setNewAddress((e.target as HTMLInputElement).value)} placeholder="udp://8.8.8.8:53 or https://dns.google/dns-query" hint="Use URL form" required />
            <Input label="Weight" type="number" value={newWeight as any} onInput={(e) => setNewWeight((e.target as HTMLInputElement).value === '' ? '' : Number((e.target as HTMLInputElement).value))} placeholder="1" />
          </div>
          
          <div className="flex gap-3 pt-4 border-t border-gray-200 dark:border-gray-700">
            <Button
              type="submit"
              variant="primary"
              loading={isSubmitting}
              disabled={isSubmitting}
            >
              {isSubmitting ? 'Adding...' : 'Add Upstream'}
            </Button>
          </div>
        </form>
      </Card>

      {/* Upstreams List */}
      <Card title="Configured Upstreams" subtitle={`${upstreams.length} upstream server${upstreams.length === 1 ? '' : 's'} configured`}>
        <div className="flex justify-between items-center mb-6">
          <div className="flex items-center gap-3">
            <Badge variant="primary">{upstreams.length}</Badge>
            <span className="text-sm text-gray-600 dark:text-gray-400">
              {upstreams.length === 1 ? 'upstream server' : 'upstream servers'}
            </span>
          </div>
          <Button
            variant="secondary"
            size="sm"
            onClick={toggleSort}
            title={`Sort ${sortOrder === 'asc' ? 'Z-A' : 'A-Z'}`}
          >
            {sortOrder === 'asc' ? 'A-Z' : 'Z-A'}
          </Button>
        </div>

        {sortedUpstreams.length === 0 ? (
          <div className="text-center py-8">
            <p className="text-gray-600 dark:text-gray-400">
              No upstream servers configured yet.
            </p>
          </div>
        ) : (
          <div className="space-y-3">
            {sortedUpstreams.map((item) => (
              <div key={item.address} className="p-4 bg-gray-50 dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
                {editing[item.address] ? (
                  <div className="grid grid-cols-1 md:grid-cols-4 gap-3 items-end">
                    <Input label="Name" value={(editing[item.address]?.name ?? item.name) as any} onInput={(e) => updateEditField(item.address, 'name', (e.target as HTMLInputElement).value)} />
                    <Input label="Address" value={(editing[item.address]?.address ?? item.address) as any} onInput={(e) => updateEditField(item.address, 'address', (e.target as HTMLInputElement).value)} />
                    <Input label="Weight" type="number" value={(editing[item.address]?.weight ?? item.weight ?? '') as any} onInput={(e) => updateEditField(item.address, 'weight', (e.target as HTMLInputElement).value)} />
                    <div className="flex gap-2 justify-end">
                      <Button variant="outline" size="sm" onClick={() => cancelEdit(item.address)}>Cancel</Button>
                      <Button variant="primary" size="sm" onClick={() => applyEdit(item.address)}>Save</Button>
                    </div>
                  </div>
                ) : (
                  <div className="flex items-center justify-between gap-4">
                    <div className="flex flex-col min-w-0">
                      <span className="text-sm text-gray-500 dark:text-gray-400 truncate max-w-[420px]" title={item.name || 'Unnamed'}>{item.name || 'Unnamed'}</span>
                      <span className="mt-1 px-2 py-1 rounded bg-gray-200 dark:bg-gray-700 text-gray-900 dark:text-gray-100 text-xs font-mono truncate max-w-[520px]" title={item.address}>{item.address}</span>
                    </div>
                    <div className="flex items-center gap-3">
                      <span className="text-xs text-gray-500 dark:text-gray-400">weight: {item.weight ?? 1}</span>
                      <Button variant="secondary" size="sm" onClick={() => startEdit(item.address)}>Edit</Button>
                      <Button variant="outline" size="sm" onClick={() => setDeleteConfirm(item.address)}>Delete</Button>
                    </div>
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </Card>

      {/* Delete Confirmation Modal */}
      {deleteConfirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black bg-opacity-50">
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl border border-gray-200 dark:border-gray-700 p-6 max-w-md w-full">
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
              Delete Upstream
            </h3>
            <p className="text-gray-600 dark:text-gray-400 mb-6">
              Are you sure you want to delete upstream <strong>"{deleteConfirm}"</strong>? This action cannot be undone.
            </p>
            <div className="flex gap-3 justify-end">
              <Button
                variant="outline"
                onClick={() => setDeleteConfirm(null)}
              >
                Cancel
              </Button>
              <Button
                variant="primary"
                onClick={() => handleDelete(deleteConfirm)}
              >
                Delete
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
