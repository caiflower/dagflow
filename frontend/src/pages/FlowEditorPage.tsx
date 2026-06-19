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
import { DAGViewer } from '../components/dag';
import { useFlowStore, useExecutionStore } from '../store';
import { flowApi, protocolApi } from '../api/client';
import type { FlowNode as FlowNodeType,  Protocol, ConfigField, NodeInput } from '../types';
import {
  flowNodesToReactFlow,
  flowEdgesToReactFlow,
  reactFlowToFlowNodes,
  reactFlowToFlowEdges,
  parseNodesJSON,
  parseEdgesJSON,
} from '../utils/flowSerializer';
import { autoLayout } from '../utils/dagLayout';

const NODE_TEMPLATES: { type: FlowNodeType['type']; label: string; icon: string }[] = [
  { type: 'task', label: '任务节点', icon: '⚙️' },
  { type: 'branch', label: '分支节点', icon: '◇' },
  { type: 'start', label: '开始节点', icon: '▶' },
  { type: 'end', label: '结束节点', icon: '⏹' },
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

  // 加载 Flow
  useEffect(() => {
    if (id) loadFlow(Number(id));
  }, [id, loadFlow]);

  // 加载协议列表
  useEffect(() => {
    protocolApi.list().then(setProtocols).catch(() => {});
  }, []);

  // Flow 数据 → ReactFlow 状态
  useEffect(() => {
    if (!currentFlow) return;
    const flowNodes = parseNodesJSON(currentFlow.nodesJSON);
    const flowEdges = parseEdgesJSON(currentFlow.edgesJSON);
    const hasPosition = flowNodes.some((n) => n.position && (n.position.x !== 0 || n.position.y !== 0));
    const laid = hasPosition ? flowNodes : autoLayout(flowNodes, flowEdges);
    setNodes(flowNodesToReactFlow(laid));
    setEdges(flowEdgesToReactFlow(flowEdges));
  }, [currentFlow, setNodes, setEdges]);

  // 当前选中的节点
  const selectedNode = useMemo(
    () => nodes.find((n) => n.id === selectedNodeId) || null,
    [nodes, selectedNodeId],
  );

  // 选中节点的协议 config schema
  const selectedProtocolSchema = useMemo(() => {
    if (!selectedNode) return null;
    const proto = (selectedNode.data as Record<string, unknown>)?.protocol as string;
    if (!proto) return null;
    return protocols.find((p) => p.name === proto) || null;
  }, [selectedNode, protocols]);

  // 连线
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

  // 点击节点
  const handleNodeClick = useCallback((nodeId: string) => {
    setSelectedNodeId(nodeId);
  }, []);

  // 点击画布空白处取消选中
  const handlePaneClick = useCallback(() => {
    setSelectedNodeId(null);
  }, []);

  // 添加节点
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

  // 更新选中节点的属性
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
            // 切换协议时重置 config
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

  // 删除选中节点
  const deleteSelectedNode = useCallback(() => {
    if (!selectedNodeId) return;
    setNodes((prev) => prev.filter((n) => n.id !== selectedNodeId));
    setEdges((prev) => prev.filter((e) => e.source !== selectedNodeId && e.target !== selectedNodeId));
    setSelectedNodeId(null);
  }, [selectedNodeId, setNodes, setEdges]);

  // 可编辑输入的节点列表（task / branch）
  const inputNodes = useMemo(() => {
    return nodes.filter((n) => {
      const data = n.data as Record<string, unknown>;
      const t = data?.nodeType as string;
      return t === 'task' || t === 'branch';
    });
  }, [nodes]);

  // 保存
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

  // 运行 Flow
  const handleRun = useCallback(async () => {
    if (!currentFlow) return;
    setRunning(true);
    try {
      const inputs: NodeInput[] = Object.entries(runNodeInputs)
        .filter(([, v]) => v.trim() !== '')
        .map(([nodeName, input]) => ({ nodeName, input }));
      const exec = await runFlow(currentFlow.id, inputs.length > 0 ? inputs : undefined);
      setShowRunModal(false);
      setRunNodeInputs({});
      // 跳转到执行页
      navigate('/executions');
    } finally {
      setRunning(false);
    }
  }, [currentFlow, runNodeInputs, runFlow, navigate]);

  if (!currentFlow) return <div className="p-6 text-gray-500">加载中...</div>;

  return (
    <div className="h-full flex flex-col">
      <div className="flex justify-between items-center p-4 border-b border-gray-700">
        <div className="flex items-center gap-3">
          <button onClick={() => navigate('/flows')} className="text-gray-400 hover:text-white text-sm">
            ← 返回
          </button>
          <h2 className="text-lg font-bold">{currentFlow.name}</h2>
          <span className="text-xs text-gray-500">v{currentFlow.version}</span>
        </div>
        <div className="flex gap-2">
          <button
            onClick={() => { setRunNodeInputs({}); setShowRunModal(true); }}
            className="px-4 py-1.5 bg-green-600 hover:bg-green-700 rounded text-sm"
          >
            ▶ 运行
          </button>
          <button
            onClick={handleSave}
            disabled={saving}
            className="px-4 py-1.5 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-600 rounded text-sm"
          >
            {saving ? '保存中...' : '保存'}
          </button>
        </div>
      </div>
      <div className="flex-1 flex overflow-hidden">
        {/* 左侧栏：添加节点 */}
        <div className="w-48 bg-gray-800 border-r border-gray-700 p-3 space-y-2 flex-shrink-0 overflow-y-auto">
          <div className="text-xs text-gray-400 font-medium mb-2">添加节点</div>
          {NODE_TEMPLATES.map((t) => (
            <button
              key={t.type}
              onClick={() => addNode(t.type)}
              className="w-full flex items-center gap-2 px-3 py-2 bg-gray-700 hover:bg-gray-600 rounded text-sm text-gray-200 transition-colors"
            >
              <span>{t.icon}</span>
              <span>{t.label}</span>
            </button>
          ))}
          <div className="text-[10px] text-gray-500 mt-4 pt-3 border-t border-gray-700">
            点击添加到画布，<br />拖拽调整位置，<br />连接 Handle 创建边
          </div>
        </div>
        {/* 画布 */}
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
        {/* 右侧栏：属性编辑 */}
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
      {/* 运行弹窗 */}
      {showRunModal && (
        <RunModal
          nodes={inputNodes}
          nodeInputs={runNodeInputs}
          onUpdate={(name, val) => setRunNodeInputs((prev) => ({ ...prev, [name]: val }))}
          onRun={handleRun}
          onCancel={() => { setShowRunModal(false); setRunNodeInputs({}); }}
          running={running}
        />
      )}
    </div>
  );
}

