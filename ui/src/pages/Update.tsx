import { useState, useEffect } from 'preact/hooks';
import { Button } from '../components/Button';
import { Card } from '../components/Card';
import { LoadingSpinner } from '../components/LoadingSpinner';
import { Toast } from '../components/Toast';

interface UpdateInfo {
  current_version: string;
  latest_version: string;
  has_update: boolean;
  release?: {
    tag_name: string;
    name: string;
    body: string;
    published_at: string;
    assets: Array<{
      name: string;
      browser_download_url: string;
      size: number;
    }>;
  };
}

interface UpdateStatus {
  current_version: string;
  build_time: string;
  uptime: string;
  platform: string;
}

// Helper function to find the appropriate asset for the platform
const findAssetForPlatform = (assets: Array<{ name: string; browser_download_url: string; size: number }>, platform: string) => {
  // Extract OS and arch from platform string (e.g., "linux/amd64")
  const parts = platform.split('/');
  const os = parts[0];
  const arch = parts[1];
  
  if (!os || !arch) {
    return null;
  }
  
  // Look for exact platform match first
  for (const asset of assets) {
    const name = asset.name.toLowerCase();
    if (name.includes(os.toLowerCase()) && name.includes(arch.toLowerCase())) {
      return asset;
    }
  }
  
  // Look for OS match
  for (const asset of assets) {
    const name = asset.name.toLowerCase();
    if (name.includes(os.toLowerCase())) {
      return asset;
    }
  }
  
  // Look for any binary asset
  for (const asset of assets) {
    const name = asset.name.toLowerCase();
    if (name.includes('.tar.gz') || name.includes('.zip') || name.includes('.bin')) {
      return asset;
    }
  }
  
  return null;
};

interface UpdateProps {
  provider?: any; // Provider interface
}

