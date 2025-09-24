import { useState, useEffect } from 'preact/hooks';
import { Card } from '../components/Card.js';
import { StatsCard } from '../components/StatsCard.js';
import { Button } from '../components/Button.js';
// import { Input } from '../components/Input.js';
import { Chart } from '../components/Chart.js';
import { 
  useStats, useRuleGroups, useUpstreams, useStatsActions,
  useRuleGroupsActions, useUpstreamsActions,
  useServerInfo, useServerInfoActions
} from '../store/store.js';
import { FailoverProvider } from '../providers/failoverProvider.js';
// import { formatDuration } from '../utils/time.js';
import { CHART_WINDOWS, OverviewData } from '../providers/types.js';

interface OverviewProps {
  provider: FailoverProvider;
}

export function Overview({ provider }: OverviewProps) {
  const stats = useStats();
  const { setStats, setError: setStatsError } = useStatsActions();
  const ruleGroups = useRuleGroups();
  const upstreams = useUpstreams();
  const serverInfo = useServerInfo();
  const { setRuleGroups, setError: setRGError } = useRuleGroupsActions();
  const { setUpstreams, setError: setUSError } = useUpstreamsActions();
  const { setInfo, setError: setInfoErr } = useServerInfoActions();
  
  const [chartWindow, setChartWindow] = useState(CHART_WINDOWS[1]!); // 5m default
  // const [refreshInterval, setRefreshInterval] = useState(REFRESH_INTERVALS[1]!); // 5s default
  const [rpsSeries, setRpsSeries] = useState<number[]>([]);
  const [rtSeries, setRtSeries] = useState<number[]>([]);

  const navigate = (tab: 'overview' | 'rules' | 'upstreams') => {
    window.location.hash = tab;
    window.scrollTo(0, 0);
  };

  useEffect(() => {
    const unsubStats = provider.onStats(setStats);
    const unsubRG = provider.onRuleGroups(setRuleGroups);
    const unsubUS = provider.onUpstreams(setUpstreams);
    const unsubOV = (provider as any).onOverview ? (provider as any).onOverview((ov: OverviewData) => {
      if (ov?.stats) setStats(ov.stats);
      try { (window as any).__ov = ov; } catch {}
      setOverview(ov);
    }) : () => {};

    provider.fetchStats().then(setStats).catch((e) => setStatsError(e.message));
    provider.fetchRuleGroups().then(setRuleGroups).catch((e) => setRGError(e.message));
    provider.fetchUpstreams().then(setUpstreams).catch((e) => setUSError(e.message));
    provider.fetchServerInfo().then(setInfo).catch((e) => setInfoErr(e.message));
    if ((provider as any).fetchOverview) {
      (provider as any).fetchOverview().then((ov: OverviewData) => {
        if (ov?.stats) setStats(ov.stats);
        setOverview(ov);
        try { (window as any).__ov = ov; } catch {}
      }).catch(() => {});
    }

    return () => { unsubStats(); unsubRG(); unsubUS(); unsubOV(); };
  }, [provider, setStats, setRuleGroups, setUpstreams, setStatsError, setRGError, setUSError, setInfo, setInfoErr]);

  const [overview, setOverview] = useState<OverviewData | null>(null);

  // Build live series from stats stream
  useEffect(() => {
    if (!stats) return;
    const maxPoints = 30; // ~5m with ~10s update cadence
    const rps = stats.effective_rps ?? (stats.dns_queries_total ? stats.dns_queries_total / 60 : 0);
    setRpsSeries(prev => {
      const next = [...prev, Math.max(0, Number(rps || 0))];
      if (next.length > maxPoints) next.shift();
      return next;
    });
    const rt = (stats.dns_request_avg_seconds ?? 0) * 1000;
    setRtSeries(prev => {
      const next = [...prev, Math.max(0, Number(rt || 0))];
      if (next.length > maxPoints) next.shift();
      return next;
    });
  }, [stats]);

  // Calculate derived stats
  const dnsRPS = stats?.effective_rps ?? (stats ? stats.dns_queries_total / 60 : 0);
  const avgResponseTime = stats?.dns_request_avg_seconds ? stats.dns_request_avg_seconds * 1000 : 0;
  const activeUpstreams = upstreams ? upstreams.length : 0;
  const totalRules = ruleGroups ? ruleGroups.reduce((sum, group) => sum + group.patterns.length, 0) : 0;
  const cacheHit = (() => {
    if (!stats) return null;
    const r = stats.cache_hit_rate;
    // If undefined or there are no queries yet, treat as N/A
    if (r == null) return null;
    if ((stats.dns_queries_total || 0) === 0) return null;
    return r;
  })();

  // Live chart data
  const makeLabels = (len: number) => Array.from({ length: len }, () => '');
  const queriesData = {
    labels: makeLabels(rpsSeries.length),
    datasets: [{ label: 'RPS', data: rpsSeries, borderColor: '#3b82f6', backgroundColor: '#3b82f6', fill: false }],
  };
  const responseTimeData = {
    labels: makeLabels(rtSeries.length),
    datasets: [{ label: 'Avg RT (ms)', data: rtSeries, borderColor: '#10b981', backgroundColor: '#10b981', fill: false }],
  };


  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Dashboard</h1>
          <p className="text-gray-600 dark:text-gray-400 mt-1">
            Monitor your DNS proxy performance and configuration
          </p>
        </div>
        <div className="flex items-center space-x-4">
          <select
            value={(chartWindow as any).ms}
            onChange={(e) => {
              const window = CHART_WINDOWS.find(w => (w as any).ms === parseInt(e.currentTarget.value));
              if (window) setChartWindow(window);
            }}
            className="px-3 py-2 text-sm border border-gray-300 rounded-md bg-white text-gray-900 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-gray-800 dark:text-gray-100 dark:border-gray-600 dark:focus:border-blue-400 dark:focus:ring-blue-400"
          >
            {CHART_WINDOWS.map(window => (
              <option key={(window as any).ms} value={(window as any).ms}>
                {window.label}
              </option>
            ))}
          </select>
        </div>
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <StatsCard
          title="DNS RPS"
          value={dnsRPS.toFixed(1)}
          icon={
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
            </svg>
          }
          color="blue"
          loading={!stats}
        />
        
        <StatsCard
          title="Response Time"
          value={`${avgResponseTime.toFixed(0)}ms`}
          icon={
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
          }
          color="green"
          loading={!stats}
        />
        
        <StatsCard
          title="Active Upstreams"
          value={activeUpstreams}
          icon={
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
            </svg>
          }
          color="purple"
          loading={!upstreams}
        />
        
        <StatsCard
          title="Total Rules"
          value={totalRules}
          icon={
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
          }
          color="indigo"
          loading={!ruleGroups}
        />
      </div>

      {/* Charts Row */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Card title="DNS Queries Over Time" subtitle={`Last ${chartWindow.label}`}>
          <Chart
            data={queriesData}
            type="line"
            height={256}
            xLabel="time"
            yLabel="RPS"
            loading={!stats}
          />
        </Card>

        <Card title="Response Time Over Time" subtitle={`Last ${chartWindow.label}`}>
          <Chart
            data={responseTimeData}
            type="line"
            height={256}
            xLabel="time"
            yLabel="ms"
            loading={!stats}
          />
        </Card>
      </div>

      {/* System Status */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <Card title="Service Status" className="lg:col-span-2">
          <div className="space-y-4">
            <div className="flex items-center justify-between p-4 bg-gray-50 dark:bg-gray-800 rounded-lg">
              <div className="flex items-center">
                <div className={`w-3 h-3 rounded-full mr-3 ${
                  (stats?.service_ready === 1) ? 'bg-green-500' : 'bg-red-500'
                }`}></div>
                <div>
                  <p className="font-medium text-gray-900 dark:text-white">DNS Proxy</p>
                  <p className="text-sm text-gray-600 dark:text-gray-400">
                    {stats?.service_ready === 1 ? 'Running' : 'Stopped'}
                  </p>
                </div>
              </div>
              <div className="text-right">
                <p className="text-sm text-gray-600 dark:text-gray-400">Uptime</p>
                <p className="font-medium text-gray-900 dark:text-white">
                  {overview?.uptime ?? (serverInfo ? serverInfo.uptime : 'Unknown')}
                </p>
              </div>
            </div>
            
            <div className="grid grid-cols-2 gap-4">
              <div className="p-4 bg-gray-50 dark:bg-gray-800 rounded-lg">
                <p className="text-sm text-gray-600 dark:text-gray-400">Cache hit rate</p>
                <p className="text-lg font-semibold text-gray-900 dark:text-white">{cacheHit == null ? 'N/A' : `${Math.round(cacheHit * 100)}%`}</p>
              </div>
              <div className="p-4 bg-gray-50 dark:bg-gray-800 rounded-lg">
                <p className="text-sm text-gray-600 dark:text-gray-400">Queries (1m)</p>
                <p className="text-lg font-semibold text-gray-900 dark:text-white">{overview?.queries_last_min ?? 'N/A'}{overview?.errors_last_min ? ` (${overview?.errors_last_min} errors)` : ''}</p>
              </div>
            </div>
          </div>
        </Card>

        <Card title="Quick Actions">
          <div className="space-y-3">
            <Button 
              variant="outline" 
              fullWidth 
              className="justify-start hover:scale-[1.01] transition-transform"
              onClick={() => navigate('rules')}
            >
              <svg className="w-5 h-5 mr-3 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
              </svg>
              Add New Rule
            </Button>
            <Button 
              variant="outline" 
              fullWidth 
              className="justify-start hover:scale-[1.01] transition-transform"
              onClick={() => navigate('upstreams')}
            >
              <svg className="w-5 h-5 mr-3 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
              </svg>
              Add Upstream
            </Button>
          </div>
        </Card>

        {/* Resolve moved to its own page */}
      </div>
    </div>
  );
}