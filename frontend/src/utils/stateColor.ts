/**
 * 节点/边状态 → 颜色映射
 */

/** 节点类型颜色 */
export const nodeTypeColorMap: Record<string, { bg: string; border: string; text: string }> = {
  start:  { bg: '#065f46', border: '#22c55e', text: '#d1fae5' },
  end:    { bg: '#7f1d1d', border: '#ef4444', text: '#fecaca' },
  task:   { bg: '#1e3a5f', border: '#3b82f6', text: '#bfdbfe' },
  branch: { bg: '#78350f', border: '#f59e0b', text: '#fef3c7' },
};

/** 节点运行状态颜色 */
export const nodeStateColorMap: Record<string, { bg: string; border: string; glow: string }> = {
  pending:   { bg: '#374151', border: '#6b7280', glow: 'none' },
  running:   { bg: '#1e3a5f', border: '#3b82f6', glow: '0 0 8px #3b82f680' },
  succeeded: { bg: '#065f46', border: '#22c55e', glow: '0 0 8px #22c55e80' },
  failed:    { bg: '#7f1d1d', border: '#ef4444', glow: '0 0 8px #ef444480' },
  skipped:   { bg: '#374151', border: '#4b5563', glow: 'none' },
};

/** 边类型颜色 */
export const edgeColorMap: Record<string, { stroke: string; label?: string }> = {
  control:        { stroke: '#6b7280' },
  data:           { stroke: '#f59e0b', label: 'data' },
  'control+data': { stroke: '#a78bfa', label: 'ctrl+data' },
};

/** 获取节点类型颜色（带 fallback） */
export function getNodeTypeColor(type: string) {
  return nodeTypeColorMap[type] || nodeTypeColorMap.task;
}

/** 获取节点运行状态颜色（带 fallback） */
export function getNodeStateColor(state: string) {
  return nodeStateColorMap[state] || nodeStateColorMap.pending;
}

/** 获取边颜色（带 fallback） */
export function getEdgeColor(type: string) {
  return edgeColorMap[type] || edgeColorMap.control;
}

/** 协议图标映射 */
export const protocolIconMap: Record<string, string> = {
  http:  '🌐',
  grpc:  '📡',
  local: '💻',
  mcp:   '🔗',
};

/** 获取协议图标（带 fallback） */
export function getProtocolIcon(protocol: string) {
  return protocolIconMap[protocol] || '⚙️';
}
