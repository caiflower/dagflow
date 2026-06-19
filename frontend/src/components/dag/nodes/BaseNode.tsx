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
  /** 节点图标 */
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
      className="relative min-w-[150px] rounded-lg shadow-md overflow-hidden transition-all cursor-pointer"
      style={{
        background: stateColor.bg,
        border: `2px solid ${isSelected ? '#60a5fa' : stateColor.border}`,
        boxShadow: isSelected
          ? '0 0 0 2px rgba(96, 165, 250, 0.4), 0 4px 12px rgba(0,0,0,0.3)'
          : stateColor.glow !== 'none'
            ? stateColor.glow
            : undefined,
      }}
    >
      {/* 入边 Handle */}
      {data.nodeType !== 'start' && (
        <Handle
          type="target"
          position={Position.Top}
          className="!w-3 !h-3 !bg-gray-500 !border-2 !border-gray-400"
        />
      )}

      {/* 顶部装饰条 */}
      <div className="h-1" style={{ background: accentColor || '#3b82f6' }} />

      {/* 主体 */}
      <div className="px-3 py-2">
        <div className="flex items-center gap-2">
          {icon && <span className="text-base flex-shrink-0">{icon}</span>}
          <div className="flex-1 min-w-0">
            <div className="text-sm font-medium text-white truncate">{data.label}</div>
            <div className="flex items-center gap-1.5 mt-0.5">
              {data.protocol && (
                <span className="text-[10px] px-1.5 py-0.5 rounded bg-white/10 text-gray-300">
                  {data.protocol}
                </span>
              )}
              {state !== 'pending' && (
                <span
                  className="text-[10px] px-1.5 py-0.5 rounded"
                  style={{ background: stateColor.border + '30', color: stateColor.border }}
                >
                  {state}
                </span>
              )}
            </div>
          </div>
        </div>
        {extra && <div className="mt-2 pt-2 border-t border-white/10">{extra}</div>}
      </div>

      {/* 出边 Handle */}
      {data.nodeType !== 'end' && (
        <Handle
          type="source"
          position={Position.Bottom}
          className="!w-3 !h-3 !bg-gray-500 !border-2 !border-gray-400"
        />
      )}
    </div>
  );
}

export default memo(BaseNode);
