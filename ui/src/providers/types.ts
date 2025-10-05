// Authentication types
export interface LoginRequest {
  email: string;
  password: string;
}

export interface LoginResponse {
  access_token: string;
  refresh_token: string;
  user: {
    email: string;
    role: string;
  };
}

export interface RefreshRequest {
  refresh_token: string;
}

export interface RefreshResponse {
  access_token: string;
  refresh_token: string;
}

export interface FirstUserRequest {
  email: string;
  password: string;
}

export interface Permission {
  name: string;
  description: string;
  category: string;
}

export interface UserResponse {
  email: string;
  role: string;
  permissions?: Permission[];
}

export interface UserRequest {
  email: string;
  password: string;
  role: string;
}

export interface Role {
  name: string;
  description: string;
  permissions_count: number;
}

export interface RolesResponse {
  roles: Role[];
  count: number;
}

export interface RolePermissionsResponse {
  role: string;
  permissions: Permission[];
  categories: Record<string, Permission[]>;
  count: number;
}

export interface UsersResponse {
  users: UserResponse[];
  count: number;
}

export interface AuthStatusResponse {
  users_exist: boolean;
}

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
export type WSMessageType = 'stats' | 'history' | 'rule_groups' | 'upstreams' | 'hosts' | 'overview' | 'update_available' | 'cache' | 'cache_updated';

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
  // Authentication methods
  getAuthStatus(): Promise<AuthStatusResponse>;
  login(credentials: LoginRequest): Promise<LoginResponse>;
  refreshToken(refreshToken: string): Promise<RefreshResponse>;
  createFirstUser(user: FirstUserRequest): Promise<LoginResponse>;
  // User management
  fetchUsers(): Promise<UsersResponse>;
  createUser(user: UserRequest): Promise<UserResponse>;
  getUser(login: string): Promise<UserResponse>;
  updateUser(login: string, user: UserRequest): Promise<UserResponse>;
  deleteUser(login: string): Promise<void>;
  changePassword(login: string, newPassword: string): Promise<void>;
  // Role and permissions
  fetchRoles(): Promise<RolesResponse>;
  fetchRolePermissions(role: string): Promise<RolePermissionsResponse>;
  // Data fetching
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
  // Generic event listener methods
  on(event: string, callback: (data: any) => void): void;
  off(event: string, callback: (data: any) => void): void;
  // Generic request method
  request(method: string, url: string, body?: any): Promise<Response>;
  cacheFlush?(): Promise<CacheOpResponse>;
  cacheDelete?(req: CacheDeleteRequest): Promise<CacheOpResponse>;
  // Local DNS methods
  fetchLocalZones?(): Promise<string[]>;
  fetchLocalLeases?(): Promise<any[]>;
  testLocalResolve?(name: string): Promise<any>;
  // Wake-on-LAN methods
  fetchWOLDevices?(): Promise<WOLDevicesResponse>;
  fetchWOLInterfaces?(): Promise<WOLInterfacesResponse>;
  fetchWOLConfig?(): Promise<WOLConfig>;
  updateWOLConfig?(config: Partial<WOLConfig>): Promise<void>;
  scanWOLDevices?(): Promise<WOLDevicesResponse>;
  addWOLDevice?(device: { name: string; mac: string; ip: string; hostname?: string; vendor?: string }): Promise<WOLDevice>;
  updateWOLDevice?(id: string, device: { name: string; mac: string; ip: string; hostname?: string; vendor?: string; status?: string }): Promise<void>;
  deleteWOLDevice?(id: string): Promise<void>;
  // Devices API methods
  fetchDevices?(): Promise<DevicesResponse>;
  fetchDevice?(id: string): Promise<Device>;
  addDevice?(device: { name: string; mac: string; ip: string; hostname?: string; vendor?: string }): Promise<Device>;
  updateDevice?(id: string, device: { name?: string; mac?: string; ip?: string; hostname?: string; vendor?: string; status?: string }): Promise<void>;
  deleteDevice?(id: string): Promise<void>;
  fetchDevicesByType?(type: string): Promise<DevicesResponse>;
  fetchOnlineDevices?(): Promise<DevicesResponse>;
  fetchWakeableDevices?(): Promise<DevicesResponse>;
  fetchResolvableDevices?(): Promise<DevicesResponse>;
  scanDevices?(): Promise<DevicesResponse>;
  fetchDeviceStats?(): Promise<DeviceStats>;
  wakeDevice?(request: DeviceWakeRequest): Promise<DeviceWakeResponse>;
  wakeAllDevices?(): Promise<DeviceWakeResponse[]>;
  resolveDevice?(id: string): Promise<ResolveResult>;
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

// Device types (unified)
export interface Device {
  id: string;
  name: string;
  mac: string;
  ip: string;
  hostname: string;
  vendor: string;
  status: 'online' | 'offline' | 'unknown';
  source: 'dhcp' | 'manual' | 'scan';
  device_type: 'computer' | 'phone' | 'router' | 'tv' | 'smart_home' | 'unknown';
  capabilities: {
    can_wake: boolean;
    can_resolve: boolean;
  };
  created_at: string;
  updated_at: string;
  last_seen: string;
}

// Wake-on-LAN types (legacy compatibility)
export interface WOLDevice {
  id: string;
  name: string;
  mac: string;
  ip: string;
  lastSeen: string;
  status: 'online' | 'offline' | 'unknown';
  vendor?: string;
  hostname?: string;
  can_wake?: boolean;
}

export interface WOLNetworkInterface {
  name: string;
  index: number;
  mtu: number;
  hardware_addr: string;
  ips: string[];
  broadcast: string;
  is_up: boolean;
  is_loopback: boolean;
}

export interface WOLConfig {
  enabled: boolean;
  default_port: number;
  default_timeout: number;
  retry_count: number;
  retry_delay: number;
}

export interface WOLWakeRequest {
  mac: string;
  hostname?: string;
  port?: number;
  timeout?: number;
  interface?: string;
}

export interface WOLWakeResponse {
  success: boolean;
  message: string;
  mac: string;
  interface?: string;
  sent_at: string;
  duration_ms: number;
}

export interface WOLDevicesResponse {
  devices: WOLDevice[];
  count: number;
}

export interface WOLInterfacesResponse {
  interfaces: WOLNetworkInterface[];
  count: number;
}

// Devices API types
export interface DevicesResponse {
  devices: Device[];
  count: number;
}

export interface DeviceStats {
  total: number;
  online: number;
  offline: number;
  by_type: Record<string, number>;
  by_source: Record<string, number>;
}

export interface DeviceWakeRequest {
  id: string;
  interface?: string;
}

export interface DeviceWakeResponse {
  success: boolean;
  message: string;
  device_id: string;
  interface?: string;
  sent_at: string;
  duration_ms: number;
}
