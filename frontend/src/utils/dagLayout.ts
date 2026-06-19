/**
 * DAG 自动布局算法 — 基于拓扑排序的层级布局
 */
import type { FlowNode, FlowEdge } from '../types';

interface LayoutOptions {
  /** 节点间水平间距 */
  horizontalGap?: number;
  /** 节点间垂直间距 */
  verticalGap?: number;
  /** 起始 X 偏移 */
  offsetX?: number;
  /** 起始 Y 偏移 */
  offsetY?: number;
  /** 节点宽度 */
  nodeWidth?: number;
  /** 节点高度 */
  nodeHeight?: number;
}

const defaultOpts: Required<LayoutOptions> = {
  horizontalGap: 80,
  verticalGap: 100,
  offsetX: 50,
  offsetY: 50,
  nodeWidth: 160,
  nodeHeight: 60,
};

/**
 * 对 DAG 节点进行拓扑排序层级布局
 * 返回带更新后 position 的节点数组
 */
export function autoLayout(nodes: FlowNode[], edges: FlowEdge[], opts?: LayoutOptions): FlowNode[] {
  const o = { ...defaultOpts, ...opts };
  if (nodes.length === 0) return [];

  // 1. 构建邻接表和入度
  const adj = new Map<string, string[]>();
  const inDeg = new Map<string, number>();
  for (const n of nodes) {
    adj.set(n.id, []);
    inDeg.set(n.id, 0);
  }
  for (const e of edges) {
    adj.get(e.source)?.push(e.target);
    inDeg.set(e.target, (inDeg.get(e.target) || 0) + 1);
  }

  // 2. BFS 拓扑排序 → 分层
  const levels: string[][] = [];
  const visited = new Set<string>();
  let queue = nodes.filter((n) => (inDeg.get(n.id) || 0) === 0).map((n) => n.id);

  while (queue.length > 0) {
    levels.push([...queue]);
    queue.forEach((id) => visited.add(id));
    const next: string[] = [];
    for (const id of queue) {
      for (const target of adj.get(id) || []) {
        inDeg.set(target, (inDeg.get(target) || 0) - 1);
        if ((inDeg.get(target) || 0) <= 0 && !visited.has(target) && !next.includes(target)) {
          next.push(target);
        }
      }
    }
    queue = next;
  }

  // 处理未访问的节点（存在环或孤立节点）
  for (const n of nodes) {
    if (!visited.has(n.id)) {
      levels.push([n.id]);
    }
  }

  // 3. 按层级分配坐标
  const posMap = new Map<string, { x: number; y: number }>();
  for (let level = 0; level < levels.length; level++) {
    const ids = levels[level];
    const startX = o.offsetX;

    for (let i = 0; i < ids.length; i++) {
      posMap.set(ids[i], {
        x: startX + i * (o.nodeWidth + o.horizontalGap),
        y: o.offsetY + level * (o.nodeHeight + o.verticalGap),
      });
    }
  }

  // 4. 返回更新后的节点
  return nodes.map((n) => ({
    ...n,
    position: posMap.get(n.id) || n.position || { x: o.offsetX, y: o.offsetY },
  }));
}

/**
 * 居中布局 — 让 DAG 在画布中心展开
 */
export function centerLayout(
  nodes: FlowNode[],
  edges: FlowEdge[],
  canvasWidth = 800,
  opts?: LayoutOptions,
): FlowNode[] {
  const laid = autoLayout(nodes, edges, opts);
  if (laid.length === 0) return [];

  // 计算包围盒
  let minX = Infinity, maxX = -Infinity;
  for (const n of laid) {
    minX = Math.min(minX, n.position!.x);
    maxX = Math.max(maxX, n.position!.x + (opts?.nodeWidth || defaultOpts.nodeWidth));
  }
  const dagWidth = maxX - minX;
  const shiftX = Math.max(0, (canvasWidth - dagWidth) / 2 - minX);

  return laid.map((n) => ({
    ...n,
    position: { x: n.position!.x + shiftX, y: n.position!.y },
  }));
}