// ===== 节点属性编辑面板 =====
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
    <div className="w-72 bg-gray-800 border-l border-gray-700 flex-shrink-0 flex flex-col overflow-hidden">
      {/* 头部 */}
      <div className="flex items-center justify-between p-3 border-b border-gray-700">
        <span className="text-sm font-medium text-gray-200">节点属性</span>
        <button onClick={onClose} className="text-gray-500 hover:text-gray-300 text-xs">✕</button>
      </div>

      <div className="flex-1 overflow-y-auto p-3 space-y-4">
        {/* 节点 ID（只读） */}
        <div>
          <label className="block text-[10px] text-gray-500 mb-1">ID</label>
          <div className="text-xs text-gray-400 bg-gray-900 rounded px-2 py-1.5 font-mono">{node.id}</div>
        </div>

        {/* 名称 */}
        <div>
          <label className="block text-[10px] text-gray-500 mb-1">名称</label>
          <input
            type="text"
            value={(data.label as string) || ''}
            onChange={(e) => onUpdate('label', e.target.value)}
            className="w-full bg-gray-900 border border-gray-600 rounded px-2 py-1.5 text-sm text-gray-200 focus:border-blue-500 focus:outline-none"
          />
        </div>

        {/* 类型（只读） */}
        <div>
          <label className="block text-[10px] text-gray-500 mb-1">类型</label>
          <div className="text-xs text-gray-400 bg-gray-900 rounded px-2 py-1.5">{nodeType}</div>
        </div>

        {/* 协议选择（仅 task/branch 节点） */}
        {isEditable && (
          <div>
            <label className="block text-[10px] text-gray-500 mb-1">协议</label>
            <select
              value={protocol}
              onChange={(e) => onUpdate('protocol', e.target.value)}
              className="w-full bg-gray-900 border border-gray-600 rounded px-2 py-1.5 text-sm text-gray-200 focus:border-blue-500 focus:outline-none"
            >
              <option value="">选择协议...</option>
              {protocols.map((p) => (
                <option key={p.name} value={p.name}>
                  {p.displayName || p.name}
                </option>
              ))}
            </select>
            {protocolSchema?.description && (
              <p className="text-[10px] text-gray-500 mt-1">{protocolSchema.description}</p>
            )}
          </div>
        )}

        {/* Config 字段（基于协议 schema） */}
        {isEditable && protocolSchema && protocolSchema.configSchema?.fields.length > 0 && (
          <div className="space-y-3">
            <div className="text-[10px] text-gray-500 font-medium">配置参数</div>
            {protocolSchema.configSchema.fields.map((field: ConfigField) => (
              <ConfigFieldInput
                key={field.name}
                field={field}
                value={config[field.name]}
                onChange={(v) => onUpdate(`config.${field.name}`, v)}
              />
            ))}
          </div>
        )}

        {/* 无 schema 时的手动 config 编辑 */}
        {isEditable && protocol && !protocolSchema && (
          <div className="space-y-3">
            <div className="text-[10px] text-gray-500 font-medium">配置参数</div>
            {Object.entries(config).map(([k, v]) => (
              <div key={k}>
                <label className="block text-[10px] text-gray-500 mb-1">{k}</label>
                <input
                  type="text"
                  value={String(v ?? '')}
                  onChange={(e) => onUpdate(`config.${k}`, e.target.value)}
                  className="w-full bg-gray-900 border border-gray-600 rounded px-2 py-1.5 text-sm text-gray-200 focus:border-blue-500 focus:outline-none"
                />
              </div>
            ))}
            <button
              onClick={() => {
                const key = prompt('输入参数名:');
                if (key) onUpdate(`config.${key}`, '');
              }}
              className="w-full text-xs text-blue-400 hover:text-blue-300 py-1"
            >
              + 添加参数
            </button>
          </div>
        )}
      </div>

      {/* 底部：删除按钮 */}
      <div className="p-3 border-t border-gray-700">
        <button
          onClick={onDelete}
          className="w-full px-3 py-1.5 bg-red-900/40 hover:bg-red-900/60 text-red-400 rounded text-sm transition-colors"
        >
          删除节点
        </button>
      </div>
    </div>
  );
}

