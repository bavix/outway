import { create } from 'zustand';
import { RuleGroup, Stats, QueryEvent, ServerInfo, UpstreamItem } from '../providers/types.js';

// Rule Groups slice
interface RuleGroupsState {
  ruleGroups: RuleGroup[];
  loading: boolean;
  error: string | null;
  setRuleGroups: (groups: RuleGroup[]) => void;
  addRuleGroup: (group: RuleGroup) => void;
  updateRuleGroup: (name: string, group: RuleGroup) => void;
  removeRuleGroup: (name: string) => void;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
}

const useRuleGroupsStore = create<RuleGroupsState>((set) => ({
  ruleGroups: [],
  loading: false,
  error: null,
  
  setRuleGroups: (ruleGroups) => set({ ruleGroups, error: null }),
  
  addRuleGroup: (group) => set((state) => ({
    ruleGroups: [...state.ruleGroups.filter(g => g.name !== group.name), group]
  })),
  
  updateRuleGroup: (name, group) => set((state) => ({
    ruleGroups: state.ruleGroups.map(g => g.name === name ? group : g)
  })),
  
  removeRuleGroup: (name) => set((state) => ({
    ruleGroups: state.ruleGroups.filter(g => g.name !== name)
  })),
  
  setLoading: (loading) => set({ loading }),
  
  setError: (error) => set({ error })
}));

// Upstreams slice
interface UpstreamsState {
  upstreams: UpstreamItem[];
  loading: boolean;
  error: string | null;
  setUpstreams: (upstreams: UpstreamItem[]) => void;
  addUpstream: (upstream: UpstreamItem) => void;
  removeUpstream: (address: string) => void;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
}

const useUpstreamsStore = create<UpstreamsState>((set) => ({
  upstreams: [],
  loading: false,
  error: null,
  
  setUpstreams: (upstreams) => set({ upstreams, error: null }),
  
  addUpstream: (upstream) => set((state) => ({
    upstreams: [...state.upstreams, upstream]
  })),
  
  removeUpstream: (address) => set((state) => ({
    upstreams: state.upstreams.filter(u => u.address !== address)
  })),
  
  setLoading: (loading) => set({ loading }),
  
  setError: (error) => set({ error })
}));

// Stats slice
interface StatsState {
  stats: Stats | null;
  loading: boolean;
  error: string | null;
  setStats: (stats: Stats) => void;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
}

const useStatsStore = create<StatsState>((set) => ({
  stats: null,
  loading: false,
  error: null,
  
  setStats: (stats) => set({ stats, error: null }),
  
  setLoading: (loading) => set({ loading }),
  
  setError: (error) => set({ error })
}));

// History slice
interface HistoryState {
  events: QueryEvent[];
  loading: boolean;
  error: string | null;
  setEvents: (events: QueryEvent[]) => void;
  addEvent: (event: QueryEvent) => void;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
}

// Server info slice
interface ServerInfoState {
  info: ServerInfo | null;
  loading: boolean;
  error: string | null;
  setInfo: (info: ServerInfo) => void;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
}

const useServerInfoStore = create<ServerInfoState>((set) => ({
  info: null,
  loading: false,
  error: null,
  
  setInfo: (info) => set({ info, error: null }),
  
  setLoading: (loading) => set({ loading }),
  
  setError: (error) => set({ error })
}));

const useHistoryStore = create<HistoryState>((set) => ({
  events: [],
  loading: false,
  error: null,
  
  setEvents: (events) => set({ events, error: null }),
  
  addEvent: (event) => set((state) => {
    // Keep only last 2000 events to prevent memory issues
    const newEvents = [event, ...state.events].slice(0, 2000);
    return { events: newEvents };
  }),
  
  setLoading: (loading) => set({ loading }),
  
  setError: (error) => set({ error })
}));

// Combined store hooks
export const useStore = () => ({
  ruleGroups: useRuleGroupsStore(),
  upstreams: useUpstreamsStore(),
  stats: useStatsStore(),
  history: useHistoryStore(),
});

// Selectors for optimized re-renders
export const useRuleGroups = () => useRuleGroupsStore((state) => state.ruleGroups);
export const useRuleGroupsActions = () => useRuleGroupsStore((state) => ({
  setRuleGroups: state.setRuleGroups,
  addRuleGroup: state.addRuleGroup,
  updateRuleGroup: state.updateRuleGroup,
  removeRuleGroup: state.removeRuleGroup,
  setLoading: state.setLoading,
  setError: state.setError
}));

export const useUpstreams = () => useUpstreamsStore((state) => state.upstreams);
export const useUpstreamsActions = () => useUpstreamsStore((state) => ({
  setUpstreams: state.setUpstreams,
  addUpstream: state.addUpstream,
  removeUpstream: state.removeUpstream,
  setLoading: state.setLoading,
  setError: state.setError
}));

export const useStats = () => useStatsStore((state) => state.stats);
export const useStatsActions = () => useStatsStore((state) => ({
  setStats: state.setStats,
  setLoading: state.setLoading,
  setError: state.setError
}));

export const useHistory = () => useHistoryStore((state) => state.events);
export const useHistoryActions = () => useHistoryStore((state) => ({
  setEvents: state.setEvents,
  addEvent: state.addEvent,
  setLoading: state.setLoading,
  setError: state.setError
}));

// Derived selectors
export const useReadinessStatus = () => useStatsStore((state) => {
  const stats = state.stats;
  if (!stats) return null;
  
  return {
    ready: stats.dns_queries_total > 0,
    queries: stats.dns_queries_total,
    rules: 0, // Will be computed from rules store
    upstreams: 0 // Will be computed from upstreams store
  };
});

export const useRPS = () => useStatsStore((_state) => {
  // RPS calculation would need historical data
  // For now, return null or a placeholder
  return null;
});

export const useServerInfo = () => useServerInfoStore((state) => state.info);
export const useServerInfoActions = () => useServerInfoStore((state) => ({
  setInfo: state.setInfo,
  setLoading: state.setLoading,
  setError: state.setError
}));
