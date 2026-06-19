import { useEffect, useState } from 'react';
import { useExecutionStore } from '../store';
import { useFlowStore } from '../store';

const stateColors: Record<string, string> = {
  pending: 'bg-yellow-500/20 text-yellow-400',
  running: 'bg-blue-500/20 text-blue-400',
  succeeded: 'bg-green-500/20 text-green-400',
  failed: 'bg-red-500/20 text-red-400',
};

export default function ExecutionPage() {
  const { executions, currentExec, runFlow, loadExecutions } = useExecutionStore();
  const { flows, loadFlows } = useFlowStore();
  const [selectedFlowId, setSelectedFlowId] = useState<number | null>(null);
  const [running, setRunning] = useState(false);

  useEffect(() => { loadExecutions(); loadFlows(); }, [loadExecutions, loadFlows]);

  const handleRun = async () => {
    if (!selectedFlowId) return;
    setRunning(true);
    try { await runFlow(selectedFlowId); } finally { setRunning(false); }
  };

  return (
    <div className="p-6">
      <h2 className="text-2xl font-bold mb-6">执行记录</h2>

      <div className="flex gap-3 mb-6">
        <select value={selectedFlowId || ''} onChange={(e) => setSelectedFlowId(Number(e.target.value))}
          className="px-3 py-2 bg-gray-800 border border-gray-700 rounded-lg text-sm focus:outline-none focus:border-blue-500">
          <option value="">选择 Flow...</option>
          {flows.map((f) => <option key={f.id} value={f.id}>{f.name}</option>)}
        </select>
        <button onClick={handleRun} disabled={!selectedFlowId || running}
          className="px-4 py-2 bg-green-600 hover:bg-green-700 disabled:bg-gray-600 rounded-lg text-sm">
          {running ? '执行中...' : '▶ 执行'}
        </button>
      </div>

      {/* Current execution */}
      {currentExec && (
        <div className="mb-6 bg-gray-800 border border-gray-700 rounded-lg p-4">
          <div className="flex justify-between items-center mb-3">
            <h3 className="font-semibold">{currentExec.flowName} — 当前执行</h3>
            <span className={`px-2 py-0.5 rounded text-xs ${stateColors[currentExec.state]}`}>{currentExec.state}</span>
          </div>
          <div className="flex gap-2 flex-wrap">
            {currentExec.nodes?.map((n) => (
              <div key={n.id} className={`px-2 py-1 rounded text-xs ${stateColors[n.state] || 'bg-gray-600'}`}>
                {n.name}: {n.state}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Execution history */}
      <div className="space-y-2">
        {executions.map((exec) => (
          <div key={exec.id} className="bg-gray-800 border border-gray-700 rounded-lg p-3 flex justify-between items-center">
            <div>
              <span className="font-medium text-sm">{exec.flowName}</span>
              <span className="text-xs text-gray-500 ml-2">{exec.id}</span>
            </div>
            <span className={`px-2 py-0.5 rounded text-xs ${stateColors[exec.state]}`}>{exec.state}</span>
          </div>
        ))}
        {executions.length === 0 && <div className="text-center py-8 text-gray-500">暂无执行记录</div>}
      </div>
    </div>
  );
}
