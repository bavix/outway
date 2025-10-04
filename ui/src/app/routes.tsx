import { useState, useEffect } from 'preact/hooks';
import { FailoverProvider } from '../providers/failoverProvider.js';
import { Overview } from '../pages/Overview.js';
import { Resolve } from '../pages/Resolve.js';
import { Rules } from '../pages/Rules.js';
import { Upstreams } from '../pages/Upstreams.js';
import { History } from '../pages/History.js';
import { Hosts } from '../pages/Hosts.js';
import { Devices } from '../pages/Devices.js';
import Update from '../pages/Update.js';
import CachePage from '../pages/Cache.js';
import { Sidebar } from '../components/Sidebar.js';
import { Header } from '../components/Header.js';
import { useTheme } from '../hooks/useTheme.js';
import { ToastContainer } from '../components/Toast.js';

export function App() {
  // Initialize activeTab from hash if available, otherwise default to 'overview'
  const getInitialTab = () => {
    const hash = window.location.hash.slice(1);
    const validTabs = ['overview', 'rules', 'upstreams', 'history', 'hosts', 'devices', 'resolve', 'cache', 'update'];
    return validTabs.includes(hash) ? hash : 'overview';
  };
  
  const [activeTab, setActiveTab] = useState(getInitialTab());
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [provider, setProvider] = useState<FailoverProvider | null>(null);
  const [toasts, setToasts] = useState<Array<{ id: string; message: string; type?: 'success' | 'error' | 'warning' | 'info'; duration?: number }>>([]);
  useTheme();

  useEffect(() => {
    // Initialize provider
    const failoverProvider = new FailoverProvider('/ws');
    let restToastTimer: number | null = null;
    failoverProvider.connect()
      .then(() => {
        setProvider(failoverProvider);
        console.log('Provider connected');
      })
      .catch((error) => {
        console.error('Failed to connect provider:', error);
      });

    // Handle hash changes for routing
    const handleHashChange = () => {
      const hash = window.location.hash.slice(1);
      const validTabs = ['overview', 'rules', 'upstreams', 'history', 'hosts', 'devices', 'resolve', 'cache', 'update'];
      if (validTabs.includes(hash)) {
        setActiveTab(hash);
      }
    };

    // Set initial tab from hash
    handleHashChange();
    window.addEventListener('hashchange', handleHashChange);

    // Handle responsive sidebar
    const handleResize = () => {
      if (window.innerWidth < 1024) {
        setSidebarCollapsed(true);
      }
    };

    handleResize();
    window.addEventListener('resize', handleResize);

    // Status toasts
    const offStatus = failoverProvider.onStatus?.((s) => {
      // Debounce REST warning; cancel if WS becomes active soon after
      if (s.channel === 'rest') {
        if (restToastTimer) clearTimeout(restToastTimer);
        restToastTimer = window.setTimeout(() => {
          setToasts((prev) => [{ id: `${Date.now()}`, message: 'WebSocket unavailable — using polling', type: 'warning' as const, duration: 2500 }, ...prev].slice(0, 4));
          restToastTimer = null;
        }, 1200);
        return;
      }
      // WS active
      if (restToastTimer) {
        clearTimeout(restToastTimer);
        restToastTimer = null;
      }
      setToasts((prev) => [{ id: `${Date.now()}`, message: 'Realtime updates via WebSocket', type: 'success' as const, duration: 2500 }, ...prev].slice(0, 4));
    });

    return () => {
      window.removeEventListener('hashchange', handleHashChange);
      window.removeEventListener('resize', handleResize);
      if (restToastTimer) clearTimeout(restToastTimer);
      offStatus && offStatus();
      failoverProvider.close();
    };
  }, []);

  const handleTabChange = (tabId: string) => {
    // Update hash and rely on the hashchange listener to set active tab.
    // This avoids double state updates and extra re-renders causing lag.
    if (window.location.hash.slice(1) !== tabId) {
      window.location.hash = tabId;
    } else {
      // If already on the same hash, manually sync state
      setActiveTab(tabId);
    }
    // On mobile, hide the sidebar after selecting a menu item
    if (window.innerWidth < 1024) {
      setSidebarCollapsed(true);
    }
    window.scrollTo(0, 0);
  };

  const handleSidebarToggle = () => {
    setSidebarCollapsed(!sidebarCollapsed);
  };

  if (!provider) {
    return (
      <div className="min-h-screen bg-gray-50 dark:bg-gray-900 flex items-center justify-center">
        <div className="text-center">
          <div className="w-16 h-16 bg-gray-200 dark:bg-gray-700 rounded-full animate-pulse mx-auto mb-4"></div>
          <p className="text-gray-600 dark:text-gray-400">Connecting to server...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-800 dark:to-gray-700">
      <ToastContainer toasts={toasts} onRemove={(id) => setToasts((prev) => prev.filter((t) => t.id !== id))} />
      {/* Sidebar */}
      <Sidebar
        activeTab={activeTab}
        onTabChange={handleTabChange}
        collapsed={sidebarCollapsed}
      />

      {/* Main content */}
      <div className={`transition-all duration-300 ${
        // On large screens, respect collapsed width; on mobile, take full width
        sidebarCollapsed ? 'lg:ml-16' : 'lg:ml-64'
      }`}>
        {/* Header */}
        <Header
          onSidebarToggle={handleSidebarToggle}
          sidebarCollapsed={sidebarCollapsed}
          provider={provider}
        />

        {/* Page content */}
        <main className="p-6">
          <div className="max-w-7xl mx-auto">
            {activeTab === 'overview' && <Overview provider={provider} />}
            {activeTab === 'rules' && <Rules provider={provider} />}
            {activeTab === 'upstreams' && <Upstreams provider={provider} />}
            {activeTab === 'history' && <History provider={provider} />}
            {activeTab === 'hosts' && <Hosts provider={provider} />}
            {activeTab === 'devices' && <Devices provider={provider} />}
            {activeTab === 'resolve' && <Resolve provider={provider} />}
            {activeTab === 'cache' && <CachePage provider={provider} />}
            {activeTab === 'update' && <Update provider={provider} />}
          </div>
        </main>
      </div>

      {/* Mobile sidebar overlay */}
      {!sidebarCollapsed && (
        <div
          className="fixed inset-0 z-30 bg-black/50 lg:hidden"
          onClick={handleSidebarToggle}
        />
      )}
    </div>
  );
}
