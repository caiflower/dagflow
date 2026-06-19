import { useCallback, useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import type { Connection, Node, Edge } from '@xyflow/react';
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

export default function FlowEditorPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { currentFlow, loadFlow } = useFlowStore();
  const [saving, setSaving] = useState(false);
  const [flowNodes, setFlowNodes] = useState<FlowNodeType[]>([]);
  const [_flowEdges, setFlowEdges] = useState<FlowEdgeType[]>([]);
  const [rfNodes, setRfNodes] = useState<Node[]>([]);
  const [rfEdges, setRfEdges] = useState<Edge[]>([]);

  useEffect(() => {
    if (id) loadFlow(Number(id));
  }, [id, loadFlow]);

  useEffect(() => {
    if (!currentFlow) return;
    const nodes = parseNodesJSON(currentFlow.nodesJSON);
    const edges = parseEdgesJSON(currentFlow.edgesJSON);
    // 如果没有位置信息，自动布局
    const hasPosition = nodes.some((n) => n.position && (n.position.x !== 0 || n.position.y !== 0));
    const laid = hasPosition ? nodes : autoLayout(nodes, edges);
    setFlowNodes(laid);
    setFlowEdges(edges);
    setRfNodes(flowNodesToReactFlow(laid));
    setRfEdges(flowEdgesToReactFlow(edges));
  }, [currentFlow]);

  const onConnect = useCallback(
    (params: Connection) => {
      const newEdge: FlowEdgeType = {
        id: `e-${params.source}-${params.target}`,
        source: params.source || '',
        target: params.target || '',
        type: 'control',
      };
      setFlowEdges((prev) => [...prev, newEdge]);
      setRfEdges((prev) => [
        ...prev,
        ...flowEdgesToReactFlow([newEdge]),
      ]);
    },
    [],
  );

  const handleSave = async () => {
    if (!currentFlow) return;
    setSaving(true);
    try {
      // 从 ReactFlow 节点中提取最新位置
      const updatedNodes = reactFlowToFlowNodes(rfNodes).map((fn) => {
        const orig = flowNodes.find((n) => n.id === fn.id);
        return orig ? { ...orig, ...fn, position: fn.position } : fn;
      });
      const updatedEdges = reactFlowToFlowEdges(rfEdges);
      await flowApi.update({
        id: currentFlow.id,
        nodes: updatedNodes,
        edges: updatedEdges,
      });
      loadFlow(currentFlow.id);
    } finally {
      setSaving(false);
    }
  };

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
      <div className="flex-1">
        <DAGViewer
          nodes={rfNodes}
          edges={rfEdges}
          mode="editor"
          onConnect={onConnect}
        />
      </div>
    </div>
  );
}
