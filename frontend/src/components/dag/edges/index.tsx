/**
 * 自定义边组件 — ControlEdge / DataEdge / MixedEdge
 */
import { memo } from 'react';
import {
  BaseEdge,
  getSmoothStepPath,
  type EdgeProps,
  type Edge,
} from '@xyflow/react';
import { getEdgeColor } from '../../../utils/stateColor';

// ===== ControlEdge — 灰色实线 =====
function ControlEdge({
  id, sourceX, sourceY, targetX, targetY,
  sourcePosition, targetPosition, style,
}: EdgeProps<Edge<{ edgeType?: string; expr?: string }>>) {
  const colors = getEdgeColor('control');
  const [edgePath] = getSmoothStepPath({
    sourceX, sourceY, targetX, targetY,
    sourcePosition, targetPosition,
    borderRadius: 8,
  });
  return (
    <BaseEdge
      id={id}
      path={edgePath}
      style={{ ...style, stroke: colors.stroke, strokeWidth: 2 }}
    />
  );
}

// ===== DataEdge — 黄色虚线动画 =====
function DataEdge({
  id, sourceX, sourceY, targetX, targetY,
  sourcePosition, targetPosition, style,
}: EdgeProps<Edge<{ edgeType?: string; expr?: string }>>) {
  const colors = getEdgeColor('data');
  const [edgePath] = getSmoothStepPath({
    sourceX, sourceY, targetX, targetY,
    sourcePosition, targetPosition,
    borderRadius: 12,
  });
  return (
    <BaseEdge
      id={id}
      path={edgePath}
      style={{
        ...style,
        stroke: colors.stroke,
        strokeWidth: 2,
        strokeDasharray: '6 3',
        animation: 'dashdraw 0.5s linear infinite',
      }}
    />
  );
}

// ===== MixedEdge — 紫色双线效果（实线 + 叠加虚线） =====
function MixedEdge({
  id, sourceX, sourceY, targetX, targetY,
  sourcePosition, targetPosition, style,
}: EdgeProps<Edge<{ edgeType?: string; expr?: string }>>) {
  const colors = getEdgeColor('control+data');
  const [edgePath] = getSmoothStepPath({
    sourceX, sourceY, targetX, targetY,
    sourcePosition, targetPosition,
    borderRadius: 10,
  });
  return (
    <>
      <BaseEdge
        id={id}
        path={edgePath}
        style={{ ...style, stroke: colors.stroke, strokeWidth: 3 }}
      />
      <BaseEdge
        id={`${id}-overlay`}
        path={edgePath}
        style={{
          ...style,
          stroke: '#f59e0b',
          strokeWidth: 1,
          strokeDasharray: '4 4',
          opacity: 0.6,
        }}
      />
    </>
  );
}

// ===== 导出 edgeTypes 注册对象 =====
export const edgeTypes = {
  control: memo(ControlEdge),
  data: memo(DataEdge),
  mixed: memo(MixedEdge),
};
