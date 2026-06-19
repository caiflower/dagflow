/**
 * 具体节点类型组件 — StartNode / EndNode / TaskNode / BranchNode
 */
import { memo } from 'react';
import type { NodeProps, Node } from '@xyflow/react';
import BaseNode, { type BaseNodeData } from './BaseNode';
import { getNodeTypeColor, getProtocolIcon } from '../../../utils/stateColor';

// ===== StartNode =====
function StartNode({ data }: NodeProps<Node<BaseNodeData>>) {
  const colors = getNodeTypeColor('start');
  return (
    <BaseNode
      data={data}
      accentColor={colors.border}
      icon={<span className="text-green-400">▶</span>}
    />
  );
}

// ===== EndNode =====
function EndNode({ data }: NodeProps<Node<BaseNodeData>>) {
  const colors = getNodeTypeColor('end');
  return (
    <BaseNode
      data={data}
      accentColor={colors.border}
      icon={<span className="text-red-400">⏹</span>}
    />
  );
}

// ===== TaskNode =====
function TaskNode({ data }: NodeProps<Node<BaseNodeData>>) {
  const colors = getNodeTypeColor('task');
  const icon = data.icon || getProtocolIcon(data.protocol || '');
  return (
    <BaseNode
      data={data}
      accentColor={colors.border}
      icon={<span>{icon}</span>}
      extra={
        data.config && Object.keys(data.config).length > 0 ? (
          <div className="text-[10px] text-gray-400 space-y-0.5">
            {Object.entries(data.config).slice(0, 3).map(([k, v]) => (
              <div key={k} className="truncate">
                <span className="text-gray-500">{k}:</span>{' '}
                <span className="text-gray-300">{String(v).slice(0, 20)}</span>
              </div>
            ))}
            {Object.keys(data.config).length > 3 && (
              <div className="text-gray-600">+{Object.keys(data.config).length - 3} more</div>
            )}
          </div>
        ) : null
      }
    />
  );
}

// ===== BranchNode =====
function BranchNode({ data }: NodeProps<Node<BaseNodeData>>) {
  const colors = getNodeTypeColor('branch');
  return (
    <BaseNode
      data={data}
      accentColor={colors.border}
      icon={<span className="text-yellow-400">◇</span>}
      extra={
        data.config?.condition ? (
          <div className="text-[10px] text-yellow-200/70 truncate">
            条件: {String(data.config.condition)}
          </div>
        ) : null
      }
    />
  );
}

// ===== 导出 nodeTypes 注册对象 =====
export const nodeTypes = {
  start: memo(StartNode),
  end: memo(EndNode),
  task: memo(TaskNode),
  branch: memo(BranchNode),
};