// ===== Config 字段输入组件 =====
function ConfigFieldInput({ field, value, onChange }: { field: ConfigField; value: unknown; onChange: (v: unknown) => void }) {
  const strValue = value !== undefined && value !== null ? String(value) : field.default || '';

  if (field.type === 'textarea') {
    return (
      <div>
        <label className="block text-[10px] text-gray-500 mb-1">
          {field.label} {field.required && <span className="text-red-400">*</span>}
        </label>
        <textarea
          value={strValue}
          onChange={(e) => onChange(e.target.value)}
          rows={3}
          className="w-full bg-gray-900 border border-gray-600 rounded px-2 py-1.5 text-sm text-gray-200 focus:border-blue-500 focus:outline-none resize-y"
          placeholder={field.description}
        />
      </div>
    );
  }

  if (field.type === 'select' && field.options?.length) {
    return (
      <div>
        <label className="block text-[10px] text-gray-500 mb-1">
          {field.label} {field.required && <span className="text-red-400">*</span>}
        </label>
        <select
          value={strValue}
          onChange={(e) => onChange(e.target.value)}
          className="w-full bg-gray-900 border border-gray-600 rounded px-2 py-1.5 text-sm text-gray-200 focus:border-blue-500 focus:outline-none"
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
      <div className="flex items-center justify-between">
        <label className="text-[10px] text-gray-500">
          {field.label} {field.required && <span className="text-red-400">*</span>}
        </label>
        <input
          type="checkbox"
          checked={strValue === 'true' || strValue === '1'}
          onChange={(e) => onChange(e.target.checked ? 'true' : 'false')}
          className="rounded border-gray-600 bg-gray-900"
        />
      </div>
    );
  }

  // string / number / default
  return (
    <div>
      <label className="block text-[10px] text-gray-500 mb-1">
        {field.label} {field.required && <span className="text-red-400">*</span>}
      </label>
      <input
        type={field.type === 'number' ? 'number' : 'text'}
        value={strValue}
        onChange={(e) => onChange(field.type === 'number' ? Number(e.target.value) : e.target.value)}
        className="w-full bg-gray-900 border border-gray-600 rounded px-2 py-1.5 text-sm text-gray-200 focus:border-blue-500 focus:outline-none"
        placeholder={field.description}
      />
    </div>
  );
}

