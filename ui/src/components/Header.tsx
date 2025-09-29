import { useEffect, useState } from 'preact/hooks';
import { ThemeToggle } from './ThemeToggle';
import { FailoverProvider } from '../providers/failoverProvider.js';

interface HeaderProps {
  onSidebarToggle?: () => void;
  sidebarCollapsed?: boolean;
  provider?: FailoverProvider;
}

export function Header({ onSidebarToggle, provider }: HeaderProps) {
  const [channel, setChannel] = useState<'ws' | 'rest'>('ws');
  useEffect(() => {
    if (!provider || !provider.onStatus) return;
    const off = provider.onStatus!(s => setChannel(s.channel));
    return () => { off && off(); };
  }, [provider]);
  return (
    <header className="sticky top-0 z-30 bg-white/80 dark:bg-gray-800/80 backdrop-blur-sm border-b border-gray-100 dark:border-gray-700">
      <div className="flex items-center justify-between h-14 px-4">
        {/* Left side */}
        <div className="flex items-center gap-3">
          {onSidebarToggle && (
            <button
              onClick={onSidebarToggle}
              className="p-1.5 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-500 dark:text-gray-300 transition-all duration-200"
              aria-label="Toggle sidebar"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M4 6h16M4 12h16M4 18h16" />
              </svg>
            </button>
          )}
          <div className="flex items-center lg:hidden select-none">
            <div className="w-6 h-6 bg-gradient-to-br from-blue-500 to-blue-600 rounded flex items-center justify-center shadow-sm mr-2">
              <span className="text-white font-bold text-xs">O</span>
            </div>
            <span className="text-sm font-semibold text-gray-900 dark:text-white tracking-tight">Outway</span>
          </div>
        </div>

        {/* Right side */}
        <div className="flex items-center gap-2">
          <span className={`px-2 py-0.5 text-xs rounded-full ${channel === 'ws' ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300' : 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-300'}`}
                title={channel === 'ws' ? 'WebSocket live' : 'REST fallback'}>
            {channel.toUpperCase()}
          </span>
          {provider?.reconnect && channel === 'rest' && (
            <button
              className="inline-flex items-center gap-1 text-xs px-2 py-1 rounded-md border border-gray-300 dark:border-gray-600/50 text-gray-700 dark:text-gray-200 bg-transparent hover:bg-gray-50 dark:hover:bg-gray-700/50"
              title="Try reconnect WebSocket"
              onClick={() => provider.reconnect && provider.reconnect()}
            >
              <svg className="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                <path fillRule="evenodd" d="M4.083 9.75A6.75 6.75 0 0117.5 10a.75.75 0 001.5 0A8.25 8.25 0 103.5 15.398V13a.75.75 0 00-1.5 0v4a.75.75 0 00.75.75H7a.75.75 0 000-1.5H4.84A6.73 6.73 0 014.083 9.75z" clipRule="evenodd" />
              </svg>
              Reconnect
            </button>
          )}
          {/* Theme Toggle */}
          <div className="mx-1">
            <ThemeToggle variant="button" />
          </div>
        </div>
      </div>
    </header>
  );
}