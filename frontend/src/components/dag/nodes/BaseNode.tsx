/**
 * 基础节点组件 — 所有自定义节点的共享 UI 框架
 */
import { memo, useState, type ReactNode } from 'react';
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
  const [isHovered, setIsHovered] = useState(false);

  const isRunning = state === 'running';

  // Extract glow color from shadow string (e.g., '0 0 8px #40a0ff80' → '#40a0ff80')
  const glowColor = stateColor.glow !== 'none' ? stateColor.glow.split(' ').pop() : undefined;

  return (
    <div
      className={`relative min-w-[160px] rounded-[var(--radius-lg)] overflow-hidden cursor-pointer ${isRunning ? 'animate-pulse-glow' : ''}`}
      style={{
        background: 'var(--node-bg)',
        border: `1.5px solid ${isSelected ? 'var(--node-selected-border)' : 'var(--node-border)'}`,
        boxShadow: isSelected
          ? '0 0 0 3px color-mix(in srgb, var(--node-selected-border) 25%, transparent), var(--shadow-md)'
          : isRunning
            ? undefined
            : isHovered
              ? 'var(--shadow-lg)'
              : stateColor.glow !== 'none'
                ? stateColor.glow
                : 'var(--shadow-sm)',
        transition: 'transform 200ms ease, box-shadow 200ms ease',
        transform: isHovered ? 'scale(1.02)' : 'scale(1)',
        ...(isRunning && glowColor ? ({ '--glow-color': glowColor } as React.CSSProperties) : {}),
      }}
      onMouseEnter={() => setIsHovered(true)}
      onMouseLeave={() => setIsHovered(false)}
    >
      {/* In Handle */}
      {data.nodeType !== 'start' && (
        <Handle
          type="target"
          position={Position.Top}
          className="!w-3 !h-3 !border-2 !transition-transform !duration-200 hover:!scale-150 dag-handle"
          style={{
            background: 'var(--node-bg)',
            borderColor: accentColor || 'var(--border-default)',
            ...(accentColor ? ({ '--handle-accent': accentColor } as React.CSSProperties) : {}),
          }}
        />
      )}

      {/* Accent stripe */}
      <div
        className="h-1"
        style={{
          background: `linear-gradient(90deg, ${accentColor || 'var(--accent-primary)'} 0%, ${accentColor ? accentColor + '30' : 'var(--accent-subtle)'} 100%)`,
        }}
      />

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
            <div className="text-sm font-semibold truncate" style={{ color: 'var(--text-primary)' }}>
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
          className="!w-3 !h-3 !border-2 !transition-transform !duration-200 hover:!scale-150 dag-handle"
          style={{
            background: 'var(--node-bg)',
            borderColor: accentColor || 'var(--border-default)',
            ...(accentColor ? ({ '--handle-accent': accentColor } as React.CSSProperties) : {}),
          }}
        />
      )}
    </div>
  );
}

export default memo(BaseNode);
