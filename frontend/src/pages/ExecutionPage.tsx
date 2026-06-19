import { useEffect, useState, useCallback } from 'react';
import { useExecutionStore } from '../store';
import { useFlowStore } from '../store';
import { flowApi } from '../api/client';
import type { FlowNode, NodeInput } from '../types';

const stateColors: Record<string, string> = {
  pending: 'bg-yellow-500/20 text-yellow-400',
  running: 'bg-blue-500/20 text-blue-400',
  succeeded: 'bg-green-500/20 text-green-400',
  failed: 'bg-red-500/20 text-red-400',
  skipped: 'bg-gray-500/20 text-gray-400',
};

export default function ExecutionPage() {
  const { executions, currentExec, runFlow, loadExecutions } = useExecutionStore();
  const { flows, loadFlows } = useFlowStore();
  const [selectedFlowId, setSelectedFlowId] = useState<number | null>(null);
  const [running, setRunning] = useState(false);
  const [flowNodes, setFlowNodes] = useState<FlowNode[]>([]);
  const [nodeInputs, setNodeInputs] = useState<Record<string, string>>({});

  useEffect(() => { loadExecutions(); loadFlows(); }, [loadExecutions, loadFlows]);

  // 加载选中 Flow 的节点列表
  useEffect(() => {
    if (!selectedFlowId) {
      setFlowNodes([]);
      setNodeInputs({});
      return;
    }
    flowApi.get(selectedFlowId).then((flow) => {
      try {
        const nodes: FlowNode[] = JSON.parse(flow.nodesJSON || '[]');
        // 只显示可设置输入的节点（task / branch）
        setFlowNodes(nodes.filter((n) => n.type === 'task' || n.type === 'branch'));
      } catch {
        setFlowNodes([]);
      }
      setNodeInputs({});
    }).catch(() => {
      setFlowNodes([]);
    });
  }, [selectedFlowId]);

  const handleRun = async () => {
    if (!selectedFlowId) return;
    setRunning(true);
    try {
      const inputs: NodeInput[] = Object.entries(nodeInputs)
        .filter(([, v]) => v.trim() !== '')
        .map(([nodeName, input]) => ({ nodeName, input }));
      await runFlow(selectedFlowId, inputs.length > 0 ? inputs : undefined);
    } finally {
      setRunning(false);
    }
  };

  const updateInput = useCallback((nodeName: string, value: string) => {
    setNodeInputs((prev) => ({ ...prev, [nodeName]: value }));
  }, []);

  return (
    <div className="p-6">
      <h2 className="text-2xl font-bold mb-6">执行记录</h2>

      <div className="flex gap-3 mb-4">
        <select value={selectedFlowId || ''} onChange={(e) => setSelectedFlowId(e.target.value ? Number(e.target.value) : null)}
          className="px-3 py-2 bg-gray-800 border border-gray-700 rounded-lg text-sm focus:outline-none focus:border-blue-500">
          <option value="">选择 Flow...</option>
          {flows.map((f) => <option key={f.id} value={f.id}>{f.name}</option>)}
        </select>
        <button onClick={handleRun} disabled={!selectedFlowId || running}
          className="px-4 py-2 bg-green-600 hover:bg-green-700 disabled:bg-gray-600 rounded-lg text-sm">
          {running ? '执行中...' : '▶ 执行'}
        </button>
      </div>

      {/* 节点输入参数 */}
      {flowNodes.length > 0 && (
        <div className="mb-6 bg-gray-800 border border-gray-700 rounded-lg p-4">
          <h3 className="text-sm font-medium text-gray-300 mb-3">节点输入参数</h3>
          <div className="space-y-3">
            {flowNodes.map((node) => (
              <div key={node.id}>
                <label className="block text-xs text-gray-400 mb-1">
                  <span className="font-medium text-gray-300">{node.name}</span>
                  <span className="text-gray-500 ml-2">({node.protocol || node.type})</span>
                </label>
                <textarea
                  value={nodeInputs[node.name] || ''}
                  onChange={(e) => updateInput(node.name, e.target.value)}
                  rows={2}
                  placeholder={`为 "${node.name}" 输入 JSON 参数，例如: {"key": "value"}`}
                  className="w-full bg-gray-900 border border-gray-600 rounded px-3 py-2 text-sm text-gray-200 focus:border-blue-500 focus:outline-none resize-y font-mono"
                />
              </div>
            ))}
          </div>
        </div>
      )}

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
