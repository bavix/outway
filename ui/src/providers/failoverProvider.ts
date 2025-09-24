import { Provider, RuleGroup, Stats, QueryEvent, ServerInfo, HostOverride, ResolveResult, UpstreamItem, OverviewData } from './types.js';
import { WSProvider } from './wsProvider.js';
import { RESTProvider } from './restProvider.js';

export class FailoverProvider implements Provider {
  private wsProvider: WSProvider;
  private restProvider: RESTProvider;
  private isWSConnected = false;
  private listeners = new Map<string, Set<Function>>();
  private wsUnsubs = new Map<string, () => void>();
  private restUnsubs = new Map<string, () => void>();
  private statusListeners = new Set<(s: { channel: 'ws' | 'rest'; connected: boolean }) => void>();

  constructor(wsUrl: string = '/ws', restUrl: string = '') {
    this.wsProvider = new WSProvider(wsUrl);
    this.restProvider = new RESTProvider(restUrl);
  }

  async connect(): Promise<void> {
    // Try WebSocket first
    try {
      await this.wsProvider.connect();
      this.isWSConnected = true;
      console.log('Using WebSocket provider');
    } catch (error) {
      console.warn('WebSocket connection failed, falling back to REST:', error);
      this.isWSConnected = false;
    }

    // Always start REST as fallback
    try {
      await this.restProvider.connect();
      console.log('REST provider ready');
    } catch (error) {
      console.error('REST provider failed:', error);
      throw error;
    }

    // Subscribe existing listeners to the preferred channel
    this.resubscribeAll();
    // Set up WebSocket reconnection monitoring
    this.monitorWSConnection();
    this.emitStatus();
  }

  close(): void {
    this.wsProvider.close();
    this.restProvider.close();
    this.wsUnsubs.forEach(u => u());
    this.restUnsubs.forEach(u => u());
    this.wsUnsubs.clear();
    this.restUnsubs.clear();
    this.listeners.clear();
    this.statusListeners.clear();
  }

  private monitorWSConnection(): void {
    // Debounced check to avoid false REST fallback flashes on cold start
    const checkInterval = setInterval(() => {
      const wasConnected = this.isWSConnected;
      
      // Simple connectivity check - if WebSocket is null or closed, switch to REST
      const ready = (this.wsProvider as any).ws?.readyState === WebSocket.OPEN;
      // apply small grace period: only flip to REST if we've been closed for >1.5s
      if (!ready) {
        setTimeout(() => {
          const stillClosed = (this.wsProvider as any).ws?.readyState !== WebSocket.OPEN;
          const prev = this.isWSConnected;
          this.isWSConnected = !stillClosed ? true : false;
          if (prev !== this.isWSConnected) {
            this.resubscribeAll();
            console.log(this.isWSConnected ? 'WebSocket active' : 'Using REST fallback');
            this.emitStatus();
          }
        }, 1500);
        return;
      }
      this.isWSConnected = true;
      if (wasConnected !== true) {
        this.resubscribeAll();
        console.log(this.isWSConnected ? 'WebSocket active' : 'Using REST fallback');
        this.emitStatus();
      }
    }, 10000); // Check every 10s instead of 5s

    // Clean up on close
    this.wsUnsubs.set('_monitor', () => clearInterval(checkInterval));
  }

  // Generic subscribe helper: keeps a logical listener and binds it to the active channel
  private addListener(topic: string, cb: Function): () => void {
    if (!this.listeners.has(topic)) this.listeners.set(topic, new Set());
    this.listeners.get(topic)!.add(cb);
    this.resubscribe(topic); // bind immediately
    return () => {
      const set = this.listeners.get(topic);
      if (set) {
        set.delete(cb);
        if (set.size === 0) this.listeners.delete(topic);
      }
      // Unsubscribe from both channels for safety
      const uw = this.wsUnsubs.get(topic); if (uw) uw();
      const ur = this.restUnsubs.get(topic); if (ur) ur();
      this.wsUnsubs.delete(topic);
      this.restUnsubs.delete(topic);
    };
  }

  private resubscribe(topic: string): void {
    // Tear down old binds
    const uw = this.wsUnsubs.get(topic); if (uw) uw();
    const ur = this.restUnsubs.get(topic); if (ur) ur();
    this.wsUnsubs.delete(topic);
    this.restUnsubs.delete(topic);
    const set = this.listeners.get(topic);
    if (!set || set.size === 0) return;
    const multiCb = (data: any) => set.forEach(fn => { try { fn(data); } catch {} });
    // Prefer WS if connected; maintain REST backup only when WS is down
    if (this.isWSConnected) {
      switch (topic) {
        case 'rule_groups': this.wsUnsubs.set(topic, this.wsProvider.onRuleGroups(multiCb)); break;
        case 'upstreams': this.wsUnsubs.set(topic, this.wsProvider.onUpstreams(multiCb)); break;
        case 'stats': this.wsUnsubs.set(topic, this.wsProvider.onStats(multiCb)); break;
        case 'history': this.wsUnsubs.set(topic, this.wsProvider.onHistory(multiCb)); break;
        case 'hosts': this.wsUnsubs.set(topic, this.wsProvider.onHosts(multiCb)); break;
        case 'overview': this.wsUnsubs.set(topic, this.wsProvider.onOverview(multiCb)); break;
      }
    } else {
      switch (topic) {
        case 'rule_groups': this.restUnsubs.set(topic, this.restProvider.onRuleGroups(multiCb)); break;
        case 'upstreams': this.restUnsubs.set(topic, this.restProvider.onUpstreams(multiCb)); break;
        case 'stats': this.restUnsubs.set(topic, this.restProvider.onStats(multiCb)); break;
        case 'history': this.restUnsubs.set(topic, this.restProvider.onHistory(multiCb)); break;
        case 'hosts': this.restUnsubs.set(topic, this.restProvider.onHosts(multiCb)); break;
        case 'overview': this.restUnsubs.set(topic, this.restProvider.onOverview(multiCb)); break;
      }
    }
  }

