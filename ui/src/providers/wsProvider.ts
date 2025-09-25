import { 
  Provider, 
  RuleGroup, 
  Stats, 
  QueryEvent, 
  WSMessage, 
  WSMessageType,
  HostOverride,
  UpstreamItem,
  OverviewData
} from './types.js';

export class WSProvider implements Provider {
  private ws: WebSocket | null = null;
  private reconnectTimeout: number | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 10;
  private baseDelay = 500;
  private maxDelay = 10000;
  private callbacks = new Map<string, Set<Function>>();
  private lastSnapshot: Record<string, any> = {};

  constructor(private url: string) {}

  async connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      try {
        const wsUrl = this.url.replace(/^http/, 'ws');
        this.ws = new WebSocket(wsUrl);
        
        this.ws.onopen = () => {
          console.log('WebSocket connected');
          this.reconnectAttempts = 0;
          this.clearReconnectTimeout();
          resolve();
        };

        this.ws.onmessage = (event) => {
          try {
            const message: WSMessage = JSON.parse(event.data);
            this.handleMessage(message);
          } catch (error) {
            console.warn('Invalid WebSocket message:', error);
          }
        };

        this.ws.onclose = () => {
          console.log('WebSocket disconnected');
          this.scheduleReconnect();
        };

        this.ws.onerror = (error) => {
          console.error('WebSocket error:', error);
          reject(error);
        };

        // Set read deadline simulation (60s)
        this.ws.addEventListener('ping', () => {
          if (this.ws?.readyState === WebSocket.OPEN) {
            this.ws.send('pong');
          }
        });

      } catch (error) {
        reject(error);
      }
    });
  }

  close(): void {
    this.clearReconnectTimeout();
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  private handleMessage(message: WSMessage): void {
    let payload: any = message.data;
    // Normalize payloads to UI expectations
    if (message.type === 'upstreams' && Array.isArray(payload)) {
      const prev = (this.lastSnapshot['upstreams'] as UpstreamItem[] | undefined) || [];
      payload = payload.map((u: any, idx: number) => {
        const name = (u.name || u.Name || '').toString();
        const weight = (u.weight ?? u.Weight);
        const t = (u.type || u.Type || '').toString().toLowerCase();
        const a = (u.address || u.Address || '').toString();
        let address = a;
        if (!/^(https?|udp|tcp|tls|dot|quic|doq):\/\//i.test(a)) {
          if (t && t !== 'doh') {
            address = `${t}://${a}`;
          } else {
            // No scheme and no type → assume udp
            address = `udp://${a}`;
          }
        }
        if (!address || address === 'udp://') {
          // Try to preserve previous known address by name or index
          const byName = prev.find(p => p.name === name);
          address = byName?.address || prev[idx]?.address || address;
        }
        return { name, address, weight } as UpstreamItem;
      });
    }
    if (message.type === 'history') {
      // Accept both array and wrapped {events: []}
      const arr = Array.isArray(payload) ? payload : (payload && Array.isArray(payload.events) ? payload.events : []);
      payload = arr;
      try { console.debug('[WS] history', Array.isArray(arr) ? arr.length : 0); } catch {}
    }
    if (message.type === 'overview' && payload) {
      try { (window as any).__ov = payload; } catch {}
    }

    // Update last snapshot
    this.lastSnapshot[message.type] = payload;

    // Notify callbacks
    const callbacks = this.callbacks.get(message.type);
    if (callbacks) {
      callbacks.forEach(cb => {
        try { cb(payload); } catch (error) { console.error('Callback error:', error); }
      });
    }
  }

  private scheduleReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('Max reconnection attempts reached');
      return;
    }

    this.clearReconnectTimeout();
    
    const delay = Math.min(
      this.baseDelay * Math.pow(1.8, this.reconnectAttempts),
      this.maxDelay
    );
    
    // Add jitter (±25%)
    const jitter = delay * 0.25;
    const jitteredDelay = delay + (Math.random() - 0.5) * jitter;
    
    this.reconnectTimeout = window.setTimeout(() => {
      this.reconnectAttempts++;
      console.log(`Attempting to reconnect (${this.reconnectAttempts}/${this.maxReconnectAttempts})`);
      this.connect().catch(console.error);
    }, jitteredDelay);
  }

  private clearReconnectTimeout(): void {
    if (this.reconnectTimeout) {
      clearTimeout(this.reconnectTimeout);
      this.reconnectTimeout = null;
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

  // Overview
  onOverview(cb: (ov: OverviewData) => void): () => void {
    return this.subscribe('overview' as WSMessageType, cb);
  }

  private subscribe(type: WSMessageType, cb: Function): () => void {
    if (!this.callbacks.has(type)) {
      this.callbacks.set(type, new Set());
    }
    
    this.callbacks.get(type)!.add(cb);
    
    // Send current snapshot if available
    if (this.lastSnapshot[type]) {
      try {
        cb(this.lastSnapshot[type]);
      } catch (error) {
        console.error('Snapshot callback error:', error);
      }
    }
    
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

  // REST fallback methods (will be handled by FailoverProvider)
  async fetchRuleGroups(): Promise<RuleGroup[]> {
    throw new Error('Use REST provider for fetch operations');
  }

  async createRuleGroup(_group: RuleGroup): Promise<void> {
    throw new Error('Use REST provider for mutation operations');
  }

  async updateRuleGroup(_name: string, _group: RuleGroup): Promise<void> {
    throw new Error('Use REST provider for mutation operations');
  }

  async deleteRuleGroup(_name: string): Promise<void> {
    throw new Error('Use REST provider for mutation operations');
  }

  async fetchUpstreams(): Promise<UpstreamItem[]> {
    throw new Error('Use REST provider for fetch operations');
  }

  async saveUpstreams(_items: UpstreamItem[]): Promise<void> {
    throw new Error('Use REST provider for mutation operations');
  }

  async fetchStats(): Promise<Stats> {
    throw new Error('Use REST provider for fetch operations');
  }

  async fetchHistory(): Promise<QueryEvent[]> {
    throw new Error('Use REST provider for fetch operations');
  }

  async fetchServerInfo(): Promise<any> {
    throw new Error('Use REST provider for fetch operations');
  }

  async fetchConfig(): Promise<any> {
    throw new Error('Use REST provider for fetch operations');
  }

  async updateConfig(_config: any): Promise<void> {
    throw new Error('Use REST provider for mutation operations');
  }

  async fetchHosts(): Promise<any> {
    throw new Error('Use REST provider for fetch operations');
  }

  async saveHosts(_hosts: any): Promise<void> {
    throw new Error('Use REST provider for mutation operations');
  }

  async testResolve(_name: string, _type: 'A' | 'AAAA' | 'CNAME'): Promise<import('./types.js').ResolveResult> {
    throw new Error('Use REST provider for fetch operations');
  }
}
