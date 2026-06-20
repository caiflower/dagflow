import { useCallback, useEffect, useMemo, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  useNodesState,
  useEdgesState,
  type Connection,
  type Node,
  type Edge,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import { ArrowLeft, Save, Play, Settings, GitBranch, Circle, Square, Plus } from 'lucide-react';
import { DAGViewer } from '../components/dag';
import { useFlowStore, useExecutionStore } from '../store';
import { flowApi, protocolApi } from '../api/client';
import type { FlowNode as FlowNodeType, Protocol, ConfigField, NodeInput } from '../types';
import {
  flowNodesToReactFlow,
  flowEdgesToReactFlow,
  reactFlowToFlowNodes,
  reactFlowToFlowEdges,
  parseNodesJSON,
  parseEdgesJSON,
} from '../utils/flowSerializer';
import { autoLayout } from '../utils/dagLayout';
import Button from '../components/ui/Button';
import Modal from '../components/ui/Modal';
import Input from '../components/ui/Input';
import { TextArea } from '../components/ui/Input';
import Badge from '../components/ui/Badge';

const NODE_TEMPLATES: { type: FlowNodeType['type']; label: string; icon: React.ComponentType<{ size?: number; style?: React.CSSProperties }> }[] = [
  { type: 'task', label: '任务节点', icon: Settings },
  { type: 'branch', label: '分支节点', icon: GitBranch },
  { type: 'start', label: '开始节点', icon: Circle },
  { type: 'end', label: '结束节点', icon: Square },
];

let nodeCounter = 0;

export default function FlowEditorPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { currentFlow, loadFlow } = useFlowStore();
  const { runFlow } = useExecutionStore();
  const [saving, setSaving] = useState(false);
  const [running, setRunning] = useState(false);
  const [showRunModal, setShowRunModal] = useState(false);
  const [runNodeInputs, setRunNodeInputs] = useState<Record<string, string>>({});
  const [nodes, setNodes, onNodesChange] = useNodesState<Node>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);
  const [protocols, setProtocols] = useState<Protocol[]>([]);

  useEffect(() => {
    if (id) loadFlow(Number(id));
  }, [id, loadFlow]);

  useEffect(() => {
    protocolApi.list().then(setProtocols).catch(() => {});
  }, []);

  useEffect(() => {
    if (!currentFlow) return;
    const flowNodes = parseNodesJSON(currentFlow.nodes_json);
    const flowEdges = parseEdgesJSON(currentFlow.edges_json);
    const hasPosition = flowNodes.some((n) => n.position && (n.position.x !== 0 || n.position.y !== 0));
    const laid = hasPosition ? flowNodes : autoLayout(flowNodes, flowEdges);
    setNodes(flowNodesToReactFlow(laid));
    setEdges(flowEdgesToReactFlow(flowEdges));
  }, [currentFlow, setNodes, setEdges]);

  const selectedNode = useMemo(
    () => nodes.find((n) => n.id === selectedNodeId) || null,
    [nodes, selectedNodeId],
  );

  const selectedProtocolSchema = useMemo(() => {
    if (!selectedNode) return null;
    const proto = (selectedNode.data as Record<string, unknown>)?.protocol as string;
    if (!proto) return null;
    return protocols.find((p) => p.name === proto) || null;
  }, [selectedNode, protocols]);

  const onConnect = useCallback(
    (params: Connection) => {
      const newEdge: Edge = {
        id: `e-${params.source}-${params.target}`,
        source: params.source || '',
        target: params.target || '',
        type: 'control',
        data: { edgeType: 'control' },
      };
      setEdges((prev) => [...prev, newEdge]);
    },
    [setEdges],
  );

  const handleNodeClick = useCallback((nodeId: string) => {
    setSelectedNodeId(nodeId);
  }, []);

  const handlePaneClick = useCallback(() => {
    setSelectedNodeId(null);
  }, []);

  const addNode = useCallback(
    (type: FlowNodeType['type']) => {
      nodeCounter++;
      const newNode: Node = {
        id: `${type}_${Date.now()}`,
        type,
        position: { x: 250 + nodeCounter * 30, y: 150 + nodeCounter * 80 },
        data: {
          label: `${type === 'task' ? '任务' : type === 'branch' ? '分支' : type === 'start' ? '开始' : '结束'} ${nodeCounter}`,
          nodeType: type,
          protocol: type === 'task' ? 'local' : '',
          config: {},
          state: 'pending',
        },
      };
      setNodes((prev) => [...prev, newNode]);
    },
    [setNodes],
  );

  const updateNodeProp = useCallback(
    (field: string, value: unknown) => {
      if (!selectedNodeId) return;
      setNodes((prev) =>
        prev.map((n) => {
          if (n.id !== selectedNodeId) return n;
          const data = { ...n.data } as Record<string, unknown>;
          if (field === 'label') {
            data.label = value;
          } else if (field === 'protocol') {
            data.protocol = value;
            data.config = {};
          } else if (field.startsWith('config.')) {
            const key = field.slice(7);
            const config = { ...(data.config as Record<string, unknown> || {}) };
            if (value === '' || value === undefined || value === null) {
              delete config[key];
            } else {
              config[key] = value;
            }
            data.config = config;
          }
          return { ...n, data };
        }),
      );
    },
    [selectedNodeId, setNodes],
  );

  const deleteSelectedNode = useCallback(() => {
    if (!selectedNodeId) return;
    setNodes((prev) => prev.filter((n) => n.id !== selectedNodeId));
    setEdges((prev) => prev.filter((e) => e.source !== selectedNodeId && e.target !== selectedNodeId));
    setSelectedNodeId(null);
  }, [selectedNodeId, setNodes, setEdges]);

  const inputNodes = useMemo(() => {
    return nodes.filter((n) => {
      const data = n.data as Record<string, unknown>;
      const t = data?.nodeType as string;
      return t === 'task' || t === 'branch';
    });
  }, [nodes]);

  const handleSave = useCallback(async () => {
    if (!currentFlow) return;
    setSaving(true);
    try {
      const updatedNodes = reactFlowToFlowNodes(nodes);
      const updatedEdges = reactFlowToFlowEdges(edges);
      await flowApi.update({
        id: currentFlow.id,
        nodes: updatedNodes,
        edges: updatedEdges,
      });
      loadFlow(currentFlow.id);
    } finally {
      setSaving(false);
    }
  }, [currentFlow, nodes, edges, loadFlow]);

  const handleRun = useCallback(async () => {
    if (!currentFlow) return;
    setRunning(true);
    try {
      const inputs: NodeInput[] = Object.entries(runNodeInputs)
        .filter(([, v]) => v.trim() !== '')
        .map(([nodeName, input]) => ({ node_name: nodeName, input }));
      await runFlow(currentFlow.id, inputs.length > 0 ? inputs : undefined);
      setShowRunModal(false);
      setRunNodeInputs({});
      navigate('/executions');
    } finally {
      setRunning(false);
    }
  }, [currentFlow, runNodeInputs, runFlow, navigate]);

  if (!currentFlow) return (
    <div className="flex items-center justify-center h-full" style={{ color: 'var(--text-muted)' }}>
      <div className="animate-spin w-6 h-6 border-2 border-current border-t-transparent rounded-full mr-3" />
      加载中...
    </div>
  );

  return (
    <div className="h-full flex flex-col">
      {/* Top Bar */}
      <div
        className="flex items-center justify-between px-4 h-12 flex-shrink-0"
        style={{ borderBottom: '1px solid var(--border-subtle)' }}
      >
        <div className="flex items-center gap-3">
          <button
            onClick={() => navigate('/flows')}
            className="flex items-center gap-1.5 text-sm px-2 py-1 rounded-[var(--radius-sm)] transition-colors"
            style={{ color: 'var(--text-secondary)' }}
            onMouseEnter={(e) => {
              e.currentTarget.style.background = 'var(--bg-secondary)';
              e.currentTarget.style.color = 'var(--text-primary)';
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.background = '';
              e.currentTarget.style.color = 'var(--text-secondary)';
            }}
          >
            <ArrowLeft size={15} />
            返回
          </button>
          <div className="w-px h-5" style={{ background: 'var(--border-default)' }} />
          <h2 className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>{currentFlow.name}</h2>
          <Badge variant="default">v{currentFlow.version}</Badge>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="secondary"
            size="sm"
            onClick={() => { setRunNodeInputs({}); setShowRunModal(true); }}
          >
            <Play size={14} /> 运行
          </Button>
          <Button
            size="sm"
            onClick={handleSave}
            loading={saving}
          >
            <Save size={14} /> {saving ? '保存中...' : '保存'}
          </Button>
        </div>
      </div>

      <div className="flex-1 flex overflow-hidden">
        {/* Left Panel: Add Nodes */}
        <div
          className="w-52 flex-shrink-0 overflow-y-auto p-3 space-y-1.5"
          style={{ borderRight: '1px solid var(--border-subtle)', background: 'var(--bg-secondary)' }}
        >
          <div className="text-[11px] font-medium uppercase tracking-wider px-2 mb-2" style={{ color: 'var(--text-muted)' }}>
            添加节点
          </div>
          {NODE_TEMPLATES.map((t) => {
            const Icon = t.icon;
            return (
              <button
                key={t.type}
                onClick={() => addNode(t.type)}
                className="w-full flex items-center gap-2.5 px-3 py-2.5 text-sm rounded-[var(--radius-md)] transition-all duration-150"
                style={{ color: 'var(--text-primary)' }}
                onMouseEnter={(e) => { e.currentTarget.style.background = 'var(--bg-tertiary)'; }}
                onMouseLeave={(e) => { e.currentTarget.style.background = ''; }}
              >
                <div className="p-1 rounded-[var(--radius-sm)]" style={{ background: 'var(--accent-subtle)' }}>
                  <Icon size={14} style={{ color: 'var(--accent-primary)' }} />
                </div>
                <span>{t.label}</span>
              </button>
            );
          })}
          <div
            className="text-[11px] mt-4 pt-3 px-2 leading-relaxed"
            style={{ borderTop: '1px solid var(--border-subtle)', color: 'var(--text-muted)' }}
          >
            点击添加到画布，<br />拖拽调整位置，<br />连接 Handle 创建边
          </div>
        </div>

        {/* Canvas */}
        <div className="flex-1 min-w-0">
          <DAGViewer
            nodes={nodes}
            edges={edges}
            mode="editor"
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onConnect={onConnect}
            onNodeClick={handleNodeClick}
            onPaneClick={handlePaneClick}
            selectedNodeId={selectedNodeId || undefined}
          />
        </div>

        {/* Right Panel: Node Properties */}
        {selectedNode && (
          <NodePropsPanel
            node={selectedNode}
            protocols={protocols}
            protocolSchema={selectedProtocolSchema}
            onUpdate={updateNodeProp}
            onDelete={deleteSelectedNode}
            onClose={() => setSelectedNodeId(null)}
          />
        )}
      </div>

      {/* Run Modal */}
      <Modal
        open={showRunModal}
        onClose={() => { setShowRunModal(false); setRunNodeInputs({}); }}
        title="运行 Flow"
        width={520}
        footer={
          <>
            <Button variant="secondary" onClick={() => { setShowRunModal(false); setRunNodeInputs({}); }}>
              取消
            </Button>
            <Button onClick={handleRun} loading={running}>
              <Play size={14} /> {running ? '执行中...' : '确认执行'}
            </Button>
          </>
        }
      >
        {inputNodes.length === 0 ? (
          <div className="text-center py-8" style={{ color: 'var(--text-muted)' }}>
            无可设置输入的节点
          </div>
        ) : (
          <div className="space-y-4">
            {inputNodes.map((n) => {
              const data = n.data as Record<string, unknown>;
              const name = (data.label as string) || n.id;
              const protocol = (data.protocol as string) || '';
              return (
                <TextArea
                  key={n.id}
                  label={`${name}${protocol ? ` (${protocol})` : ''}`}
                  value={runNodeInputs[name] || ''}
                  onChange={(e) => setRunNodeInputs((prev) => ({ ...prev, [name]: e.target.value }))}
                  rows={2}
                  placeholder={`输入 JSON 参数，例如: {"key": "value"}`}
                />
              );
            })}
          </div>
        )}
      </Modal>
    </div>
  );
}

