import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useFlowStore } from '../store';

export default function FlowListPage() {
  const { flows, total, loading, loadFlows, deleteFlow, createFlow } = useFlowStore();
  const navigate = useNavigate();
  const [searchName, setSearchName] = useState('');
  const [showCreate, setShowCreate] = useState(false);
  const [newName, setNewName] = useState('');
  const [newDesc, setNewDesc] = useState('');

  useEffect(() => { loadFlows(1, searchName); }, [loadFlows, searchName]);

  const handleCreate = async () => {
    if (!newName.trim()) return;
    const defaultNodes = [
      { id: 'start', name: '开始', type: 'start' as const, protocol: '', config: {}, position: { x: 250, y: 50 } },
      { id: 'end', name: '结束', type: 'end' as const, protocol: '', config: {}, position: { x: 250, y: 400 } },
    ];
    await createFlow({ name: newName, description: newDesc, nodes: defaultNodes, edges: [] });
    setShowCreate(false);
    setNewName('');
    setNewDesc('');
  };

  const stateColors: Record<string, string> = {
    1: 'bg-green-500/20 text-green-400',
    0: 'bg-red-500/20 text-red-400',
  };

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-2xl font-bold">Flow 列表</h2>
        <button onClick={() => setShowCreate(true)} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm">
          + 新建 Flow
        </button>
      </div>

      <input
        type="text" placeholder="搜索 Flow..." value={searchName}
        onChange={(e) => setSearchName(e.target.value)}
        className="w-full mb-4 px-4 py-2 bg-gray-800 border border-gray-700 rounded-lg text-sm focus:outline-none focus:border-blue-500"
      />

      {loading ? (
        <div className="text-center py-12 text-gray-500">加载中...</div>
      ) : (
        <div className="grid gap-4">
          {flows.map((flow) => (
            <div key={flow.id} className="bg-gray-800 border border-gray-700 rounded-lg p-4 hover:border-gray-600 cursor-pointer"
              onClick={() => navigate(`/flows/${flow.id}`)}>
              <div className="flex justify-between items-start">
                <div>
                  <h3 className="font-semibold text-lg">{flow.name}</h3>
                  <p className="text-sm text-gray-400 mt-1">{flow.description || '无描述'}</p>
                </div>
                <div className="flex items-center gap-2">
                  <span className={`px-2 py-0.5 rounded text-xs ${stateColors[flow.status] || ''}`}>
                    {flow.status === 1 ? '启用' : '禁用'}
                  </span>
                  <span className="text-xs text-gray-500">v{flow.version}</span>
                  <button onClick={(e) => { e.stopPropagation(); deleteFlow(flow.id); }}
                    className="text-red-400 hover:text-red-300 text-sm px-2">删除</button>
                </div>
              </div>
            </div>
          ))}
          {flows.length === 0 && <div className="text-center py-12 text-gray-500">暂无 Flow，点击上方按钮创建</div>}
        </div>
      )}
      <div className="mt-4 text-sm text-gray-500">共 {total} 条记录</div>

      {/* Create Dialog */}
      {showCreate && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-gray-800 rounded-xl p-6 w-96 border border-gray-700">
            <h3 className="text-lg font-bold mb-4">新建 Flow</h3>
            <input type="text" placeholder="Flow 名称" value={newName} onChange={(e) => setNewName(e.target.value)}
              className="w-full mb-3 px-3 py-2 bg-gray-700 border border-gray-600 rounded text-sm focus:outline-none focus:border-blue-500" />
            <textarea placeholder="描述（可选）" value={newDesc} onChange={(e) => setNewDesc(e.target.value)}
              className="w-full mb-4 px-3 py-2 bg-gray-700 border border-gray-600 rounded text-sm h-20 focus:outline-none focus:border-blue-500" />
            <div className="flex gap-2 justify-end">
              <button onClick={() => setShowCreate(false)} className="px-4 py-2 text-sm text-gray-400 hover:text-white">取消</button>
              <button onClick={handleCreate} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded text-sm">创建</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
