import { useEffect } from 'react';
import { useProtocolStore } from '../store';

export default function ProtocolPage() {
  const { protocols, loadProtocols } = useProtocolStore();
  useEffect(() => { loadProtocols(); }, [loadProtocols]);

  return (
    <div className="p-6">
      <h2 className="text-2xl font-bold mb-6">协议管理</h2>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {protocols.map((p) => (
          <div key={p.name} className="bg-gray-800 border border-gray-700 rounded-lg p-4">
            <div className="flex items-center gap-2 mb-2">
              <span className="text-lg">🔌</span>
              <h3 className="font-semibold text-lg">{p.displayName}</h3>
              <span className="text-xs bg-gray-700 px-2 py-0.5 rounded">{p.name}</span>
            </div>
            <p className="text-sm text-gray-400 mb-3">{p.description}</p>
            <div className="text-xs text-gray-500">
              <span className="font-medium">配置字段：</span>
              {p.configSchema.fields.map((f) => (
                <span key={f.name} className="inline-block bg-gray-700 px-1.5 py-0.5 rounded mr-1 mt-1">
                  {f.label} ({f.type})
                  {f.required && <span className="text-red-400 ml-0.5">*</span>}
                </span>
              ))}
            </div>
          </div>
        ))}
        {protocols.length === 0 && <div className="col-span-2 text-center py-8 text-gray-500">暂无协议</div>}
      </div>
    </div>
  );
}