// ===== 运行弹窗组件 =====
function RunModal({
  nodes,
  nodeInputs,
  onUpdate,
  onRun,
  onCancel,
  running,
}: {
  nodes: Node[];
  nodeInputs: Record<string, string>;
  onUpdate: (nodeName: string, value: string) => void;
  onRun: () => void;
  onCancel: () => void;
  running: boolean;
}) {
  return (
    <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
      <div className="bg-gray-800 border border-gray-700 rounded-xl shadow-2xl w-[500px] max-h-[80vh] flex flex-col">
        <div className="flex items-center justify-between p-4 border-b border-gray-700">
          <h3 className="text-lg font-semibold text-gray-200">▶ 运行 Flow</h3>
          <button onClick={onCancel} className="text-gray-500 hover:text-gray-300">✕</button>
        </div>
        <div className="flex-1 overflow-y-auto p-4 space-y-4">
          {nodes.length === 0 && (
            <div className="text-gray-500 text-sm text-center py-4">无可设置输入的节点</div>
          )}
          {nodes.map((n) => {
            const data = n.data as Record<string, unknown>;
            const name = (data.label as string) || n.id;
            const protocol = (data.protocol as string) || '';
            return (
              <div key={n.id}>
                <label className="block text-xs text-gray-400 mb-1">
                  <span className="font-medium text-gray-300">{name}</span>
                  {protocol && <span className="text-gray-500 ml-2">({protocol})</span>}
                </label>
                <textarea
                  value={nodeInputs[name] || ''}
                  onChange={(e) => onUpdate(name, e.target.value)}
                  rows={2}
                  placeholder={`输入 JSON 参数，例如: {"key": "value"}`}
                  className="w-full bg-gray-900 border border-gray-600 rounded px-3 py-2 text-sm text-gray-200 focus:border-blue-500 focus:outline-none resize-y font-mono"
                />
              </div>
            );
          })}
        </div>
        <div className="flex gap-3 p-4 border-t border-gray-700">
          <button
            onClick={onCancel}
            className="flex-1 px-4 py-2 bg-gray-700 hover:bg-gray-600 rounded text-sm"
          >
            取消
          </button>
          <button
            onClick={onRun}
            disabled={running}
            className="flex-1 px-4 py-2 bg-green-600 hover:bg-green-700 disabled:bg-gray-600 rounded text-sm"
          >
            {running ? '执行中...' : '▶ 确认执行'}
          </button>
        </div>
      </div>
    </div>
  );
}
