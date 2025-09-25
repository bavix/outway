import { 
  Provider, 
  RuleGroup, 
  Stats, 
  QueryEvent, 
  ServerInfo,
  RuleGroupsResponse, 
  UpstreamsResponse, 
  HistoryResponse,
  HostOverride,
  ResolveResult,
  UpstreamItem,
  OverviewData 
} from './types.js';

export class RESTProvider implements Provider {
  private pollingTimers = new Map<string, number>();
  private callbacks = new Map<string, Set<Function>>();
  private pollingConfig = {
    stats: 30000,      // 30s
    history: 60000,    // 1m
    rule_groups: 300000,     // 5m
    upstreams: 300000,  // 5m
    hosts: 300000
  };

  constructor(private baseUrl: string = '') {}

  async connect(): Promise<void> {
    // Start polling for all data types
    this.startPolling('stats', () => this.fetchStats());
    this.startPolling('history', () => this.fetchHistory());
    this.startPolling('rule_groups', () => this.fetchRuleGroups());
    this.startPolling('upstreams', () => this.fetchUpstreams());
    this.startPolling('hosts', () => this.fetchHosts());
  }

  close(): void {
    // Clear all polling timers
    this.pollingTimers.forEach(timer => clearInterval(timer));
    this.pollingTimers.clear();
    this.callbacks.clear();
  }

