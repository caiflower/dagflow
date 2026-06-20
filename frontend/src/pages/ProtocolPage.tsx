import { useEffect } from 'react';
import { Plug, Globe, Radio, Terminal, Link, Settings } from 'lucide-react';
import { useProtocolStore } from '../store';
import Badge from '../components/ui/Badge';

const protocolIcons: Record<string, React.ComponentType<{ size?: number; style?: React.CSSProperties }>> = {
  http: Globe,
  grpc: Radio,
  local: Terminal,
  mcp: Link,
};

export default function ProtocolPage() {
  const { protocols, loadProtocols } = useProtocolStore();
  useEffect(() => { loadProtocols(); }, [loadProtocols]);

  return (
    <div className="max-w-5xl mx-auto px-6 py-8">
      {/* Header */}
      <div className="mb-8">
        <h1 className="text-2xl font-bold tracking-tight" style={{ color: 'var(--text-primary)' }}>
          协议管理
        </h1>
        <p className="text-sm mt-1" style={{ color: 'var(--text-muted)' }}>
          查看和管理可用的执行协议
        </p>
      </div>

      {/* Protocol Cards Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {protocols.map((p) => {
          const Icon = protocolIcons[p.name] || Settings;
          return (
            <div
              key={p.name}
              className="rounded-[var(--radius-lg)] p-5 transition-all duration-200"
              style={{ background: 'var(--bg-secondary)', border: '1px solid var(--border-subtle)' }}
              onMouseEnter={(e) => {
                e.currentTarget.style.borderColor = 'var(--border-default)';
                e.currentTarget.style.boxShadow = 'var(--shadow-md)';
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.borderColor = 'var(--border-subtle)';
                e.currentTarget.style.boxShadow = '';
              }}
            >
              <div className="flex items-center gap-3 mb-3">
                <div
                  className="p-2 rounded-[var(--radius-md)]"
                  style={{ background: 'var(--accent-subtle)' }}
                >
                  <Icon size={18} style={{ color: 'var(--accent-primary)' }} />
                </div>
                <div>
                  <h3 className="font-semibold" style={{ color: 'var(--text-primary)' }}>
                    {p.display_name}
                  </h3>
                  <span className="text-xs font-mono" style={{ color: 'var(--text-muted)' }}>{p.name}</span>
                </div>
              </div>
              <p className="text-sm mb-4" style={{ color: 'var(--text-secondary)' }}>{p.description}</p>
              <div>
                <span className="text-xs font-medium" style={{ color: 'var(--text-muted)' }}>配置字段</span>
                <div className="flex flex-wrap gap-1.5 mt-2">
                  {p.config_schema.fields.map((f) => (
                    <Badge key={f.name} variant={f.required ? 'subtle' : 'default'}>
                      {f.label}
                      <span className="opacity-60">({f.type})</span>
                    </Badge>
                  ))}
                </div>
              </div>
            </div>
          );
        })}
        {protocols.length === 0 && (
          <div className="col-span-2 text-center py-16 rounded-[var(--radius-lg)]" style={{ background: 'var(--bg-secondary)', border: '1px dashed var(--border-default)' }}>
            <Plug size={40} className="mx-auto mb-3" style={{ color: 'var(--text-muted)' }} />
            <p style={{ color: 'var(--text-muted)' }}>暂无协议</p>
          </div>
        )}
      </div>
    </div>
  );
}
