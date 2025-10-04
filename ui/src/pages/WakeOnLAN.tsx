import { useState, useEffect } from 'preact/hooks';
import { Card } from '../components/Card.js';
import { Button } from '../components/Button.js';
import { Input } from '../components/Input.js';
import { Badge } from '../components/Badge.js';
import { Table, THead, TBody, TRow, TH, TD } from '../components/Table.js';
import { Modal } from '../components/Modal.js';
import { FailoverProvider } from '../providers/failoverProvider.js';
import { Device, DeviceWakeRequest, DeviceWakeResponse } from '../providers/types.js';

interface WakeOnLANProps {
  provider: FailoverProvider;
}

export function WakeOnLAN({ provider }: WakeOnLANProps) {
  const [devices, setDevices] = useState<Device[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showAddModal, setShowAddModal] = useState(false);
  const [showWakeModal, setShowWakeModal] = useState(false);
  const [selectedDevice, setSelectedDevice] = useState<Device | null>(null);
  const [wakeStatus, setWakeStatus] = useState<'idle' | 'waking' | 'success' | 'error'>('idle');

  // Form states
  const [newDevice, setNewDevice] = useState({
    name: '',
    mac: '',
    ip: '',
  });

  // Load devices from API
  useEffect(() => {
    const loadDevices = async () => {
      try {
        setLoading(true);
        if (provider.fetchDevices) {
          const response = await provider.fetchDevices();
          setDevices(response.devices || []);
        }
      } catch (err) {
        setError('Failed to load devices');
        console.error('Failed to load devices:', err);
      } finally {
        setLoading(false);
      }
    };

    loadDevices();
  }, [provider]);

  const handleWakeDevice = async (device: Device) => {
    setSelectedDevice(device);
    setShowWakeModal(true);
    setWakeStatus('waking');

    try {
      if (!provider.wakeDevice) {
        throw new Error('Wake device API not available');
      }

      const request: DeviceWakeRequest = {
        id: device.id,
      };

      const response: DeviceWakeResponse = await provider.wakeDevice(request);
      setWakeStatus(response.success ? 'success' : 'error');
      
      if (response.success) {
        // Update device status
        setDevices(prev => prev.map(d => 
          d.id === device.id 
            ? { ...d, status: 'online' as const, last_seen: new Date().toISOString() }
            : d
        ));
      }
    } catch (err) {
      setWakeStatus('error');
      console.error('Failed to wake device:', err);
    }
  };

  const handleAddDevice = async () => {
    if (!newDevice.name || !newDevice.mac) {
      setError('Name and MAC address are required');
      return;
    }

    // Validate MAC address format
    const macRegex = /^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$/;
    if (!macRegex.test(newDevice.mac)) {
      setError('Invalid MAC address format');
      return;
    }

    try {
      if (!provider.addDevice) {
        throw new Error('Add device API not available');
      }

      const device = await provider.addDevice({
        name: newDevice.name,
        mac: newDevice.mac.toUpperCase(),
        ip: newDevice.ip || 'Unknown',
        hostname: newDevice.name,
      });

      setDevices(prev => [...prev, device]);
      setNewDevice({ name: '', mac: '', ip: '' });
      setShowAddModal(false);
      setError(null);
    } catch (err) {
      setError('Failed to add device');
      console.error('Failed to add device:', err);
    }
  };

  const handleScanNetwork = async () => {
    setLoading(true);
    setError(null);

    try {
      if (provider.scanDevices) {
        const response = await provider.scanDevices();
        setDevices(response.devices || []);
      } else {
        // Fallback: reload devices from API
        if (provider.fetchDevices) {
          const response = await provider.fetchDevices();
          setDevices(response.devices || []);
        }
      }
    } catch (err) {
      setError('Failed to scan network');
      console.error('Failed to scan network:', err);
    } finally {
      setLoading(false);
    }
  };

  const getStatusVariant = (status: Device['status']) => {
    switch (status) {
      case 'online': return 'primary';
      case 'offline': return 'secondary';
      case 'unknown': return 'default';
      default: return 'default';
    }
  };

  const getStatusText = (status: Device['status']) => {
    switch (status) {
      case 'online': return 'Online';
      case 'offline': return 'Offline';
      case 'unknown': return 'Unknown';
      default: return 'Unknown';
    }
  };

  const formatLastSeen = (lastSeen: string) => {
    const date = new Date(lastSeen);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMins / 60);
    const diffDays = Math.floor(diffHours / 24);

    if (diffMins < 1) return 'Just now';
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;
    return `${diffDays}d ago`;
  };


  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Wake-on-LAN</h1>
          <p className="text-gray-600 dark:text-gray-400 mt-1">
            Manage and wake up devices on your network
          </p>
        </div>
        <div className="flex items-center space-x-3">
          <Button
            variant="outline"
            onClick={handleScanNetwork}
            loading={loading}
          >
            <svg className="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
            </svg>
            Scan Network
          </Button>
          <Button onClick={() => setShowAddModal(true)}>
            <svg className="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
            </svg>
            Add Device
          </Button>
        </div>
      </div>

      {/* Error Message */}
      {error && (
        <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
          <div className="flex">
            <svg className="w-5 h-5 text-red-400 mr-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <div>
              <h3 className="text-sm font-medium text-red-800 dark:text-red-200">Error</h3>
              <p className="text-sm text-red-700 dark:text-red-300 mt-1">{error}</p>
            </div>
          </div>
        </div>
      )}

      {/* Statistics Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <Card>
          <div className="flex items-center">
            <div className="p-3 bg-green-100 dark:bg-green-900/20 rounded-lg">
              <svg className="w-6 h-6 text-green-600 dark:text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
            </div>
            <div className="ml-4">
              <p className="text-sm font-medium text-gray-600 dark:text-gray-400">Online Devices</p>
              <p className="text-2xl font-bold text-gray-900 dark:text-white">
                {devices.filter(d => d.status === 'online').length}
              </p>
            </div>
          </div>
        </Card>

        <Card>
          <div className="flex items-center">
            <div className="p-3 bg-red-100 dark:bg-red-900/20 rounded-lg">
              <svg className="w-6 h-6 text-red-600 dark:text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
            </div>
            <div className="ml-4">
              <p className="text-sm font-medium text-gray-600 dark:text-gray-400">Offline Devices</p>
              <p className="text-2xl font-bold text-gray-900 dark:text-white">
                {devices.filter(d => d.status === 'offline').length}
              </p>
            </div>
          </div>
        </Card>

        <Card>
          <div className="flex items-center">
            <div className="p-3 bg-blue-100 dark:bg-blue-900/20 rounded-lg">
              <svg className="w-6 h-6 text-blue-600 dark:text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
              </svg>
            </div>
            <div className="ml-4">
              <p className="text-sm font-medium text-gray-600 dark:text-gray-400">Total Devices</p>
              <p className="text-2xl font-bold text-gray-900 dark:text-white">{devices.length}</p>
            </div>
          </div>
        </Card>
      </div>

      {/* Devices Table */}
      <Card title="Network Devices" subtitle="Manage and wake up devices on your network">
        <Table>
          <THead>
            <TRow>
              <TH>Device Name</TH>
              <TH>MAC Address</TH>
              <TH>IP Address</TH>
              <TH>Status</TH>
              <TH>Last Seen</TH>
              <TH>Vendor</TH>
              <TH>Actions</TH>
            </TRow>
          </THead>
          <TBody>
            {devices.length === 0 ? (
              <TRow>
                <TD colSpan={7}>
                  <div className="text-center py-8 text-gray-500 dark:text-gray-400">
                    {loading ? 'Loading...' : 'No devices found. Try scanning the network or adding a device manually.'}
                  </div>
                </TD>
              </TRow>
            ) : (
              devices.map(device => (
                <TRow key={device.id}>
                  <TD><span className="font-medium">{device.name}</span></TD>
                  <TD><span className="font-mono text-sm">{device.mac}</span></TD>
                  <TD>{device.ip}</TD>
                  <TD>
                    <Badge variant={getStatusVariant(device.status)}>
                      {getStatusText(device.status)}
                    </Badge>
                  </TD>
                  <TD>{formatLastSeen(device.last_seen)}</TD>
                  <TD>{device.vendor || 'Unknown'}</TD>
                  <TD>
                    <div className="flex space-x-2">
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => handleWakeDevice(device)}
                        disabled={device.status === 'online'}
                      >
                        <svg className="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
                        </svg>
                        Wake
                      </Button>
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={async () => {
                          try {
                            if (provider.deleteDevice) {
                              await provider.deleteDevice(device.id);
                            }
                            setDevices(prev => prev.filter(d => d.id !== device.id));
                          } catch (err) {
                            setError('Failed to delete device');
                            console.error('Failed to delete device:', err);
                          }
                        }}
                      >
                        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                        </svg>
                      </Button>
                    </div>
                  </TD>
                </TRow>
              ))
            )}
          </TBody>
        </Table>
      </Card>

      {/* Add Device Modal */}
      <Modal
        isOpen={showAddModal}
        onClose={() => {
          setShowAddModal(false);
          setError(null);
        }}
        title="Add New Device"
      >
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Device Name
            </label>
            <Input
              value={newDevice.name}
              onInput={(e) => setNewDevice(prev => ({ ...prev, name: (e.target as HTMLInputElement).value }))}
              placeholder="e.g., Desktop PC"
            />
          </div>
          
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              MAC Address
            </label>
            <Input
              value={newDevice.mac}
              onInput={(e) => setNewDevice(prev => ({ ...prev, mac: (e.target as HTMLInputElement).value }))}
              placeholder="e.g., 00:1B:44:11:3A:B7"
            />
          </div>
          
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              IP Address (Optional)
            </label>
            <Input
              value={newDevice.ip}
              onInput={(e) => setNewDevice(prev => ({ ...prev, ip: (e.target as HTMLInputElement).value }))}
              placeholder="e.g., 192.168.1.100"
            />
          </div>
        </div>
        
        <div className="flex justify-end space-x-3 mt-6">
          <Button
            variant="outline"
            onClick={() => {
              setShowAddModal(false);
              setError(null);
            }}
          >
            Cancel
          </Button>
          <Button onClick={handleAddDevice}>
            Add Device
          </Button>
        </div>
      </Modal>

      {/* Wake Device Modal */}
      <Modal
        isOpen={showWakeModal}
        onClose={() => {
          setShowWakeModal(false);
          setWakeStatus('idle');
        }}
        title="Wake Up Device"
      >
        <div className="space-y-4">
            <div className="text-center">
              <div className="w-16 h-16 bg-blue-100 dark:bg-blue-900/20 rounded-full flex items-center justify-center mx-auto mb-4">
                <svg className="w-8 h-8 text-blue-600 dark:text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
                </svg>
              </div>
              <h3 className="text-lg font-medium text-gray-900 dark:text-white">
                {selectedDevice?.name || 'Unknown Device'}
              </h3>
              <p className="text-sm text-gray-600 dark:text-gray-400">
                {selectedDevice?.mac || 'Unknown MAC'}
              </p>
            </div>

            {wakeStatus === 'waking' && (
              <div className="text-center">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto mb-2"></div>
                <p className="text-sm text-gray-600 dark:text-gray-400">
                  Sending wake-up packet...
                </p>
              </div>
            )}

            {wakeStatus === 'success' && (
              <div className="text-center">
                <div className="w-8 h-8 bg-green-100 dark:bg-green-900/20 rounded-full flex items-center justify-center mx-auto mb-2">
                  <svg className="w-5 h-5 text-green-600 dark:text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                  </svg>
                </div>
                <p className="text-sm text-green-600 dark:text-green-400">
                  Wake-up packet sent successfully!
                </p>
              </div>
            )}

            {wakeStatus === 'error' && (
              <div className="text-center">
                <div className="w-8 h-8 bg-red-100 dark:bg-red-900/20 rounded-full flex items-center justify-center mx-auto mb-2">
                  <svg className="w-5 h-5 text-red-600 dark:text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </div>
                <p className="text-sm text-red-600 dark:text-red-400">
                  Failed to send wake-up packet
                </p>
              </div>
            )}

            <div className="flex justify-end space-x-3 mt-6">
              <Button
                variant="outline"
                onClick={() => {
                  setShowWakeModal(false);
                  setWakeStatus('idle');
                }}
              >
                Close
              </Button>
              {wakeStatus === 'idle' && selectedDevice && (
                <Button onClick={() => handleWakeDevice(selectedDevice)}>
                  Send Wake Packet
                </Button>
              )}
            </div>
          </div>
      </Modal>
    </div>
  );
}
