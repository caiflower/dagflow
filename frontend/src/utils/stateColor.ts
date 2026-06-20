/**
 * 节点/边状态 → 颜色映射
 * 与 theme.css CSS variables 对齐
 */

/** 节点类型颜色 */
export const nodeTypeColorMap: Record<string, { bg: string; border: string; text: string }> = {
  start:  { bg: '#0a3a1a', border: '#34c759', text: '#d1fae5' },
  end:    { bg: '#3a1010', border: '#ff3b30', text: '#fecaca' },
  task:   { bg: '#0a2a4a', border: '#2997ff', text: '#bfdbfe' },
  branch: { bg: '#3a2a00', border: '#ff9f0a', text: '#fef3c7' },
};

/** 节点运行状态颜色 */
export const nodeStateColorMap: Record<string, { bg: string; border: string; glow: string }> = {
  pending:   { bg: '#1c1c1e', border: '#636366', glow: 'none' },
  running:   { bg: '#0a2a4a', border: '#2997ff', glow: '0 0 8px #2997ff80' },
  succeeded: { bg: '#0a3a1a', border: '#34c759', glow: '0 0 8px #34c75980' },
  failed:    { bg: '#3a1010', border: '#ff3b30', glow: '0 0 8px #ff3b3080' },
  skipped:   { bg: '#1c1c1e', border: '#545458', glow: 'none' },
};

/** 边类型颜色 */
export const edgeColorMap: Record<string, { stroke: string; label?: string }> = {
  control:        { stroke: '#636366' },
  data:           { stroke: '#ff9f0a', label: 'data' },
  'control+data': { stroke: '#bf5af2', label: 'ctrl+data' },
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

/** 状态 Badge 变体映射（对应 Badge 组件的 variant） */
export const stateBadgeVariant: Record<string, 'success' | 'danger' | 'warning' | 'info' | 'default'> = {
  pending:   'default',
  running:   'info',
  succeeded: 'success',
  failed:    'danger',
  skipped:   'default',
};

/** 启用/禁用 Badge 变体 */
export function statusBadgeVariant(status: number): 'success' | 'danger' {
  return status === 1 ? 'success' : 'danger';
}

/** 协议图标名称映射（返回 Lucide icon name） */
export const protocolIconMap: Record<string, string> = {
  http:  'Globe',
  grpc:  'Radio',
  local: 'Terminal',
  mcp:   'Link',
};

/** 获取协议图标名称（带 fallback） */
export function getProtocolIcon(protocol: string): string {
  return protocolIconMap[protocol] || 'Settings';
}
