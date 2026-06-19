import { useCallback, useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  useNodesState,
  useEdgesState,
  applyNodeChanges,
  applyEdgeChanges,
  type Connection,
  type Node,
  type Edge,
  type OnNodesChange,
  type OnEdgesChange,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import { DAGViewer } from '../components/dag';
import { useFlowStore } from '../store';
import { flowApi } from '../api/client';
import type { FlowNode as FlowNodeType, FlowEdge as FlowEdgeType } from '../types';
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
  const [saving, setSaving] = useState(false);
  const [nodes, setNodes, onNodesChange] = useNodesState<Node>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);

  // 加载 Flow
  useEffect(() => {
    if (id) loadFlow(Number(id));
  }, [id, loadFlow]);

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
          protocol: '',
          config: {},
          state: 'pending',
        },
      };
      setNodes((prev) => [...prev, newNode]);
    },
    [setNodes],
  );

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
        <button
          onClick={handleSave}
          disabled={saving}
          className="px-4 py-1.5 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-600 rounded text-sm"
        >
          {saving ? '保存中...' : '保存'}
        </button>
      </div>
      <div className="flex-1 flex">
        {/* 侧边栏：添加节点 */}
        <div className="w-48 bg-gray-800 border-r border-gray-700 p-3 space-y-2 flex-shrink-0">
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
        <div className="flex-1">
          <DAGViewer
            nodes={nodes}
            edges={edges}
            mode="editor"
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onConnect={onConnect}
          />
        </div>
      </div>
    </div>
  );
}
