/**
 * 具体节点类型组件 — StartNode / EndNode / TaskNode / BranchNode
 * 使用 Lucide 图标替代 emoji
 */
import { memo } from 'react';
import type { NodeProps, Node } from '@xyflow/react';
import { Play, Square, Settings, GitBranch, Globe, Radio, Terminal, Link, RadioTower } from 'lucide-react';
import BaseNode, { type BaseNodeData } from './BaseNode';
import { getNodeTypeColor } from '../../../utils/stateColor';

const lucideIcons: Record<string, React.ComponentType<{ size?: number; style?: React.CSSProperties }>> = {
  Globe, Radio, Terminal, Link, Settings, RadioTower,
};

function getProtocolLucideIcon(protocol: string) {
  const iconMap: Record<string, string> = {
    http: 'Globe', grpc: 'Radio', local: 'Terminal', mcp: 'Link', remoteFunc: 'RadioTower',
  };
  return lucideIcons[iconMap[protocol] || 'Settings'] || Settings;
}

// ===== StartNode =====
function StartNode({ data }: NodeProps<Node<BaseNodeData>>) {
  const colors = getNodeTypeColor('start');
  return (
    <BaseNode
      data={data}
      accentColor={colors.border}
      icon={<Play size={14} style={{ color: colors.border }} />}
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
      icon={<Square size={14} style={{ color: colors.border }} />}
    />
  );
}

// ===== TaskNode =====
function TaskNode({ data }: NodeProps<Node<BaseNodeData>>) {
  const colors = getNodeTypeColor('task');
  const protocol = (data.protocol as string) || '';
  const IconComp = protocol ? getProtocolLucideIcon(protocol) : Settings;
  return (
    <BaseNode
      data={data}
      accentColor={colors.border}
      icon={<IconComp size={14} style={{ color: colors.border }} />}
      extra={
        data.config && Object.keys(data.config).length > 0 ? (
          <div className="text-[10px] space-y-0.5">
            {Object.entries(data.config).slice(0, 3).map(([k, v]) => (
              <div key={k} className="text-[10px] px-1 py-0.5 rounded bg-[var(--bg-tertiary)] truncate">
                <span style={{ color: 'var(--text-muted)' }}>{k}:</span>{' '}
                <span style={{ color: 'var(--text-secondary)' }}>{String(v).slice(0, 20)}</span>
              </div>
            ))}
            {Object.keys(data.config).length > 3 && (
              <div style={{ color: 'var(--text-muted)' }}>+{Object.keys(data.config).length - 3} more</div>
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
      icon={<GitBranch size={14} style={{ color: colors.border }} />}
      extra={
        data.config?.condition ? (
          <div className="text-[10px] truncate" style={{ color: 'var(--warning)' }}>
            条件: {String(data.config.condition)}
          </div>
        ) : null
      }
    />
  );
}

// ===== Export nodeTypes =====
export const nodeTypes = {
  start: memo(StartNode),
  end: memo(EndNode),
  task: memo(TaskNode),
  branch: memo(BranchNode),
};
