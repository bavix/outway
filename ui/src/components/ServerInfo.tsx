import { useEffect } from 'preact/hooks';
import { Card } from './Card';
import { useServerInfo, useServerInfoActions } from '../store/store';
import { FailoverProvider } from '../providers/failoverProvider';

interface ServerInfoProps {
  provider: FailoverProvider;
}

export function ServerInfo({ provider }: ServerInfoProps) {
  const info = useServerInfo();
  const { setInfo, setLoading, setError } = useServerInfoActions();

  useEffect(() => {
    const loadInfo = async () => {
      setLoading(true);
      try {
        const serverInfo = await provider.fetchServerInfo();
        setInfo(serverInfo);
      } catch (error) {
        console.error('Failed to load server info:', error);
        setError(error instanceof Error ? error.message : 'Failed to load server info');
      } finally {
        setLoading(false);
      }
    };

    loadInfo();
  }, [provider, setInfo, setLoading, setError]);

  if (!info) {
    return (
      <Card title="Server Information">
        <div className="skeleton" style={{ width: '100%', height: '120px' }}></div>
      </Card>
    );
  }

  return (
    <Card title="Server Information">
      <div className="ow-grid" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))' }}>
        <div>
          <div className="muted" style={{ fontSize: 'var(--font-size-sm)' }}>Version</div>
          <div style={{ fontWeight: '600' }}>{info.version}</div>
        </div>
        
        <div>
          <div className="muted" style={{ fontSize: 'var(--font-size-sm)' }}>Go Version</div>
          <div style={{ fontWeight: '600' }}>{info.go_version}</div>
        </div>
        
        <div>
          <div className="muted" style={{ fontSize: 'var(--font-size-sm)' }}>OS</div>
          <div style={{ fontWeight: '600' }}>{info.os} ({info.arch})</div>
        </div>
        
        
        <div>
          <div className="muted" style={{ fontSize: 'var(--font-size-sm)' }}>Admin Port</div>
          <div style={{ fontWeight: '600' }}>{info.admin_port}</div>
        </div>
        
        <div>
          <div className="muted" style={{ fontSize: 'var(--font-size-sm)' }}>DNS Port</div>
          <div style={{ fontWeight: '600' }}>{info.dns_port}</div>
        </div>
        
        <div>
          <div className="muted" style={{ fontSize: 'var(--font-size-sm)' }}>Uptime</div>
          <div style={{ fontWeight: '600' }}>{info.uptime}</div>
        </div>
        
        {info.build_time && (
          <div>
            <div className="muted" style={{ fontSize: 'var(--font-size-sm)' }}>Build Time</div>
            <div style={{ fontWeight: '600' }}>
              {new Date(info.build_time).toLocaleString()}
            </div>
          </div>
        )}
      </div>
    </Card>
  );
}
