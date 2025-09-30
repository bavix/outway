import { useEffect, useMemo, useState } from 'preact/hooks';
import { Card } from '../components/Card.js';
import { Button } from '../components/Button.js';
import { Input } from '../components/Input.js';
import { FailoverProvider } from '../providers/failoverProvider.js';
import { extractDomain } from '../utils/extractDomain.js';
import { Modal } from '../components/Modal.js';
import { Select } from '../components/Select.js';
import { Table, THead, TBody, TRow, TH, TD } from '../components/Table.js';
import { CacheEntry, CacheListResponse, CacheKeyDetails } from '../providers/types.js';

interface CacheProps { provider: FailoverProvider }

export default function CachePage({ provider }: CacheProps) {
  const [name, setName] = useState('');
  const [qtype, setQtype] = useState<string>('');
  const [busy, setBusy] = useState(false);
  // extractDomain imported from utils

  // listing state
  const [rows, setRows] = useState<CacheEntry[]>([]);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [limit, setLimit] = useState(20);
  const [query, setQuery] = useState('');
  const [debouncedQuery, setDebouncedQuery] = useState('');
  const [sort, setSort] = useState<'name' | 'qtype' | 'answers' | 'expires'>('expires');
  const [order, setOrder] = useState<'asc' | 'desc'>('desc');
  const [view, setView] = useState<{ open: boolean; key?: string; data?: CacheKeyDetails } >({ open: false });
  const [confirmFlush, setConfirmFlush] = useState(false);

  // load persisted UI prefs
  useEffect(() => {
    try {
      const saved = JSON.parse(localStorage.getItem('cache.ui') || '{}');
      if (typeof saved.limit === 'number') setLimit(saved.limit);
      if (saved.sort) setSort(saved.sort);
      if (saved.order) setOrder(saved.order);
    } catch {}
  }, []);

  // persist UI prefs
  useEffect(() => {
    try { localStorage.setItem('cache.ui', JSON.stringify({ limit, sort, order })); } catch {}
  }, [limit, sort, order]);

  // debounce search
  useEffect(() => {
    const t = window.setTimeout(() => setDebouncedQuery(query), 300);
    return () => window.clearTimeout(t);
  }, [query]);

  useEffect(() => {
    // Subscribe WS snapshot
    const offWS = provider.onCache?.((data: CacheListResponse) => {
      if (!data || !Array.isArray(data.items)) return;
      setRows(data.items);
      setTotal(data.total || 0);
    });
    // Invalidation: refetch current page with rate-limit
    let lastFetch = 0;
    const offUpd = provider.onCacheUpdated?.(() => {
      const now = Date.now();
      if (now - lastFetch < 800) return; // rate limit 0.8s
      lastFetch = now;
      provider.fetchCache?.({ offset, limit, q: debouncedQuery, sort, order })
        .then((data) => { if (data) { setRows(data.items || []); setTotal(data.total || 0); } })
        .catch(() => {});
    });
    // initial fetch (REST)
    provider.fetchCache?.({ offset, limit, q: debouncedQuery, sort, order })
      .then((data) => { if (data) { setRows(data.items || []); setTotal(data.total || 0); } })
      .catch(() => {});
    return () => { offWS && offWS(); offUpd && offUpd(); };
  }, [provider, offset, limit, debouncedQuery, sort, order]);

  const totalPages = useMemo(() => Math.max(1, Math.ceil(total / limit)), [total, limit]);
  const page = useMemo(() => Math.floor(offset / limit) + 1, [offset, limit]);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Cache</h1>
        <p className="text-gray-600 dark:text-gray-400 mt-1">Manage local DNS cache</p>
      </div>

      <Card>
        <div className="space-y-4">
          {/* Toolbar */}
          <div className="flex flex-wrap items-end gap-3">
            <div className="flex-1 min-w-[260px]">
              <Input
                label="Domain"
                value={name}
                onInput={(e) => setName((e.target as HTMLInputElement).value)}
                placeholder="example.com"
              />
            </div>
            <div className="w-64">
              <Select label="Type (optional)" value={qtype} onChange={(e) => setQtype((e.target as HTMLSelectElement).value)}>
                <option value="">All</option>
                <option value="1">A</option>
                <option value="28">AAAA</option>
                <option value="5">CNAME</option>
                <option value="15">MX</option>
                <option value="2">NS</option>
                <option value="16">TXT</option>
                <option value="33">SRV</option>
                <option value="12">PTR</option>
              </Select>
            </div>
            <div className="ml-auto flex items-end gap-3">
              <Button
                type="button"
                variant="secondary"
                size="md"
                className="h-[42px]"
                disabled={busy}
                onClick={async () => {
                  const domain = extractDomain(name);
                  if (!domain) return;
                  setBusy(true);
                  try { await provider.cacheDelete({ name: domain, qtype: qtype ? parseInt(qtype, 10) : undefined }); } finally { setBusy(false); }
                }}
              >
                Delete domain cache
              </Button>
              <Button type="button" variant="outline" size="md" className="h-[42px]" disabled={busy} onClick={() => setConfirmFlush(true)}>Flush all</Button>
              <div className="flex items-center gap-2">
                <label className="text-sm text-gray-600 dark:text-gray-300">Per page</label>
                <Select value={String(limit)} onChange={(e) => setLimit(parseInt((e.target as HTMLSelectElement).value, 10))}>
                  <option value="10">10</option>
                  <option value="20">20</option>
                  <option value="50">50</option>
                </Select>
              </div>
            </div>
          </div>

          {/* Search */}
          <div>
            <Input label="Search" value={query} onInput={(e) => setQuery((e.target as HTMLInputElement).value)} placeholder="Filter by domain" />
          </div>

          <Table>
            <THead>
              <TRow>
                {[
                  { key: 'name', label: 'Domain' },
                  { key: 'qtype', label: 'Type' },
                  { key: 'answers', label: 'Answers' },
                  { key: 'expires', label: 'Expires' }
                ].map((c) => (
                  <TH key={c.key} sortable active={sort === (c.key as any)} order={order} onClick={() => { if (sort === (c.key as any)) { setOrder(order === 'asc' ? 'desc' : 'asc'); } else { setSort(c.key as any); setOrder('asc'); } setOffset(0); }}>
                    {c.label}
                  </TH>
                ))}
                <TH />
              </TRow>
            </THead>
            <TBody>
              {rows.map((r) => (
                <TRow key={r.key}>
                  <TD>{r.name}</TD>
                  <TD>{({1:'A',28:'AAAA',5:'CNAME',15:'MX',2:'NS',16:'TXT',33:'SRV',12:'PTR'} as any)[r.qtype] || r.qtype}</TD>
                  <TD>{r.answers}</TD>
                  <TD>{Math.max(0, Math.floor((new Date(r.expires_at as any).getTime() - Date.now())/1000))}s</TD>
                  <TD align="right">
                      <Button
                        type="button"
                        size="sm"
                        variant="outline"
                        onClick={async () => {
                          setBusy(true);
                          try { await provider.cacheDelete?.({ name: r.name, qtype: r.qtype }); await provider.fetchCache?.({ offset, limit, q: debouncedQuery, sort, order }); } finally { setBusy(false); }
                        }}
                      >
                        Delete
                      </Button>
                      <Button
                        type="button"
                        size="sm"
                        className="ml-2"
                        variant="secondary"
                        onClick={async () => {
                          try { const d = await (provider as any).restProvider.fetchCacheKey(r.key); setView({ open: true, key: r.key, data: d }); } catch {}
                        }}
                      >
                        View
                      </Button>
                  </TD>
                </TRow>
              ))}
              {rows.length === 0 && (
                <TRow>
                  <TD align="left" colSpan={5}> <span className="px-4 py-6 block text-sm text-gray-500 dark:text-gray-400">No cache entries</span> </TD>
                </TRow>
              )}
            </TBody>
          </Table>

          <div className="flex items-center justify-between">
            <div className="text-sm text-gray-600 dark:text-gray-400">Total: {total}</div>
            <div className="flex items-center gap-2">
              <Button type="button" variant="outline" size="sm" disabled={page <= 1} onClick={() => setOffset(Math.max(0, offset - limit))}>Prev</Button>
              <span className="text-sm text-gray-700 dark:text-gray-300">{page} / {totalPages}</span>
              <Button type="button" variant="outline" size="sm" disabled={page >= totalPages} onClick={() => setOffset(offset + limit)}>Next</Button>
            </div>
          </div>
        </div>
      </Card>
      <Modal isOpen={view.open} title={`Cache entry: ${view.key}`} onClose={() => setView({ open: false })}>
        {view.data ? (
          <div className="space-y-2">
            <div className="text-sm text-gray-700 dark:text-gray-300">RCODE: {view.data.rcode}</div>
            <pre className="p-3 rounded-md bg-gray-50 text-gray-800 dark:bg-gray-900 dark:text-gray-200 whitespace-pre-wrap text-sm border border-gray-200 dark:border-gray-700 max-h-80 overflow-auto">{(view.data.answers || []).join('\n')}</pre>
          </div>
        ) : (
          <div className="text-sm text-gray-600 dark:text-gray-400">Loading…</div>
        )}
      </Modal>

      <Modal isOpen={confirmFlush} title="Confirm flush" onClose={() => setConfirmFlush(false)}
        primaryAction={{ label: 'Flush all', onClick: async () => { setBusy(true); try { await provider.cacheFlush?.(); await provider.fetchCache?.({ offset: 0, limit, q: debouncedQuery, sort, order }); setOffset(0); } finally { setBusy(false); setConfirmFlush(false); } }, variant: 'primary' }}
        secondaryAction={{ label: 'Cancel', onClick: () => setConfirmFlush(false) }}
      >
        <div className="text-sm text-gray-700 dark:text-gray-300">This will clear the entire cache. Continue?</div>
      </Modal>
    </div>
  );
}
