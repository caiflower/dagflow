import { useEffect, useState, useCallback } from 'react';
import { Play, Zap, Clock, ChevronDown, ChevronRight, ChevronLeft, Terminal, Globe, Search, X, Filter } from 'lucide-react';
import { useExecutionStore } from '../store';
import { useFlowStore } from '../store';
import { flowApi, executionApi } from '../api/client';
import type { FlowNode, NodeInput, Execution, NodeStatus } from '../types';
import Button from '../components/ui/Button';
import Badge from '../components/ui/Badge';
import { TextArea } from '../components/ui/Input';
import { stateBadgeVariant } from '../utils/stateColor';

function formatDuration(ms: number): string {
  if (!ms || ms <= 0) return '-';
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${Math.floor(ms / 60000)}m ${Math.round((ms % 60000) / 1000)}s`;
}

function formatJson(str: string): string {
  if (!str) return '-';
  try {
    return JSON.stringify(JSON.parse(str), null, 2);
  } catch {
    return str;
  }
}

function NodeDetailPanel({ node }: { node: NodeStatus }) {
  const [expanded, setExpanded] = useState(false);
  const hasDetails = node.input || node.output || node.duration_ms > 0;

  return (
    <div
      className="rounded-[var(--radius-md)] overflow-hidden transition-all duration-150"
      style={{ background: 'var(--bg-primary)', border: '1px solid var(--border-subtle)' }}
    >
      <button
        className="w-full flex items-center justify-between px-3 py-2.5 text-left hover:bg-[var(--bg-hover)]"
        onClick={() => hasDetails && setExpanded(!expanded)}
        style={{ cursor: hasDetails ? 'pointer' : 'default' }}
      >
        <div className="flex items-center gap-2.5 min-w-0">
          {hasDetails ? (
            expanded ? <ChevronDown size={14} style={{ color: 'var(--text-muted)' }} /> : <ChevronRight size={14} style={{ color: 'var(--text-muted)' }} />
          ) : (
            <div style={{ width: 14 }} />
          )}
          <span className="text-xs font-mono px-1.5 py-0.5 rounded" style={{ background: 'var(--bg-secondary)', color: 'var(--text-muted)' }}>
            {node.node_type || 'task'}
          </span>
          <span className="text-sm font-medium truncate" style={{ color: 'var(--text-primary)' }}>{node.name}</span>
          {node.protocol && (
            <span className="flex items-center gap-1 text-xs" style={{ color: 'var(--text-muted)' }}>
              {node.protocol === 'http' ? <Globe size={11} /> : <Terminal size={11} />}
              {node.protocol}
            </span>
          )}
        </div>
        <div className="flex items-center gap-2.5 shrink-0 ml-3">
          {node.duration_ms > 0 && (
            <span className="text-xs font-mono" style={{ color: 'var(--text-muted)' }}>
              {formatDuration(node.duration_ms)}
            </span>
          )}
          <Badge variant={stateBadgeVariant[node.state] || 'default'} dot>
            {node.state}
          </Badge>
        </div>
      </button>

      {expanded && hasDetails && (
        <div className="px-3 pb-3 space-y-2.5" style={{ borderTop: '1px solid var(--border-subtle)' }}>
          {node.start_time && (
            <div className="flex items-center gap-2 pt-2">
              <Clock size={12} style={{ color: 'var(--text-muted)' }} />
              <span className="text-xs" style={{ color: 'var(--text-muted)' }}>
                {node.start_time} → {node.end_time || '-'}
              </span>
            </div>
          )}
          {node.input && (
            <div>
              <div className="text-xs font-medium mb-1" style={{ color: 'var(--text-muted)' }}>Input</div>
              <pre className="text-xs p-2 rounded-[var(--radius-sm)] overflow-x-auto max-h-32 whitespace-pre-wrap" style={{ background: 'var(--bg-secondary)', color: 'var(--text-primary)' }}>
                {formatJson(node.input)}
              </pre>
            </div>
          )}
          {node.output && (
            <div>
              <div className="text-xs font-medium mb-1" style={{ color: node.state === 'failed' ? 'var(--color-danger)' : 'var(--text-muted)' }}>
                {node.state === 'failed' ? 'Error' : 'Output'}
              </div>
              <pre className="text-xs p-2 rounded-[var(--radius-sm)] overflow-x-auto max-h-48 whitespace-pre-wrap" style={{ background: 'var(--bg-secondary)', color: 'var(--text-primary)' }}>
                {formatJson(node.output)}
              </pre>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function ExecutionDetail({ exec }: { exec: Execution }) {
  return (
    <div
      className="rounded-[var(--radius-lg)] p-5 animate-slide-up"
      style={{ background: 'var(--bg-secondary)', border: '1px solid var(--accent-primary)', boxShadow: 'var(--shadow-glow-blue)' }}
    >
      <div className="flex items-center justify-between mb-4">
        <h3 className="font-semibold text-sm" style={{ color: 'var(--text-primary)' }}>
          {exec.flow_name} — 执行详情
        </h3>
        <div className="flex items-center gap-2">
          {exec.task_id && (
            <span className="text-xs font-mono" style={{ color: 'var(--text-muted)' }}>task: {exec.task_id}</span>
          )}
          <Badge variant={stateBadgeVariant[exec.state] || 'default'} dot>
            {exec.state}
          </Badge>
        </div>
      </div>
      <div className="space-y-1.5">
        {exec.nodes?.map((n) => (
          <NodeDetailPanel key={n.id} node={n} />
        ))}
      </div>
    </div>
  );
}

export default function ExecutionPage() {
  const { executions, currentExec, runFlow, loadExecutions, pollExecution, total } = useExecutionStore();
  const { flows, loadFlows } = useFlowStore();
  const [page, setPage] = useState(1);
  const pageSize = 20;
  const totalPages = Math.max(1, Math.ceil(total / pageSize));
  const [selectedFlowId, setSelectedFlowId] = useState<number | null>(null);
  const [running, setRunning] = useState(false);
  const [flowNodes, setFlowNodes] = useState<FlowNode[]>([]);
  const [nodeInputs, setNodeInputs] = useState<Record<string, string>>({});
  const [expandedExecId, setExpandedExecId] = useState<string | null>(null);
  const [expandedExec, setExpandedExec] = useState<Execution | null>(null);
  const [loadingDetail, setLoadingDetail] = useState(false);
  const [filterFlowId, setFilterFlowId] = useState<number | null>(null);
  const [searchId, setSearchId] = useState('');
  const [searchError, setSearchError] = useState('');

  const handleSearchExec = async () => {
    const id = searchId.trim();
    if (!id) return;
    setSearchError('');
    try {
      const exec = await executionApi.get(id);
      setExpandedExecId(exec.id);
      setExpandedExec(exec);
      if (exec.state === 'running' || exec.state === 'pending') {
        pollExecution(exec.id);
      }
    } catch {
      setSearchError('未找到执行记录: ' + id);
    }
  };

  const handleClearSearch = () => {
    setSearchId('');
    setSearchError('');
  };

  useEffect(() => { loadExecutions(page, pageSize, filterFlowId || undefined); }, [loadExecutions, page, filterFlowId]);
  useEffect(() => { loadFlows(); }, [loadFlows]);

  useEffect(() => {
    if (!selectedFlowId) {
      setFlowNodes([]);
      setNodeInputs({});
      return;
    }
    flowApi.get(selectedFlowId).then((flow) => {
      try {
        const nodes: FlowNode[] = JSON.parse(flow.nodes_json || '[]');
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
        .map(([nodeName, input]) => ({ node_name: nodeName, input }));
      await runFlow(selectedFlowId, inputs.length > 0 ? inputs : undefined);
    } finally {
      setRunning(false);
    }
  };

  const updateInput = useCallback((nodeName: string, value: string) => {
    setNodeInputs((prev) => ({ ...prev, [nodeName]: value }));
  }, []);

  const handleExpandExec = async (execId: string) => {
    if (expandedExecId === execId) {
      setExpandedExecId(null);
      setExpandedExec(null);
      return;
    }
    setExpandedExecId(execId);
    setLoadingDetail(true);
    try {
      const exec = await executionApi.get(execId);
      setExpandedExec(exec);
      // 如果还在运行，开始轮询
      if (exec.state === 'running' || exec.state === 'pending') {
        pollExecution(execId);
      }
    } finally {
      setLoadingDetail(false);
    }
  };

  // 当 currentExec 更新时，如果正在查看它，同步更新
  useEffect(() => {
    if (expandedExecId && currentExec && currentExec.id === expandedExecId) {
      setExpandedExec(currentExec);
    }
  }, [currentExec, expandedExecId]);

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

      {/* Run Section */}
      <div
        className="rounded-[var(--radius-lg)] p-4 mb-6"
        style={{ background: 'var(--bg-secondary)', border: '1px solid var(--border-subtle)' }}
      >
        <div className="flex items-center gap-3">
          <Play size={15} style={{ color: 'var(--accent-primary)', flexShrink: 0 }} />
          <select
            value={selectedFlowId || ''}
            onChange={(e) => setSelectedFlowId(e.target.value ? Number(e.target.value) : null)}
            className="px-3 py-2 text-sm rounded-[var(--radius-md)] border outline-none transition-all min-w-[180px]"
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
            size="sm"
            onClick={handleRun}
            disabled={!selectedFlowId}
            loading={running}
          >
            <Play size={14} />
            {running ? '执行中...' : '执行'}
          </Button>
        </div>

        {/* Node Inputs (inline) */}
        {flowNodes.length > 0 && (
          <div className="mt-4 pt-4 space-y-3" style={{ borderTop: '1px solid var(--border-subtle)' }}>
            <div className="flex items-center gap-2 mb-2">
              <Zap size={14} style={{ color: 'var(--accent-primary)' }} />
              <span className="text-xs font-semibold" style={{ color: 'var(--text-secondary)' }}>节点输入参数</span>
            </div>
            {flowNodes.map((node) => (
              <TextArea
                key={node.id}
                label={`${node.name} (${node.protocol || node.type})`}
                value={nodeInputs[node.name] || ''}
                onChange={(e) => updateInput(node.name, e.target.value)}
                rows={2}
                placeholder={`为 "${node.name}" 输入 JSON 参数`}
              />
            ))}
          </div>
        )}
      </div>

      {/* Current execution (auto-expanded) */}
      {currentExec && expandedExecId !== currentExec.id && (
        <div className="mb-6">
          <ExecutionDetail exec={currentExec} />
        </div>
      )}

      {/* Toolbar: Search + Filter + Count */}
      <div
        className="flex items-center gap-2 mb-3 p-2 rounded-[var(--radius-md)]"
        style={{ background: 'var(--bg-secondary)', border: '1px solid var(--border-subtle)' }}
      >
        {/* ID Search */}
        <div className="relative flex-1 max-w-sm">
          <Search size={13} className="absolute left-2.5 top-1/2 -translate-y-1/2" style={{ color: 'var(--text-muted)' }} />
          <input
            type="text"
            value={searchId}
            onChange={(e) => { setSearchId(e.target.value); setSearchError(''); }}
            onKeyDown={(e) => e.key === 'Enter' && handleSearchExec()}
            placeholder="执行 ID 查询..."
            className="w-full pl-7 pr-7 py-1.5 text-xs rounded-[var(--radius-sm)] border outline-none transition-all"
            style={{
              background: 'var(--bg-input)',
              borderColor: searchError ? 'var(--danger)' : 'var(--border-subtle)',
              color: 'var(--text-primary)',
            }}
            onFocus={(e) => { e.currentTarget.style.borderColor = searchError ? 'var(--danger)' : 'var(--accent-primary)'; }}
            onBlur={(e) => { e.currentTarget.style.borderColor = searchError ? 'var(--danger)' : 'var(--border-subtle)'; }}
          />
          {searchId && (
            <button
              className="absolute right-2 top-1/2 -translate-y-1/2 p-0.5 rounded"
              style={{ color: 'var(--text-muted)' }}
              onClick={handleClearSearch}
            >
              <X size={12} />
            </button>
          )}
        </div>
        {/* Flow Filter */}
        <div className="flex items-center gap-1.5">
          <Filter size={12} style={{ color: 'var(--text-muted)', flexShrink: 0 }} />
          <select
            value={filterFlowId || ''}
            onChange={(e) => { setFilterFlowId(e.target.value ? Number(e.target.value) : null); setPage(1); }}
            className="px-2.5 py-1.5 text-xs rounded-[var(--radius-sm)] border outline-none transition-all"
            style={{
              background: 'var(--bg-input)',
              borderColor: filterFlowId ? 'var(--accent-primary)' : 'var(--border-subtle)',
              color: 'var(--text-primary)',
            }}
          >
            <option value="">全部 Flow</option>
            {flows.map((f) => <option key={f.id} value={f.id}>{f.name}</option>)}
          </select>
          {filterFlowId && (
            <button
              className="p-1 rounded-[var(--radius-sm)] transition-colors"
              style={{ color: 'var(--text-muted)' }}
              onClick={() => { setFilterFlowId(null); setPage(1); }}
            >
              <X size={12} />
            </button>
          )}
        </div>
        {/* Separator */}
        <div className="w-px h-4 mx-1" style={{ background: 'var(--border-subtle)' }} />
        {/* Count + Error */}
        <span className="text-xs whitespace-nowrap" style={{ color: 'var(--text-muted)' }}>
          {total > 0 ? `${total} 条` : ''}
        </span>
        {searchError && (
          <span className="text-xs whitespace-nowrap" style={{ color: 'var(--danger)' }}>{searchError}</span>
        )}
      </div>
      <div className="space-y-2">
        {executions.map((exec) => (
          <div key={exec.id}>
            <div
              className="flex items-center justify-between px-4 py-3 rounded-[var(--radius-md)] transition-all duration-150 cursor-pointer"
              style={{
                background: expandedExecId === exec.id ? 'var(--bg-tertiary, var(--bg-secondary))' : 'var(--bg-secondary)',
                border: `1px solid ${expandedExecId === exec.id ? 'var(--accent-primary)' : 'var(--border-subtle)'}`,
              }}
              onClick={() => handleExpandExec(exec.id)}
              onMouseEnter={(e) => {
                if (expandedExecId !== exec.id) {
                  e.currentTarget.style.borderColor = 'var(--border-default)';
                  e.currentTarget.style.boxShadow = 'var(--shadow-sm)';
                }
              }}
              onMouseLeave={(e) => {
                if (expandedExecId !== exec.id) {
                  e.currentTarget.style.borderColor = 'var(--border-subtle)';
                  e.currentTarget.style.boxShadow = '';
                }
              }}
            >
              <div className="flex items-center gap-3">
                <Clock size={14} style={{ color: 'var(--text-muted)' }} />
                <span className="font-medium text-sm" style={{ color: 'var(--text-primary)' }}>{exec.flow_name}</span>
                <span className="text-xs font-mono" style={{ color: 'var(--text-muted)' }}>{exec.id}</span>
                {exec.start_time && (
                  <span className="text-xs" style={{ color: 'var(--text-muted)' }}>{exec.start_time}</span>
                )}
              </div>
              <div className="flex items-center gap-2">
                <Badge variant={stateBadgeVariant[exec.state] || 'default'} dot>
                  {exec.state}
                </Badge>
                {expandedExecId === exec.id ? <ChevronDown size={14} style={{ color: 'var(--text-muted)' }} /> : <ChevronRight size={14} style={{ color: 'var(--text-muted)' }} />}
              </div>
            </div>
            {/* Expanded detail */}
            {expandedExecId === exec.id && (
              <div className="mt-2 ml-4">
                {loadingDetail && !expandedExec ? (
                  <div className="text-center py-4 text-sm" style={{ color: 'var(--text-muted)' }}>加载中...</div>
                ) : expandedExec ? (
                  <ExecutionDetail exec={expandedExec} />
                ) : null}
              </div>
            )}
          </div>
        ))}
        {executions.length === 0 && (
          <div className="text-center py-16 rounded-[var(--radius-lg)]" style={{ background: 'var(--bg-secondary)', border: '1px dashed var(--border-default)' }}>
            <Play size={40} className="mx-auto mb-3" style={{ color: 'var(--text-muted)' }} />
            <p style={{ color: 'var(--text-muted)' }}>暂无执行记录</p>
          </div>
        )}
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-center gap-3 mt-6 px-1">
          <button
            className="flex items-center gap-1 text-xs px-3 py-1.5 rounded-[var(--radius-sm)] transition-colors"
            style={{ color: page <= 1 ? 'var(--text-muted)' : 'var(--text-secondary)', cursor: page <= 1 ? 'default' : 'pointer', opacity: page <= 1 ? 0.5 : 1 }}
            disabled={page <= 1}
            onClick={() => setPage((p) => Math.max(1, p - 1))}
          >
            <ChevronLeft size={14} /> 上一页
          </button>
          <span className="text-xs" style={{ color: 'var(--text-muted)' }}>
            第 {page}/{totalPages} 页
          </span>
          <button
            className="flex items-center gap-1 text-xs px-3 py-1.5 rounded-[var(--radius-sm)] transition-colors"
            style={{ color: page >= totalPages ? 'var(--text-muted)' : 'var(--text-secondary)', cursor: page >= totalPages ? 'default' : 'pointer', opacity: page >= totalPages ? 0.5 : 1 }}
            disabled={page >= totalPages}
            onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
          >
            下一页 <ChevronRight size={14} />
          </button>
        </div>
      )}
    </div>
  );
}
