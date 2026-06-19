/**
 * DAGViewer — 封装 ReactFlow 容器（受控模式）
 *
 * 支持两种模式：
 *  - editor: 可编辑（拖拽、连线、删除）
 *  - preview: 只读（查看 + 状态展示）
 */
import { useCallback, useMemo } from 'react';
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  Panel,
  type Connection,
  type Node,
  type Edge,
  type OnNodesChange,
  type OnEdgesChange,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import { nodeTypes } from './nodes';
import { edgeTypes } from './edges';
import { getNodeTypeColor } from '../../utils/stateColor';

interface DAGViewerProps {
  nodes: Node[];
  edges: Edge[];
  mode?: 'editor' | 'preview';
  onNodesChange: OnNodesChange;
  onEdgesChange: OnEdgesChange;
  onConnect?: (params: Connection) => void;
  onNodeClick?: (nodeId: string) => void;
  onPaneClick?: () => void;
  selectedNodeId?: string;
  nodeStates?: Record<string, string>;
}

export default function DAGViewer({
  nodes,
  edges,
  mode = 'editor',
  onNodesChange,
  onEdgesChange,
  onConnect: externalOnConnect,
  onNodeClick,
  onPaneClick,
  selectedNodeId,
}: DAGViewerProps) {
  const isReadOnly = mode === 'preview';

  // 将选中状态注入 node data
  const nodesWithSelection = useMemo(
    () =>
      nodes.map((n) => ({
        ...n,
        data: { ...n.data, selected: n.id === selectedNodeId },
      })),
    [nodes, selectedNodeId],
  );

  const handleConnect = useCallback(
    (params: Connection) => {
      if (isReadOnly) return;
      externalOnConnect?.(params);
    },
    [isReadOnly, externalOnConnect],
  );

  const miniMapNodeColor = useMemo(
    () => (n: Node) => {
      const data = n.data as { nodeType?: string } | undefined;
      const type = data?.nodeType || 'task';
      return getNodeTypeColor(type).border;
    },
    [],
  );

  const handleNodeClick = useCallback(
    (_: React.MouseEvent, node: Node) => {
      onNodeClick?.(node.id);
    },
    [onNodeClick],
  );

  return (
    <ReactFlow
      nodes={nodesWithSelection}
      edges={edges}
      nodeTypes={nodeTypes}
      edgeTypes={edgeTypes}
      onNodesChange={isReadOnly ? undefined : onNodesChange}
      onEdgesChange={isReadOnly ? undefined : onEdgesChange}
      onConnect={handleConnect}
      onNodeClick={isReadOnly ? undefined : handleNodeClick}
      onPaneClick={isReadOnly ? undefined : onPaneClick}
      nodesDraggable={!isReadOnly}
      nodesConnectable={!isReadOnly}
      elementsSelectable={!isReadOnly}
      fitView
      fitViewOptions={{ padding: 0.2 }}
      className="bg-gray-900"
    >
      <Background color="#374151" gap={16} size={1} />
      <Controls className="!bg-gray-800 !border-gray-700 !rounded-lg [&>button]:!bg-gray-800 [&>button]:!border-gray-700 [&>button]:!text-gray-300 [&>button:hover]:!bg-gray-700" />
      <MiniMap
        nodeColor={miniMapNodeColor}
        className="!bg-gray-800 !border-gray-700 !rounded-lg"
        maskColor="rgba(0,0,0,0.5)"
      />

      {/* 图例 */}
      <Panel position="top-right">
        <div className="bg-gray-800/90 backdrop-blur-sm border border-gray-700 rounded-lg p-3 text-xs space-y-1.5">
          <div className="text-gray-400 font-medium mb-1">节点类型</div>
          <div className="flex items-center gap-2">
            <span className="w-3 h-3 rounded" style={{ background: getNodeTypeColor('start').border }} />
            <span className="text-gray-300">开始</span>
          </div>
          <div className="flex items-center gap-2">
            <span className="w-3 h-3 rounded" style={{ background: getNodeTypeColor('task').border }} />
            <span className="text-gray-300">任务</span>
          </div>
          <div className="flex items-center gap-2">
            <span className="w-3 h-3 rounded" style={{ background: getNodeTypeColor('branch').border }} />
            <span className="text-gray-300">分支</span>
          </div>
          <div className="flex items-center gap-2">
            <span className="w-3 h-3 rounded" style={{ background: getNodeTypeColor('end').border }} />
            <span className="text-gray-300">结束</span>
          </div>
          {mode === 'editor' && (
            <div className="text-gray-500 mt-2 pt-2 border-t border-gray-700">
              拖拽移动 · 连线创建
            </div>
          )}
        </div>
      </Panel>
    </ReactFlow>
  );
}
