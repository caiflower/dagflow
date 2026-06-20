import { useEffect, useState, useCallback } from 'react';
import { Play, Zap, Clock } from 'lucide-react';
import { useExecutionStore } from '../store';
import { useFlowStore } from '../store';
import { flowApi } from '../api/client';
import type { FlowNode, NodeInput } from '../types';
import Button from '../components/ui/Button';
import Badge from '../components/ui/Badge';
import { TextArea } from '../components/ui/Input';
import { stateBadgeVariant } from '../utils/stateColor';

export default function ExecutionPage() {
  const { executions, currentExec, runFlow, loadExecutions } = useExecutionStore();
  const { flows, loadFlows } = useFlowStore();
  const [selectedFlowId, setSelectedFlowId] = useState<number | null>(null);
  const [running, setRunning] = useState(false);
  const [flowNodes, setFlowNodes] = useState<FlowNode[]>([]);
  const [nodeInputs, setNodeInputs] = useState<Record<string, string>>({});

  useEffect(() => { loadExecutions(); loadFlows(); }, [loadExecutions, loadFlows]);

  useEffect(() => {
    if (!selectedFlowId) {
      setFlowNodes([]);
      setNodeInputs({});
      return;
    }
    flowApi.get(selectedFlowId).then((flow) => {
      try {
        const nodes: FlowNode[] = JSON.parse(flow.nodesJSON || '[]');
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
    <div className="max-w-5xl mx-auto px-6 py-8">
      {/* Header */}
      <div className="mb-8">
        <h1 className="text-2xl font-bold tracking-tight" style={{ color: 'var(--text-primary)' }}>
          执行记录
        </h1>
        <p className="text-sm mt-1" style={{ color: 'var(--text-muted)' }}>
          运行和追踪你的工作流
        </p>
      </div>

      {/* Controls */}
      <div className="flex items-center gap-3 mb-6">
        <select
          value={selectedFlowId || ''}
          onChange={(e) => setSelectedFlowId(e.target.value ? Number(e.target.value) : null)}
          className="px-3.5 py-2.5 text-sm rounded-[var(--radius-md)] border outline-none transition-all min-w-[200px]"
          style={{
            background: 'var(--bg-input)',
            borderColor: 'var(--border-default)',
            color: 'var(--text-primary)',
          }}
          onFocus={(e) => { e.currentTarget.style.borderColor = 'var(--accent-primary)'; }}
          onBlur={(e) => { e.currentTarget.style.borderColor = 'var(--border-default)'; }}
        >
          <option value="">选择 Flow...</option>
          {flows.map((f) => <option key={f.id} value={f.id}>{f.name}</option>)}
        </select>
        <Button
          onClick={handleRun}
          disabled={!selectedFlowId}
          loading={running}
        >
          <Play size={15} />
          {running ? '执行中...' : '执行'}
        </Button>
      </div>

      {/* Node Inputs */}
      {flowNodes.length > 0 && (
        <div
          className="mb-8 rounded-[var(--radius-lg)] p-5"
          style={{ background: 'var(--bg-secondary)', border: '1px solid var(--border-subtle)' }}
        >
          <div className="flex items-center gap-2 mb-4">
            <Zap size={16} style={{ color: 'var(--accent-primary)' }} />
            <h3 className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>节点输入参数</h3>
          </div>
          <div className="space-y-4">
            {flowNodes.map((node) => (
              <TextArea
                key={node.id}
                label={`${node.name} (${node.protocol || node.type})`}
                value={nodeInputs[node.name] || ''}
                onChange={(e) => updateInput(node.name, e.target.value)}
                rows={2}
                placeholder={`为 "${node.name}" 输入 JSON 参数，例如: {"key": "value"}`}
              />
            ))}
          </div>
        </div>
      )}

      {/* Current execution */}
      {currentExec && (
        <div
          className="mb-6 rounded-[var(--radius-lg)] p-5 animate-slide-up"
          style={{ background: 'var(--bg-secondary)', border: '1px solid var(--accent-primary)', boxShadow: 'var(--shadow-glow-blue)' }}
        >
          <div className="flex items-center justify-between mb-4">
            <h3 className="font-semibold text-sm" style={{ color: 'var(--text-primary)' }}>
              {currentExec.flowName} — 当前执行
            </h3>
            <Badge variant={stateBadgeVariant[currentExec.state]} dot>
              {currentExec.state}
            </Badge>
          </div>
          <div className="flex gap-2 flex-wrap">
            {currentExec.nodes?.map((n) => (
              <Badge key={n.id} variant={stateBadgeVariant[n.state] || 'default'} dot>
                {n.name}: {n.state}
              </Badge>
            ))}
          </div>
        </div>
      )}

      {/* Execution history */}
      <div className="space-y-2">
        {executions.map((exec) => (
          <div
            key={exec.id}
            className="flex items-center justify-between px-4 py-3 rounded-[var(--radius-md)] transition-all duration-150"
            style={{ background: 'var(--bg-secondary)', border: '1px solid var(--border-subtle)' }}
            onMouseEnter={(e) => {
              e.currentTarget.style.borderColor = 'var(--border-default)';
              e.currentTarget.style.boxShadow = 'var(--shadow-sm)';
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.borderColor = 'var(--border-subtle)';
              e.currentTarget.style.boxShadow = '';
            }}
          >
            <div className="flex items-center gap-3">
              <Clock size={14} style={{ color: 'var(--text-muted)' }} />
              <span className="font-medium text-sm" style={{ color: 'var(--text-primary)' }}>{exec.flowName}</span>
              <span className="text-xs font-mono" style={{ color: 'var(--text-muted)' }}>{exec.id}</span>
            </div>
            <Badge variant={stateBadgeVariant[exec.state]} dot>
              {exec.state}
            </Badge>
          </div>
        ))}
        {executions.length === 0 && (
          <div className="text-center py-16 rounded-[var(--radius-lg)]" style={{ background: 'var(--bg-secondary)', border: '1px dashed var(--border-default)' }}>
            <Play size={40} className="mx-auto mb-3" style={{ color: 'var(--text-muted)' }} />
            <p style={{ color: 'var(--text-muted)' }}>暂无执行记录</p>
          </div>
        )}
      </div>
    </div>
  );
}