// ===== Node Properties Panel =====
interface NodePropsPanelProps {
  node: Node;
  protocols: Protocol[];
  protocolSchema: Protocol | null;
  onUpdate: (field: string, value: unknown) => void;
  onDelete: () => void;
  onClose: () => void;
}

function NodePropsPanel({ node, protocols, protocolSchema, onUpdate, onDelete, onClose }: NodePropsPanelProps) {
  const data = node.data as Record<string, unknown>;
  const nodeType = (data.nodeType as string) || 'task';
  const protocol = (data.protocol as string) || '';
  const config = (data.config as Record<string, unknown>) || {};
  const isEditable = nodeType === 'task' || nodeType === 'branch';

  return (
    <div
      className="w-72 flex-shrink-0 flex flex-col overflow-hidden"
      style={{ borderLeft: '1px solid var(--border-subtle)', background: 'var(--bg-secondary)' }}
    >
      {/* Header */}
      <div
        className="flex items-center justify-between px-4 h-12"
        style={{ borderBottom: '1px solid var(--border-subtle)' }}
      >
        <span className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>节点属性</span>
        <button
          onClick={onClose}
          className="p-1 rounded-[var(--radius-sm)] transition-colors"
          style={{ color: 'var(--text-muted)' }}
          onMouseEnter={(e) => {
            e.currentTarget.style.background = 'var(--bg-tertiary)';
            e.currentTarget.style.color = 'var(--text-primary)';
          }}
          onMouseLeave={(e) => {
            e.currentTarget.style.background = '';
            e.currentTarget.style.color = 'var(--text-muted)';
          }}
        >
          ✕
        </button>
      </div>

      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        {/* ID (read-only) */}
        <div>
          <label className="block text-[11px] font-medium mb-1.5" style={{ color: 'var(--text-muted)' }}>ID</label>
          <div
            className="text-xs font-mono px-2.5 py-1.5 rounded-[var(--radius-sm)]"
            style={{ background: 'var(--bg-input)', color: 'var(--text-secondary)', border: '1px solid var(--border-subtle)' }}
          >
            {node.id}
          </div>
        </div>

        {/* Name */}
        <Input
          label="名称"
          type="text"
          value={(data.label as string) || ''}
          onChange={(e) => onUpdate('label', e.target.value)}
        />

        {/* Type (read-only) */}
        <div>
          <label className="block text-[11px] font-medium mb-1.5" style={{ color: 'var(--text-muted)' }}>类型</label>
          <Badge variant="default">{nodeType}</Badge>
        </div>

        {/* Protocol select */}
        {isEditable && (
          <div className="flex flex-col gap-1.5">
            <label className="text-[11px] font-medium" style={{ color: 'var(--text-muted)' }}>协议</label>
            <select
              value={protocol}
              onChange={(e) => onUpdate('protocol', e.target.value)}
              className="w-full px-3 py-2 text-sm rounded-[var(--radius-md)] border outline-none transition-all"
              style={{
                background: 'var(--bg-input)',
                borderColor: 'var(--border-default)',
                color: 'var(--text-primary)',
              }}
              onFocus={(e) => { e.currentTarget.style.borderColor = 'var(--accent-primary)'; }}
              onBlur={(e) => { e.currentTarget.style.borderColor = 'var(--border-default)'; }}
            >
              <option value="">选择协议...</option>
              {protocols.map((p) => (
                <option key={p.name} value={p.name}>
                  {p.display_name || p.name}
                </option>
              ))}
            </select>
            {protocolSchema?.description && (
              <p className="text-[11px]" style={{ color: 'var(--text-muted)' }}>{protocolSchema.description}</p>
            )}
          </div>
        )}

        {/* Config fields */}
        {isEditable && protocolSchema && protocolSchema.config_schema?.fields.length > 0 && (
          <div className="space-y-3">
            <div className="text-[11px] font-medium uppercase tracking-wider" style={{ color: 'var(--text-muted)' }}>
              配置参数
            </div>
            {protocolSchema.config_schema.fields.map((field: ConfigField) => (
              <ConfigFieldInput
                key={field.name}
                field={field}
                value={config[field.name]}
                onChange={(v) => onUpdate(`config.${field.name}`, v)}
              />
            ))}
          </div>
        )}

        {/* Manual config editing */}
        {isEditable && protocol && !protocolSchema && (
          <div className="space-y-3">
            <div className="text-[11px] font-medium uppercase tracking-wider" style={{ color: 'var(--text-muted)' }}>
              配置参数
            </div>
            {Object.entries(config).map(([k, v]) => (
              <Input
                key={k}
                label={k}
                type="text"
                value={String(v ?? '')}
                onChange={(e) => onUpdate(`config.${k}`, e.target.value)}
              />
            ))}
            <button
              onClick={() => {
                const key = prompt('输入参数名:');
                if (key) onUpdate(`config.${key}`, '');
              }}
              className="flex items-center gap-1 text-xs font-medium px-2 py-1.5 rounded-[var(--radius-sm)] transition-colors"
              style={{ color: 'var(--accent-primary)' }}
              onMouseEnter={(e) => { e.currentTarget.style.background = 'var(--accent-subtle)'; }}
              onMouseLeave={(e) => { e.currentTarget.style.background = ''; }}
            >
              <Plus size={12} /> 添加参数
            </button>
          </div>
        )}
      </div>

      {/* Footer */}
      <div className="p-4" style={{ borderTop: '1px solid var(--border-subtle)' }}>
        <Button variant="danger" size="sm" className="w-full" onClick={onDelete}>
          删除节点
        </Button>
      </div>
    </div>
  );
}

