/**
 * 基础节点组件 — 所有自定义节点的共享 UI 框架
 */
import { memo, type ReactNode } from 'react';
import { Handle, Position } from '@xyflow/react';
import { getNodeStateColor } from '../../../utils/stateColor';

export interface BaseNodeData extends Record<string, unknown> {
  label: string;
  nodeType: string;
  protocol?: string;
  config?: Record<string, unknown>;
  state?: string;
  icon?: string;
  selected?: boolean;
  children?: ReactNode;
}

interface BaseNodeProps {
  data: BaseNodeData;
  /** 顶部装饰色（节点类型色） */
  accentColor?: string;
  /** 节点图标 (React element from Lucide) */
  icon?: ReactNode;
  /** 额外内容（展开区域） */
  extra?: ReactNode;
}

function BaseNode({ data, accentColor, icon, extra }: BaseNodeProps) {
  const state = data.state || 'pending';
  const stateColor = getNodeStateColor(state);
  const isSelected = data.selected === true;

  return (
    <div
      className="relative min-w-[160px] rounded-[var(--radius-md)] overflow-hidden transition-all duration-200 cursor-pointer"
      style={{
        background: 'var(--node-bg)',
        border: `1.5px solid ${isSelected ? 'var(--node-selected-border)' : 'var(--node-border)'}`,
        boxShadow: isSelected
          ? '0 0 0 3px color-mix(in srgb, var(--node-selected-border) 25%, transparent), var(--shadow-md)'
          : stateColor.glow !== 'none'
            ? stateColor.glow
            : 'var(--shadow-sm)',
      }}
    >
      {/* In Handle */}
      {data.nodeType !== 'start' && (
        <Handle
          type="target"
          position={Position.Top}
          className="!w-2.5 !h-2.5 !border-2"
          style={{
            background: 'var(--node-bg)',
            borderColor: accentColor || 'var(--border-default)',
          }}
        />
      )}

      {/* Accent stripe */}
      <div className="h-0.5" style={{ background: accentColor || 'var(--accent-primary)' }} />

      {/* Body */}
      <div className="px-3 py-2.5">
        <div className="flex items-center gap-2.5">
          {icon && (
            <div
              className="p-1.5 rounded-[var(--radius-sm)] flex-shrink-0"
              style={{ background: accentColor ? `${accentColor}20` : 'var(--accent-subtle)' }}
            >
              {icon}
            </div>
          )}
          <div className="flex-1 min-w-0">
            <div className="text-sm font-medium truncate" style={{ color: 'var(--text-primary)' }}>
              {data.label}
            </div>
            <div className="flex items-center gap-1.5 mt-0.5">
              {data.protocol && (
                <span
                  className="text-[10px] px-1.5 py-0.5 rounded-[var(--radius-sm)] font-mono"
                  style={{ background: 'var(--bg-tertiary)', color: 'var(--text-muted)' }}
                >
                  {data.protocol}
                </span>
              )}
              {state !== 'pending' && (
                <span
                  className="text-[10px] px-1.5 py-0.5 rounded-[var(--radius-sm)] font-medium"
                  style={{ background: stateColor.border + '20', color: stateColor.border }}
                >
                  {state}
                </span>
              )}
            </div>
          </div>
        </div>
        {extra && (
          <div className="mt-2 pt-2" style={{ borderTop: '1px solid var(--border-subtle)' }}>
            {extra}
          </div>
        )}
      </div>

      {/* Out Handle */}
      {data.nodeType !== 'end' && (
        <Handle
          type="source"
          position={Position.Bottom}
          className="!w-2.5 !h-2.5 !border-2"
          style={{
            background: 'var(--node-bg)',
            borderColor: accentColor || 'var(--border-default)',
          }}
        />
      )}
    </div>
  );
}

export default memo(BaseNode);
