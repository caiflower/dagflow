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
      style={{ background: 'var(--canvas-bg)' }}
    >
      <Background color="var(--canvas-grid)" gap={16} size={1} />
      <Controls
        className="[&>button]:!border-[var(--control-border)] [&>button]:!rounded-[var(--radius-sm)]"
        style={{
          background: 'var(--control-bg)',
          border: '1px solid var(--control-border)',
          borderRadius: 'var(--radius-md)',
        }}
      />
      <MiniMap
        nodeColor={miniMapNodeColor}
        className="!border-[var(--control-border)] !rounded-[var(--radius-md)]"
        style={{ background: 'var(--control-bg)', border: '1px solid var(--control-border)' }}
        maskColor="rgba(0,0,0,0.4)"
      />

      {/* Legend */}
      <Panel position="top-right">
        <div
          className="rounded-[var(--radius-md)] p-3 text-xs space-y-1.5"
          style={{
            background: 'var(--glass-bg)',
            backdropFilter: 'blur(12px)',
            border: '1px solid var(--border-subtle)',
          }}
        >
          <div className="font-medium mb-1" style={{ color: 'var(--text-muted)' }}>节点类型</div>
          <div className="flex items-center gap-2">
            <span className="w-2.5 h-2.5 rounded-full" style={{ background: getNodeTypeColor('start').border }} />
            <span style={{ color: 'var(--text-secondary)' }}>开始</span>
          </div>
          <div className="flex items-center gap-2">
            <span className="w-2.5 h-2.5 rounded-full" style={{ background: getNodeTypeColor('task').border }} />
            <span style={{ color: 'var(--text-secondary)' }}>任务</span>
          </div>
          <div className="flex items-center gap-2">
            <span className="w-2.5 h-2.5 rounded-full" style={{ background: getNodeTypeColor('branch').border }} />
            <span style={{ color: 'var(--text-secondary)' }}>分支</span>
          </div>
          <div className="flex items-center gap-2">
            <span className="w-2.5 h-2.5 rounded-full" style={{ background: getNodeTypeColor('end').border }} />
            <span style={{ color: 'var(--text-secondary)' }}>结束</span>
          </div>
          {mode === 'editor' && (
            <div
              className="mt-2 pt-2 text-[10px]"
              style={{ borderTop: '1px solid var(--border-subtle)', color: 'var(--text-muted)' }}
            >
              拖拽移动 · 连线创建
            </div>
          )}
        </div>
      </Panel>
    </ReactFlow>
  );
}
