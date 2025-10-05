import { useState, useEffect } from 'preact/hooks';
import { FailoverProvider } from '../providers/failoverProvider.js';
import { Overview } from '../pages/Overview.js';
import { Resolve } from '../pages/Resolve.js';
import { Rules } from '../pages/Rules.js';
import { Upstreams } from '../pages/Upstreams.js';
import { History } from '../pages/History.js';
import { Hosts } from '../pages/Hosts.js';
import { Devices } from '../pages/Devices.js';
import { UserManagement } from '../components/UserManagement.js';
import { RolesOverview } from '../components/RolesOverview.js';
import Update from '../pages/Update.js';
import CachePage from '../pages/Cache.js';
import { Sidebar } from '../components/Sidebar.js';
import { Header } from '../components/Header.js';
import { useTheme } from '../hooks/useTheme.js';
import { ToastContainer } from '../components/Toast.js';
import { authService, AuthState } from '../services/authService.js';
import { LoginForm } from '../components/LoginForm.js';
import { LoadingSpinner } from '../components/LoadingSpinner.js';

export function App() {
  // Initialize activeTab from hash if available, otherwise default to 'overview'
  const getInitialTab = () => {
    const hash = window.location.hash.slice(1);
    const validTabs = ['overview', 'rules', 'upstreams', 'history', 'hosts', 'devices', 'resolve', 'cache', 'users', 'roles', 'update'];
    return validTabs.includes(hash) ? hash : 'overview';
  };
  
  const [activeTab, setActiveTab] = useState(getInitialTab());
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [provider, setProvider] = useState<FailoverProvider | null>(null);
  const [toasts, setToasts] = useState<Array<{ id: string; message: string; type?: 'success' | 'error' | 'warning' | 'info'; duration?: number }>>([]);
  const [authState, setAuthState] = useState<AuthState>(authService.state);
  const [authStatus, setAuthStatus] = useState<{ usersExist: boolean } | null>(null);
  const [isCheckingAuth, setIsCheckingAuth] = useState(true);
  useTheme();

  useEffect(() => {
    // Subscribe to auth state changes
    const unsubscribeAuth = authService.subscribe(setAuthState);

    return () => {
      unsubscribeAuth();
    };
  }, []);

  useEffect(() => {
    // Check authentication status
    const checkAuthStatus = async () => {
      try {
        const status = await authService.getAuthStatus();
        setAuthStatus(status);
        authService.setAuthStatus(status);
      } catch (error) {
        console.error('Failed to check auth status:', error);
        // Fallback: assume users exist
        const fallbackStatus = { usersExist: true };
        setAuthStatus(fallbackStatus);
        authService.setAuthStatus(fallbackStatus);
      } finally {
        setIsCheckingAuth(false);
      }
    };

    checkAuthStatus();
  }, [authState.isAuthenticated]);

  useEffect(() => {
    // Initialize provider only if authenticated
    if (!authState.isAuthenticated) {
      if (provider) {
        provider.close();
        setProvider(null);
      }
      return;
    }

    const failoverProvider = new FailoverProvider(`${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws`);
    let restToastTimer: number | null = null;
    
    failoverProvider.connect()
      .then(() => {
        setProvider(failoverProvider);
        authService.setProvider(failoverProvider);
        console.log('Provider connected');
        
        // Check auth status now that provider is available
        authService.getAuthStatus()
          .then((status) => {
            setAuthStatus(status);
            authService.setAuthStatus(status);
          })
          .catch((error) => {
            console.error('Failed to check auth status after provider connection:', error);
          });
      })
      .catch((error) => {
        console.error('Failed to connect provider:', error);
      });

    // Handle hash changes for routing
    const handleHashChange = () => {
      const hash = window.location.hash.slice(1);
      const validTabs = ['overview', 'rules', 'upstreams', 'history', 'hosts', 'devices', 'resolve', 'cache', 'users', 'roles', 'update'];
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
  }, [authState.isAuthenticated]);

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

  // Show login form if not authenticated
  if (!authState.isAuthenticated) {
    if (isCheckingAuth) {
      return <LoadingSpinner text="Checking authentication status..." className="min-h-screen" />;
    }

    // If we don't have auth status yet, show loading
    if (!authStatus) {
      return <LoadingSpinner text="Loading authentication status..." className="min-h-screen" />;
    }

    return (
      <LoginForm
        onLogin={() => setActiveTab('overview')}
        isFirstUser={!authStatus.usersExist}
      />
    );
  }

  if (!provider) {
    return <LoadingSpinner text="Connecting to server..." className="min-h-screen" />;
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
        sidebarCollapsed ? 'ml-0 lg:ml-16' : 'ml-0 lg:ml-64'
      }`}>
        {/* Header */}
        <Header
          onSidebarToggle={handleSidebarToggle}
          sidebarCollapsed={sidebarCollapsed}
          provider={provider}
        />

        {/* Page content */}
        <main className="p-6 pt-24">
          <div className="max-w-7xl mx-auto">
            {activeTab === 'overview' && <Overview provider={provider} />}
            {activeTab === 'rules' && <Rules provider={provider} />}
            {activeTab === 'upstreams' && <Upstreams provider={provider} />}
            {activeTab === 'history' && <History provider={provider} />}
            {activeTab === 'hosts' && <Hosts provider={provider} />}
            {activeTab === 'devices' && <Devices provider={provider} />}
            {activeTab === 'resolve' && <Resolve provider={provider} />}
            {activeTab === 'cache' && <CachePage provider={provider} />}
            {activeTab === 'users' && <UserManagement />}
            {activeTab === 'roles' && <RolesOverview />}
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