// ===== Config Field Input =====
function ConfigFieldInput({ field, value, onChange }: { field: ConfigField; value: unknown; onChange: (v: unknown) => void }) {
  const strValue = value !== undefined && value !== null ? String(value) : field.default || '';

  if (field.type === 'textarea') {
    return (
      <TextArea
        label={field.label}
        value={strValue}
        onChange={(e) => onChange(e.target.value)}
        rows={3}
        placeholder={field.description}
      />
    );
  }

  if (field.type === 'select' && field.options?.length) {
    return (
      <div className="flex flex-col gap-1.5">
        <label className="text-[11px] font-medium" style={{ color: 'var(--text-muted)' }}>
          {field.label} {field.required && <span style={{ color: 'var(--danger)' }}>*</span>}
        </label>
        <select
          value={strValue}
          onChange={(e) => onChange(e.target.value)}
          className="w-full px-3 py-2 text-sm rounded-[var(--radius-md)] border outline-none transition-all"
          style={{
            background: 'var(--bg-input)',
            borderColor: 'var(--border-default)',
            color: 'var(--text-primary)',
          }}
        >
          <option value="">选择...</option>
          {field.options.map((opt) => (
            <option key={opt} value={opt}>{opt}</option>
          ))}
        </select>
      </div>
    );
  }

  if (field.type === 'boolean') {
    return (
      <div className="flex items-center justify-between py-1">
        <label className="text-[11px] font-medium" style={{ color: 'var(--text-muted)' }}>
          {field.label} {field.required && <span style={{ color: 'var(--danger)' }}>*</span>}
        </label>
        <input
          type="checkbox"
          checked={strValue === 'true' || strValue === '1'}
          onChange={(e) => onChange(e.target.checked ? 'true' : 'false')}
          className="rounded"
        />
      </div>
    );
  }

  return (
    <Input
      label={field.label}
      type={field.type === 'number' ? 'number' : 'text'}
      value={strValue}
      onChange={(e) => onChange(field.type === 'number' ? Number(e.target.value) : e.target.value)}
      placeholder={field.description}
    />
  );
}