  private async fetchJSON<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
    const response = await fetch(`${this.baseUrl}${endpoint}`, {
      headers: {
        'Content-Type': 'application/json',
        ...options.headers
      },
      ...options
    });

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${await response.text()}`);
    }

    const contentType = response.headers.get('content-type');
    if (contentType?.includes('application/json')) {
      return response.json();
    }
    
    return response.text() as unknown as T;
  }

  private startPolling(type: string, fetcher: () => Promise<any>): void {
    const interval = this.pollingConfig[type as keyof typeof this.pollingConfig];
    
    // Fetch immediately
    fetcher().then(data => {
      this.notifyCallbacks(type, data);
    }).catch(console.error);

    // Set up polling
    const timer = window.setInterval(() => {
      fetcher().then(data => {
        this.notifyCallbacks(type, data);
      }).catch(console.error);
    }, interval);

    this.pollingTimers.set(type, timer);
  }

  private notifyCallbacks(type: string, data: any): void {
    const callbacks = this.callbacks.get(type);
    if (callbacks) {
      callbacks.forEach(cb => {
        try {
          cb(data);
        } catch (error) {
          console.error('Polling callback error:', error);
        }
      });
    }
  }

  // Event subscription methods
  onRuleGroups(cb: (groups: RuleGroup[]) => void): () => void {
    return this.subscribe('rule_groups', cb);
  }

  onUpstreams(cb: (items: UpstreamItem[]) => void): () => void {
    return this.subscribe('upstreams', cb);
  }

  onStats(cb: (s: Stats) => void): () => void {
    return this.subscribe('stats', cb);
  }

  onHistory(cb: (evs: QueryEvent[]) => void): () => void {
    return this.subscribe('history', cb);
  }

  onHosts(cb: (hosts: HostOverride[]) => void): () => void {
    return this.subscribe('hosts', cb);
  }

  onUpdateAvailable(_cb: (updateInfo: any) => void): () => void {
    // REST provider doesn't support real-time update notifications
    // This is a no-op for REST fallback
    return () => {};
  }

  private subscribe(type: string, cb: Function): () => void {
    if (!this.callbacks.has(type)) {
      this.callbacks.set(type, new Set());
    }
    
    this.callbacks.get(type)!.add(cb);
    
    return () => {
      const callbacks = this.callbacks.get(type);
      if (callbacks) {
        callbacks.delete(cb);
        if (callbacks.size === 0) {
          this.callbacks.delete(type);
        }
      }
    };
  }

  // REST API methods
  async fetchRuleGroups(): Promise<RuleGroup[]> {
    const response = await this.fetchJSON<RuleGroupsResponse>('/api/v1/rule-groups');
    return response.rule_groups;
  }

  async createRuleGroup(group: RuleGroup): Promise<void> {
    await this.fetchJSON('/api/v1/rule-groups', {
      method: 'POST',
      body: JSON.stringify(group)
    });
  }

  async updateRuleGroup(name: string, group: RuleGroup): Promise<void> {
    await this.fetchJSON(`/api/v1/rule-groups/${encodeURIComponent(name)}`, {
      method: 'PUT',
      body: JSON.stringify(group)
    });
  }

  async deleteRuleGroup(name: string): Promise<void> {
    await this.fetchJSON(`/api/v1/rule-groups/${encodeURIComponent(name)}`, {
      method: 'DELETE'
    });
  }

  async fetchUpstreams(): Promise<UpstreamItem[]> {
    const response = await this.fetchJSON<UpstreamsResponse>('/api/v1/upstreams');
    // Normalize to UpstreamItem with URL-form address and preserve name/weight
    return response.upstreams.map((u) => {
      const name = (u as any).name || (u as any).Name || '';
      const weight = (u as any).weight ?? (u as any).Weight;
      const t = (u as any).type?.toString().toLowerCase();
      const addr = (u as any).address?.toString() || '';
      let address = addr;
      // If address already contains a known scheme, keep as-is
      if (/^(https?|udp|tcp|tls|dot|quic|doq):\/\//i.test(addr)) {
        address = addr;
      } else if (t && t !== 'doh') {
        // t is one of udp|tcp|dot|doq etc.; ensure scheme prefix
        address = `${t}://${addr.replace(/^(udp|tcp|tls|dot|quic|doq):\/\//i, '')}`;
      }
      return { name, address, weight } as UpstreamItem;
    });
  }

  async saveUpstreams(items: UpstreamItem[]): Promise<void> {
    // Convert UpstreamItem (with URL address) to backend objects WITHOUT 'type'
    // Ensure address is URL with scheme; auto-derive name if missing
    const upstreams = items.map((raw) => {
      const s = String(raw.address || '').trim();
      let name = (raw.name || '').trim();
      const weight = typeof raw.weight === 'number' ? raw.weight : undefined;
      try {
        const url = new URL(s);
        const scheme = url.protocol.replace(':', '').toLowerCase();
        // Normalize address to include scheme explicitly
        let address = s;
        if (scheme !== 'https') {
          // For udp/tcp/dot/doq/tls/quic, collapse to canonical schemes
          const canonical = scheme === 'tls' ? 'dot' : scheme === 'quic' ? 'doq' : scheme;
          address = `${canonical}://${url.host}`;
        }
        if (!name) name = url.hostname;
        return { name, address, ...(weight !== undefined ? { weight } : {}) } as any;
      } catch {}
      // Fallback: assume raw host[:port] → udp URL, derive name from host
      const host = s.replace(/^(udp|tcp|tls|dot|quic|doq):\/\//i, '');
      if (!name) name = host.split(':')[0] || host;
      const address = `udp://${host}`;
      return { name, address, ...(weight !== undefined ? { weight } : {}) } as any;
    });
    await this.fetchJSON('/api/v1/upstreams', {
      method: 'POST',
      body: JSON.stringify({ upstreams })
    });
  }

  async fetchStats(): Promise<Stats> {
    return this.fetchJSON<Stats>('/api/v1/stats');
  }

  async fetchHistory(): Promise<QueryEvent[]> {
    const response = await this.fetchJSON<HistoryResponse>('/api/v1/history');
    return response.events;
  }

  async fetchHosts(): Promise<HostOverride[]> {
    const response = await this.fetchJSON<{ hosts: HostOverride[] }>('/api/v1/hosts');
    return response.hosts;
  }

  async saveHosts(hosts: HostOverride[]): Promise<void> {
    await this.fetchJSON('/api/v1/hosts', {
      method: 'PUT',
      body: JSON.stringify({ hosts })
    });
  }

  async fetchServerInfo(): Promise<ServerInfo> {
    return this.fetchJSON<ServerInfo>('/api/v1/info');
  }

  async testResolve(name: string, type: string): Promise<ResolveResult> {
    const params = new URLSearchParams({ name, type });
    return this.fetchJSON<ResolveResult>(`/api/v1/resolve?${params.toString()}`);
  }

  // Overview (REST)
  onOverview(cb: (ov: OverviewData) => void): () => void {
    return this.subscribe('overview', cb);
  }
  async fetchOverview(): Promise<OverviewData> {
    return this.fetchJSON<OverviewData>('/api/v1/overview');
  }
}
