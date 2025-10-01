// Data Transfer Objects (DTOs)
export interface RuleGroup {
  name: string;
  description?: string;
  via: string;
  patterns: string[];
  pin_ttl: boolean;
}

export interface QueryEvent {
  time: string;
  name: string;
  qtype: number;
  upstream: string;
  duration: string;
  status: 'ok' | 'error';
  client_ip: string;
}

export interface Stats {
  dns_queries_total: number;
  dns_marks_total?: number;
  marks_dropped_total?: number;
  dns_upstream_rtt_avg_seconds?: number;
  dns_request_avg_seconds?: number;
  service_ready?: number;
  effective_rps?: number;
  origin_rps?: number;
  cache_hit_rate?: number;
}

export interface OverviewData {
  stats: Stats;
  rules_total: number;
  upstreams_total: number;
  history_total?: number;
  uptime?: string;
  queries_last_min?: number;
  errors_last_min?: number;
}

export interface HostOverride {
  pattern: string;
  a?: string[];
  aaaa?: string[];
  ttl?: number;
}

export interface Lease {
  hostname: string;
  ip: string;
  mac: string;
  expires_at: string;
  id?: string;
}

export interface LocalZonesData {
  zones: string[];
  leases: Lease[];
}

export interface ServerInfo {
  version: string;
  go_version: string;
  os: string;
  arch: string;
  admin_port: number;
  dns_port: string;
  uptime: string;
  build_time?: string;
}

// Upstream item used in UI
export type UpstreamItem = {
  name: string;
  address: string; // URL form (udp://host:port or https://...)
} & Partial<{ weight: number }>;

export interface Config {
  history_enabled: boolean;
  cache_enabled: boolean;
  log_level: string;
}

// WebSocket message types
export type WSMessageType = 'stats' | 'history' | 'rule_groups' | 'upstreams' | 'hosts' | 'overview' | 'update_available' | 'cache' | 'cache_updated' | 'local_zones';

export interface WSMessage {
  type: WSMessageType;
  data: any;
}

// Provider interface
export interface Provider {
  connect(): Promise<void>;
  close(): void;
  // Channel status events (ws/rest)
  onStatus?(cb: (status: { channel: 'ws' | 'rest'; connected: boolean }) => void): () => void;
  reconnect?(): Promise<void>;
  onRuleGroups(cb: (groups: RuleGroup[]) => void): () => void;
  onUpstreams(cb: (items: UpstreamItem[]) => void): () => void;
  onStats(cb: (s: Stats) => void): () => void;
  onHistory(cb: (evs: QueryEvent[]) => void): () => void;
  onHosts(cb: (hosts: HostOverride[]) => void): () => void;
  onUpdateAvailable(cb: (updateInfo: any) => void): () => void;
  fetchRuleGroups(): Promise<RuleGroup[]>;
  createRuleGroup(group: RuleGroup): Promise<void>;
  updateRuleGroup(name: string, group: RuleGroup): Promise<void>;
  deleteRuleGroup(name: string): Promise<void>;
  fetchUpstreams(): Promise<UpstreamItem[]>;
  saveUpstreams(items: UpstreamItem[]): Promise<void>;
  fetchStats(): Promise<Stats>;
  fetchHistory(): Promise<QueryEvent[]>;
  fetchHosts(): Promise<HostOverride[]>;
  saveHosts(hosts: HostOverride[]): Promise<void>;
  fetchServerInfo(): Promise<ServerInfo>;
  // Run a single DNS resolve through active pipeline
  testResolve(name: string, type: string): Promise<ResolveResult>;
  // Cache listing and events
  onCache?(cb: (data: CacheListResponse) => void): () => void;
  onCacheUpdated?(cb: () => void): () => void;
  fetchCache?(params?: { offset?: number; limit?: number; q?: string; sort?: string; order?: 'asc' | 'desc' }): Promise<CacheListResponse>;
  cacheFlush?(): Promise<CacheOpResponse>;
  cacheDelete?(req: CacheDeleteRequest): Promise<CacheOpResponse>;
  // Local DNS (LAN resolver)
  onLocalZones?(cb: (data: LocalZonesData) => void): () => void;
  fetchLocalZones?(): Promise<{ zones: string[] }>;
  fetchLocalLeases?(): Promise<{ leases: Lease[] }>;
  resolveLocal?(name: string): Promise<ResolveResult>;
}

// API Response types
export interface RuleGroupsResponse {
  rule_groups: RuleGroup[];
}

export interface UpstreamsResponse {
  upstreams: Array<{ name: string; address: string; type?: string; weight?: number }>;
}

export interface HistoryResponse {
  events: QueryEvent[];
}

export interface ResolveResult {
  upstream: string;
  rcode: number;
  answers: number;
  records: string[];
  ttl?: number;
  response_time_ms?: number;
}

// Chart data types
export interface ChartDataPoint {
  timestamp: number;
  value: number;
}

export interface ChartWindow {
  label: string;
  seconds: number;
}

// Constants
export const CHART_WINDOWS: ChartWindow[] = [
  { label: '1m', seconds: 60 },
  { label: '5m', seconds: 300 },
  { label: '15m', seconds: 900 }
];

export const REFRESH_INTERVALS = [
  { label: '10s', ms: 10000 },
  { label: '30s', ms: 30000 },
  { label: '1m', ms: 60000 }
] as const;

export const QTYPE_NAMES: Record<number, string> = {
  1: 'A',
  28: 'AAAA',
  2: 'NS',
  5: 'CNAME',
  6: 'SOA',
  12: 'PTR',
  15: 'MX',
  16: 'TXT',
  33: 'SRV'
};

export type CacheDeleteRequest = { name: string } | { name: string; qtype: number | undefined };

export interface CacheOpResponse {
  status: string;
}

export interface CacheEntry {
  key: string;
  name: string;
  qtype: number;
  answers: number;
  expires_at: string | Date;
}

export interface CacheListResponse {
  items: CacheEntry[];
  total: number;
  offset: number;
  limit: number;
}

export interface CacheKeyDetails {
  key: string;
  answers: string[];
  rcode: number;
}
