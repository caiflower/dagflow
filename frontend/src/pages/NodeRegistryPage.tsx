import { useEffect, useMemo, useRef, useState } from 'react';
import type { NodeDetail } from '../api/client';
import { nodeApi } from '../api/client';
import Badge from '../components/ui/Badge';
import Input from '../components/ui/Input';
import { RefreshCw, Server } from 'lucide-react';

function formatHeartbeat(ts: number): string {
  if (!ts) return '-';
  const diff = Math.floor((Date.now() - ts * 1000) / 1000);
  if (diff < 5) return 'just now';
  if (diff < 60) return `${diff}s ago`;
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  return `${Math.floor(diff / 3600)}h ago`;
}

export default function NodeRegistryPage() {
  const [nodes, setNodes] = useState<NodeDetail[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [search, setSearch] = useState('');
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [refreshInterval, setRefreshInterval] = useState(5);
  const intervalRef = useRef<ReturnType<typeof setInterval>>(null);

  const fetchNodes = async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await nodeApi.list();
      setNodes(data);
    } catch (err: any) {
      setError(err.message || 'Failed to fetch nodes');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchNodes();
  }, []);

  useEffect(() => {
    if (autoRefresh) {
      intervalRef.current = setInterval(fetchNodes, refreshInterval * 1000);
    }
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current);
    };
  }, [autoRefresh, refreshInterval]);

  const filtered = useMemo(() => {
    if (!search) return nodes;
    const s = search.toLowerCase();
    return nodes.filter(
      (n) =>
        n.nodeId.toLowerCase().includes(s) ||
        n.address.toLowerCase().includes(s) ||
        n.functions.some((f) => f.toLowerCase().includes(s))
    );
  }, [nodes, search]);

  const onlineCount = nodes.filter((n) => n.status === 'online').length;
  const offlineCount = nodes.length - onlineCount;

  return (
    <div className="p-6 space-y-5 animate-fade-in">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold" style={{ color: 'var(--text-primary)' }}>
            Node Registry
          </h1>
          <p className="text-sm mt-1" style={{ color: 'var(--text-secondary)' }}>
            Registered remote function nodes
          </p>
        </div>
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-2">
            <label className="text-xs font-medium" style={{ color: 'var(--text-secondary)' }}>
              Auto-refresh
            </label>
            <label className="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                checked={autoRefresh}
                onChange={(e) => setAutoRefresh(e.target.checked)}
                className="sr-only peer"
              />
              <div className="w-8 h-4.5 rounded-full peer peer-checked:bg-[var(--accent-primary)] bg-[var(--bg-tertiary)] transition-colors" />
              <div className="absolute left-0.5 top-0.25 w-3.5 h-3.5 rounded-full bg-white shadow-sm peer-checked:translate-x-3.5 transition-transform" />
            </label>
            <select
              value={refreshInterval}
              onChange={(e) => setRefreshInterval(Number(e.target.value))}
              className="text-xs rounded-md px-2 py-1 outline-none"
              style={{
                background: 'var(--bg-input)',
                border: '1px solid var(--border-default)',
                color: 'var(--text-primary)',
              }}
            >
              <option value={5}>5s</option>
              <option value={10}>10s</option>
              <option value={30}>30s</option>
              <option value={60}>60s</option>
            </select>
          </div>
          <button
            onClick={fetchNodes}
            disabled={loading}
            className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-full transition-all duration-150"
            style={{
              background: 'var(--bg-secondary)',
              color: 'var(--text-primary)',
              border: '1px solid var(--border-subtle)',
            }}
          >
            <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
            Refresh
          </button>
        </div>
      </div>

      {/* Stats */}
      <div className="flex gap-6 text-sm">
        <span style={{ color: 'var(--text-secondary)' }}>
          Total: <strong style={{ color: 'var(--text-primary)' }}>{nodes.length}</strong>
        </span>
        <span style={{ color: 'var(--success)' }}>
          Online: <strong>{onlineCount}</strong>
        </span>
        <span style={{ color: 'var(--danger)' }}>
          Offline: <strong>{offlineCount}</strong>
        </span>
      </div>

      {/* Search */}
      <Input
        placeholder="Search by node ID, address, or function name..."
        value={search}
        onChange={(e) => setSearch(e.target.value)}
      />

      {/* Error */}
      {error && (
        <div
          className="px-4 py-3 rounded-lg text-sm"
          style={{
            background: 'var(--danger-subtle)',
            border: '1px solid var(--danger)',
            color: 'var(--danger)',
          }}
        >
          {error}
        </div>
      )}

      {/* Table */}
      <div
        className="rounded-xl overflow-hidden"
        style={{ border: '1px solid var(--border-subtle)' }}
      >
        <table className="w-full text-sm">
          <thead style={{ background: 'var(--bg-secondary)' }}>
            <tr style={{ borderBottom: '1px solid var(--border-subtle)' }}>
              <th className="text-left px-4 py-3 font-medium" style={{ color: 'var(--text-secondary)' }}>
                Node ID
              </th>
              <th className="text-left px-4 py-3 font-medium" style={{ color: 'var(--text-secondary)' }}>
                Address
              </th>
              <th className="text-left px-4 py-3 font-medium" style={{ color: 'var(--text-secondary)' }}>
                Functions
              </th>
              <th className="text-left px-4 py-3 font-medium" style={{ color: 'var(--text-secondary)' }}>
                Status
              </th>
              <th className="text-left px-4 py-3 font-medium" style={{ color: 'var(--text-secondary)' }}>
                Last Heartbeat
              </th>
            </tr>
          </thead>
          <tbody>
            {filtered.length === 0 && !loading && (
              <tr>
                <td colSpan={5} className="text-center py-16">
                  <Server
                    className="mx-auto mb-3 opacity-20"
                    size={40}
                    style={{ color: 'var(--text-muted)' }}
                  />
                  <p className="text-sm" style={{ color: 'var(--text-muted)' }}>
                    {search ? 'No nodes match your search' : 'No nodes registered yet'}
                  </p>
                  <p className="text-xs mt-1" style={{ color: 'var(--text-muted)' }}>
                    {search
                      ? 'Try a different search term'
                      : 'Remote function nodes will appear here once they register'}
                  </p>
                </td>
              </tr>
            )}
            {filtered.map((node) => (
              <tr
                key={node.nodeId}
                style={{ borderBottom: '1px solid var(--border-subtle)' }}
                className="transition-colors duration-100"
                onMouseEnter={(e) => {
                  e.currentTarget.style.background = 'var(--bg-secondary)';
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.background = '';
                }}
              >
                <td
                  className="px-4 py-3 font-mono text-xs"
                  style={{ color: 'var(--text-primary)' }}
                >
                  {node.nodeId}
                </td>
                <td
                  className="px-4 py-3 font-mono text-xs"
                  style={{ color: 'var(--text-secondary)' }}
                >
                  {node.address}
                </td>
                <td className="px-4 py-3">
                  <div className="flex flex-wrap gap-1">
                    {node.functions.map((fn) => (
                      <Badge key={fn} variant="subtle">
                        {fn}
                      </Badge>
                    ))}
                  </div>
                </td>
                <td className="px-4 py-3">
                  <Badge
                    variant={node.status === 'online' ? 'success' : 'danger'}
                    dot
                  >
                    {node.status === 'online' ? 'Online' : 'Offline'}
                  </Badge>
                </td>
                <td className="px-4 py-3" style={{ color: 'var(--text-muted)' }}>
                  {formatHeartbeat(node.lastHeartbeat)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
