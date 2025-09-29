import { useState, useEffect, useMemo } from 'preact/hooks';
import { Card, Input } from '../components/index.js';
import { useHistory, useHistoryActions } from '../store/store.js';
import { FailoverProvider } from '../providers/failoverProvider.js';
// import type { QueryEvent } from '../providers/types.js';
import { formatQType, formatUpstream, formatTimestamp } from '../utils/format.js';

interface HistoryProps {
  provider: FailoverProvider;
}

export function History({ provider }: HistoryProps) {
  const events = useHistory();
  const { setEvents, setError } = useHistoryActions();
  
  const [filter, setFilter] = useState('');
  const [maxRows, setMaxRows] = useState(200);

  // Subscribe to history updates
  useEffect(() => {
    const unsubscribe = provider.onHistory((newEvents) => {
      setEvents(newEvents);
    });

    // Load initial history
    provider.fetchHistory()
      .then(setEvents)
      .catch((error) => {
        console.error('Failed to fetch history:', error);
        setError(error.message);
      });

    return unsubscribe;
  }, [provider, setEvents, setError]);

  // Filter and limit events for display
  const displayEvents = useMemo(() => {
    let filtered = events;
    
    // Filter by name if specified
    if (filter.trim()) {
      const filterLower = filter.toLowerCase();
      filtered = events.filter(event => 
        event.name.toLowerCase().includes(filterLower)
      );
    }
    
    // Limit to maxRows for performance
    return filtered.slice(0, maxRows);
  }, [events, filter, maxRows]);

  // removed unused helper

  const renderPagination = () => {
    if (events.length <= maxRows) return <></>;
    
    return (
      <div style={{ marginTop: '12px', textAlign: 'center' }}>
        <span className="muted">
          Showing {maxRows} of {events.length} events. 
        </span>
        {maxRows < 1000 && (
          <button
            className="ow-btn ow-btn--secondary ow-btn--sm"
            style={{ marginLeft: '8px' }}
            onClick={() => setMaxRows(Math.min(maxRows * 2, 2000))}
          >
            Show More
          </button>
        )}
      </div>
    );
  };

  // render directly

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Query History</h1>
          <p className="text-gray-600 dark:text-gray-400 mt-1">
            View DNS query history and performance metrics
          </p>
        </div>
      </div>

      {/* Search and Filters */}
      <Card title="Search and Filters" subtitle="Filter DNS query history">
        <div className="space-y-4">
          <Input
            label="Filter by Domain"
            value={filter}
            onInput={(e) => setFilter((e.target as HTMLInputElement).value)}
            placeholder="Filter by domain name..."
            hint="Enter a domain name to filter the results"
          />
          
          <div className="flex items-center justify-between">
            <div className="text-sm text-gray-600 dark:text-gray-400">
              Showing {displayEvents.length} of {events.length} events
            </div>
            {events.length > maxRows && (
              <button
                className="text-sm text-primary-600 dark:text-primary-400 hover:text-primary-700 dark:hover:text-primary-300"
                onClick={() => setMaxRows(Math.min(maxRows * 2, 2000))}
              >
                Show More
              </button>
            )}
          </div>
        </div>
      </Card>

      {/* History Table */}
      <Card title="Query History" subtitle={`${displayEvents.length} DNS queries`}>

        {displayEvents.length === 0 ? (
          <div className="text-center py-8">
            <p className="text-gray-600 dark:text-gray-400">
              {events.length === 0 ? 'No query history available' : 'No queries match filter'}
            </p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700 table-auto">
              <thead className="bg-gray-50 dark:bg-gray-800">
                <tr>
                  <th className="px-4 sm:px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Time</th>
                  <th className="px-4 sm:px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Domain</th>
                  <th className="px-4 sm:px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Type</th>
                  <th className="px-4 sm:px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Client IP</th>
                  <th className="px-4 sm:px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Upstream</th>
                  <th className="px-4 sm:px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Duration</th>
                  <th className="px-4 sm:px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Status</th>
                </tr>
              </thead>
              <tbody className="bg-white dark:bg-gray-900 divide-y divide-gray-200 dark:divide-gray-700">
                {displayEvents.map((event, index) => (
                  <tr key={`${event.time}-${event.name}-${index}`} className="hover:bg-gray-50 dark:hover:bg-gray-800">
                    <td className="px-4 sm:px-6 py-3 sm:py-4 whitespace-nowrap text-sm text-gray-900 dark:text-gray-100">
                      {formatTimestamp(event.time)}
                    </td>
                    <td className="px-4 sm:px-6 py-3 sm:py-4 whitespace-normal sm:whitespace-nowrap break-words text-sm font-medium text-gray-900 dark:text-gray-100">
                      {event.name}
                    </td>
                    <td className="px-4 sm:px-6 py-3 sm:py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                      {formatQType(event.qtype)}
                    </td>
                    <td className="px-4 sm:px-6 py-3 sm:py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400 font-mono">
                      {event.client_ip || 'unknown'}
                    </td>
                    <td className="px-4 sm:px-6 py-3 sm:py-4 whitespace-normal sm:whitespace-nowrap break-words text-sm text-gray-500 dark:text-gray-400" title={event.upstream}>
                      {formatUpstream(event.upstream)}
                    </td>
                    <td className="px-4 sm:px-6 py-3 sm:py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                      {event.duration}
                    </td>
                    <td className="px-4 sm:px-6 py-3 sm:py-4 whitespace-nowrap">
                      <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                        event.status === 'ok' 
                          ? 'bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400' 
                          : 'bg-red-100 text-red-800 dark:bg-red-900/20 dark:text-red-400'
                      }`}>
                        {event.status === 'ok' ? 'OK' : 'Error'}
                      </span>
                    </td>
                  </tr>
                ))}
            </tbody>
          </table>
          {renderPagination()}
        </div>
        )}
      </Card>
    </div>
  );
}