const Update = ({ provider }: UpdateProps) => {
  const [updateInfo, setUpdateInfo] = useState<UpdateInfo | null>(null);
  const [updateStatus, setUpdateStatus] = useState<UpdateStatus | null>(null);
  const [loading, setLoading] = useState(false);
  const [downloading, setDownloading] = useState(false);
  const [installing, setInstalling] = useState(false);
  const [toast, setToast] = useState<{ message: string; type: 'success' | 'error' } | null>(null);
  const [downloadedPath, setDownloadedPath] = useState<string | null>(null);

  const showToast = (message: string, type: 'success' | 'error') => {
    setToast({ message, type });
    setTimeout(() => setToast(null), 5000);
  };

  // Subscribe to WebSocket update notifications
  useEffect(() => {
    if (!provider) return;

    const unsubscribe = provider.onUpdateAvailable((updateData: any) => {
      console.log('WebSocket update notification received:', updateData);
      setUpdateInfo(updateData);
      showToast(`Update available: ${updateData.latest_version}`, 'success');
    });

    return unsubscribe;
  }, [provider]);

  const checkForUpdates = async () => {
    setLoading(true);
    try {
      const response = await fetch('/api/v1/update/check');
      if (!response.ok) {
        throw new Error('Failed to check for updates');
      }
      const data = await response.json();
      setUpdateInfo(data);
    } catch (error) {
      showToast(`Failed to check for updates: ${error}`, 'error');
    } finally {
      setLoading(false);
    }
  };

  const downloadUpdate = async () => {
    if (!updateInfo?.release?.assets) return;

    // Get platform info from update status
    const platform = updateStatus?.platform || '';
    
    // Find the appropriate asset for current platform
    const asset = findAssetForPlatform(updateInfo.release.assets, platform);
    if (!asset) {
      showToast('No suitable update found for your platform', 'error');
      return;
    }

    setDownloading(true);
    try {
      const response = await fetch('/api/v1/update/download', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          download_url: asset.browser_download_url,
        }),
      });

      if (!response.ok) {
        throw new Error('Failed to download update');
      }

      const data = await response.json();
      setDownloadedPath(data.path);
      showToast('Update downloaded successfully', 'success');
    } catch (error) {
      showToast(`Failed to download update: ${error}`, 'error');
    } finally {
      setDownloading(false);
    }
  };

  const installUpdate = async () => {
    if (!downloadedPath) return;

    setInstalling(true);
    try {
      const response = await fetch('/api/v1/update/install', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          update_path: downloadedPath,
        }),
      });

      if (!response.ok) {
        throw new Error('Failed to install update');
      }

      const data = await response.json();
      showToast(data.message || 'Update installed successfully. Please restart the application.', 'success');
      setDownloadedPath(null);
    } catch (error) {
      showToast(`Failed to install update: ${error}`, 'error');
    } finally {
      setInstalling(false);
    }
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  const formatUptime = (uptimeString: string) => {
    // Parse uptime string like "29m4.372583295s" or "1h23m45s"
    const match = uptimeString.match(/(?:(\d+)h)?(?:(\d+)m)?(?:(\d+(?:\.\d+)?)s)?/);
    if (!match) return uptimeString;
    
    const hours = parseInt(match[1] || '0');
    const minutes = parseInt(match[2] || '0');
    const seconds = parseFloat(match[3] || '0');
    
    const parts = [];
    if (hours > 0) parts.push(`${hours}h`);
    if (minutes > 0) parts.push(`${minutes}m`);
    if (seconds > 0) parts.push(`${Math.floor(seconds)}s`);
    
    return parts.join(' ') || '0s';
  };

  useEffect(() => {
    // Load current status
    const loadStatus = async () => {
      try {
        const response = await fetch('/api/v1/update/status');
        if (response.ok) {
          const data = await response.json();
          setUpdateStatus(data);
        }
      } catch (error) {
        console.error('Failed to load update status:', error);
      }
    };

    loadStatus();
    checkForUpdates();
  }, []);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Updates</h1>
      </div>

      {toast && (
        <Toast
          message={toast.message}
          type={toast.type}
          onClose={() => setToast(null)}
        />
      )}

      {/* Current Status */}
      <Card>
        <div className="p-6">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-6">
            Current Version
          </h2>
          {updateStatus ? (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
              <div className="bg-gray-50 dark:bg-gray-800/50 rounded-lg p-4">
                <p className="text-sm font-medium text-gray-600 dark:text-gray-400 mb-1">Version</p>
                <p className="text-xl font-bold text-gray-900 dark:text-white">
                  {updateStatus.current_version}
                </p>
              </div>
              <div className="bg-gray-50 dark:bg-gray-800/50 rounded-lg p-4">
                <p className="text-sm font-medium text-gray-600 dark:text-gray-400 mb-1">Build Time</p>
                <p className="text-lg font-semibold text-gray-900 dark:text-white">
                  {formatDate(updateStatus.build_time)}
                </p>
              </div>
              <div className="bg-gray-50 dark:bg-gray-800/50 rounded-lg p-4">
                <p className="text-sm font-medium text-gray-600 dark:text-gray-400 mb-1">Uptime</p>
                <p className="text-lg font-semibold text-gray-900 dark:text-white">
                  {formatUptime(updateStatus.uptime)}
                </p>
              </div>
              <div className="bg-gray-50 dark:bg-gray-800/50 rounded-lg p-4">
                <p className="text-sm font-medium text-gray-600 dark:text-gray-400 mb-1">Platform</p>
                <p className="text-lg font-semibold text-gray-900 dark:text-white">
                  {updateStatus.platform}
                </p>
              </div>
            </div>
          ) : (
            <div className="flex items-center justify-center py-12">
              <LoadingSpinner />
            </div>
          )}
        </div>
      </Card>

      {/* Update Check */}
      <Card>
        <div className="p-6">
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
              Check for Updates
            </h2>
            <Button
              onClick={checkForUpdates}
              disabled={loading}
              className="flex items-center gap-2 w-full sm:w-auto"
            >
              {loading ? <LoadingSpinner size="sm" /> : null}
              Check for Updates
            </Button>
          </div>

          {updateInfo && (
            <div className="space-y-4">
              {updateInfo.has_update ? (
                <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-xl p-6">
                  <div className="flex items-center gap-3 mb-3">
                    <div className="w-3 h-3 bg-blue-500 rounded-full animate-pulse"></div>
                    <p className="text-blue-800 dark:text-blue-200 font-semibold text-lg">
                      Update Available
                    </p>
                  </div>
                  <p className="text-blue-700 dark:text-blue-300 text-base mb-2">
                    New version <span className="font-bold">{updateInfo.latest_version}</span> is available
                  </p>
                  <p className="text-blue-600 dark:text-blue-400 text-sm">
                    Current version: {updateInfo.current_version}
                  </p>
                  {updateInfo.release && (
                    <div className="mt-3 space-y-2">
                      <p className="text-sm text-blue-600 dark:text-blue-400">
                        Released: {formatDate(updateInfo.release.published_at)}
                      </p>
                      {updateInfo.release.body && (
                        <details className="mt-2">
                          <summary className="text-sm text-blue-600 dark:text-blue-400 cursor-pointer hover:underline">
                            Release Notes
                          </summary>
                          <div className="mt-2 p-3 bg-white dark:bg-gray-800 rounded border text-sm text-gray-700 dark:text-gray-300 whitespace-pre-wrap">
                            {updateInfo.release.body}
                          </div>
                        </details>
                      )}
                    </div>
                  )}
                </div>
              ) : (
                <div className="bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-xl p-6">
                  <div className="flex items-center gap-3 mb-3">
                    <div className="w-3 h-3 bg-green-500 rounded-full"></div>
                    <p className="text-green-800 dark:text-green-200 font-semibold text-lg">
                      You are up to date
                    </p>
                  </div>
                  <p className="text-green-700 dark:text-green-300 text-base">
                    Running latest version <span className="font-bold">{updateInfo.current_version}</span>
                  </p>
                </div>
              )}

              {updateInfo.has_update && (
                <div className="flex flex-col sm:flex-row gap-3 mt-6">
                  {!downloadedPath ? (
                    <Button
                      onClick={downloadUpdate}
                      disabled={downloading}
                      className="flex items-center justify-center gap-2 px-6 py-3 bg-blue-600 hover:bg-blue-700 text-white font-medium"
                    >
                      {downloading ? <LoadingSpinner size="sm" /> : null}
                      Download Update
                    </Button>
                  ) : (
                    <Button
                      onClick={installUpdate}
                      disabled={installing}
                      className="flex items-center justify-center gap-2 px-6 py-3 bg-red-600 hover:bg-red-700 text-white font-medium"
                    >
                      {installing ? <LoadingSpinner size="sm" /> : null}
                      Install Update
                    </Button>
                  )}
                </div>
              )}
            </div>
          )}
        </div>
      </Card>
    </div>
  );
};

export default Update;