  private resubscribeAll(): void {
    Array.from(this.listeners.keys()).forEach(t => this.resubscribe(t));
  }

  private emitStatus(): void {
    const status = { channel: this.isWSConnected ? 'ws' : 'rest', connected: true } as const;
    this.statusListeners.forEach(fn => { try { fn(status); } catch {} });
  }

  onStatus(cb: (status: { channel: 'ws' | 'rest'; connected: boolean }) => void): () => void {
    this.statusListeners.add(cb);
    // fire immediately
    cb({ channel: this.isWSConnected ? 'ws' : 'rest', connected: true });
    return () => { this.statusListeners.delete(cb); };
  }

  async reconnect(): Promise<void> {
    try {
      await this.wsProvider.connect();
      this.isWSConnected = true;
      this.resubscribeAll();
      this.emitStatus();
    } catch (e) {
      // stay on REST
    }
  }

  // Event subscription with automatic failover
  onRuleGroups(cb: (groups: RuleGroup[]) => void): () => void { return this.addListener('rule_groups', cb); }

  onUpstreams(cb: (items: UpstreamItem[]) => void): () => void { return this.addListener('upstreams', cb); }

  onStats(cb: (s: Stats) => void): () => void { return this.addListener('stats', cb); }

  onHistory(cb: (evs: QueryEvent[]) => void): () => void { return this.addListener('history', cb); }

  onHosts(cb: (hosts: HostOverride[]) => void): () => void { return this.addListener('hosts', cb); }
  onOverview(cb: (ov: OverviewData) => void): () => void { return this.addListener('overview', cb); }

  // All mutations go through REST (optimistic updates handled by store)
  async fetchRuleGroups(): Promise<RuleGroup[]> {
    try {
      return await this.restProvider.fetchRuleGroups();
    } catch (error) {
      console.error('Failed to fetch rule groups:', error);
      throw error;
    }
  }

  async createRuleGroup(group: RuleGroup): Promise<void> {
    try {
      await this.restProvider.createRuleGroup(group);
    } catch (error) {
      console.error('Failed to create rule group:', error);
      throw error;
    }
  }

  async updateRuleGroup(name: string, group: RuleGroup): Promise<void> {
    try {
      await this.restProvider.updateRuleGroup(name, group);
    } catch (error) {
      console.error('Failed to update rule group:', error);
      throw error;
    }
  }

  async deleteRuleGroup(name: string): Promise<void> {
    try {
      await this.restProvider.deleteRuleGroup(name);
    } catch (error) {
      console.error('Failed to delete rule group:', error);
      throw error;
    }
  }

  async fetchUpstreams(): Promise<UpstreamItem[]> {
    try {
      // Avoid duplicate initial fetch if WS will deliver a snapshot soon
      if (this.isWSConnected && (this as any).wsProvider && (this as any).wsProvider['lastSnapshot']?.['upstreams']) {
        return (this as any).wsProvider['lastSnapshot']['upstreams'];
      }
      return await this.restProvider.fetchUpstreams();
    } catch (error) {
      console.error('Failed to fetch upstreams:', error);
      throw error;
    }
  }

  async saveUpstreams(items: UpstreamItem[]): Promise<void> {
    try {
      await this.restProvider.saveUpstreams(items);
    } catch (error) {
      console.error('Failed to save upstreams:', error);
      throw error;
    }
  }

  async fetchStats(): Promise<Stats> {
    try {
      return await this.restProvider.fetchStats();
    } catch (error) {
      console.error('Failed to fetch stats:', error);
      throw error;
    }
  }

  async fetchHistory(): Promise<QueryEvent[]> {
    try {
      return await this.restProvider.fetchHistory();
    } catch (error) {
      console.error('Failed to fetch history:', error);
      throw error;
    }
  }

  async fetchHosts(): Promise<HostOverride[]> {
    try {
      return await this.restProvider.fetchHosts();
    } catch (error) {
      console.error('Failed to fetch hosts:', error);
      throw error;
    }
  }

  async saveHosts(hosts: HostOverride[]): Promise<void> {
    try {
      await this.restProvider.saveHosts(hosts);
    } catch (error) {
      console.error('Failed to save hosts:', error);
      throw error;
    }
  }

  async fetchServerInfo(): Promise<ServerInfo> {
    try {
      return await this.restProvider.fetchServerInfo();
    } catch (error) {
      console.error('Failed to fetch server info:', error);
      throw error;
    }
  }

  async fetchOverview(): Promise<OverviewData> {
    try {
      return await this.restProvider.fetchOverview();
    } catch (error) {
      console.error('Failed to fetch overview:', error);
      throw error;
    }
  }


  async testResolve(name: string, type: string): Promise<ResolveResult> {
    try {
      return await this.restProvider.testResolve(name, type);
    } catch (error) {
      console.error('Failed to run resolve test:', error);
      throw error;
    }
  }
}
