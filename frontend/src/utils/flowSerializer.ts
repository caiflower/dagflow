/**
 * Flow ↔ ReactFlow 节点/边双向转换
 */
import type { Node, Edge } from '@xyflow/react';
import type { FlowNode, FlowEdge } from '../types';
import { getNodeTypeColor, getEdgeColor, getProtocolIcon } from './stateColor';

// ===== Flow → ReactFlow =====

/** FlowNode[] → ReactFlow Node[] */
export function flowNodesToReactFlow(nodes: FlowNode[], nodeStates?: Record<string, string>): Node[] {
  return nodes.map((fn) => {
    const colors = getNodeTypeColor(fn.type);
    const state = nodeStates?.[fn.id];
    return {
      id: fn.id,
      type: fn.type, // custom node type
      position: fn.position || { x: 100, y: 100 },
      data: {
        label: fn.name,
        nodeType: fn.type,
        protocol: fn.protocol,
        config: fn.config,
        state: state || 'pending',
        icon: getProtocolIcon(fn.protocol),
      },
      style: {
        background: colors.bg,
        border: `2px solid ${colors.border}`,
        borderRadius: '8px',
        color: colors.text,
        padding: '0',
        fontSize: '13px',
        minWidth: '140px',
      },
    };
  });
}

/** FlowEdge[] → ReactFlow Edge[] */
export function flowEdgesToReactFlow(edges: FlowEdge[]): Edge[] {
  return edges.map((fe) => {
    const colors = getEdgeColor(fe.type);
    return {
      id: fe.id,
      source: fe.source,
      target: fe.target,
      type: fe.type === 'data' ? 'data' : fe.type === 'control+data' ? 'mixed' : 'control',
      label: colors.label,
      style: { stroke: colors.stroke, strokeWidth: 2 },
      animated: fe.type === 'data',
      data: { edgeType: fe.type, expr: fe.expr },
    };
  });
}

// ===== ReactFlow → Flow =====

/** ReactFlow Node[] → FlowNode[] */
export function reactFlowToFlowNodes(nodes: Node[]): FlowNode[] {
  return nodes.map((n) => ({
    id: n.id,
    name: (n.data as { label?: string })?.label || n.id,
    type: ((n.data as { nodeType?: string })?.nodeType || n.type || 'task') as FlowNode['type'],
    protocol: (n.data as { protocol?: string })?.protocol || '',
    config: (n.data as { config?: Record<string, unknown> })?.config || {},
    position: { x: n.position.x, y: n.position.y },
  }));
}

/** ReactFlow Edge[] → FlowEdge[] */
export function reactFlowToFlowEdges(edges: Edge[]): FlowEdge[] {
  return edges.map((e) => ({
    id: e.id,
    source: e.source,
    target: e.target,
    type: ((e.data as { edgeType?: string })?.edgeType || 'control') as FlowEdge['type'],
    expr: (e.data as { expr?: string })?.expr,
  }));
}

// ===== JSON 序列化 =====

/** 从 Flow JSON 字符串解析节点 */
export function parseNodesJSON(json: string): FlowNode[] {
  try { return json ? JSON.parse(json) : []; } catch { return []; }
}

/** 从 Flow JSON 字符串解析边 */
export function parseEdgesJSON(json: string): FlowEdge[] {
  try { return json ? JSON.parse(json) : []; } catch { return []; }
}

/** 序列化节点为 JSON 字符串 */
export function serializeNodes(nodes: FlowNode[]): string {
  return JSON.stringify(nodes, null, 2);
}

/** 序列化边为 JSON 字符串 */
export function serializeEdges(edges: FlowEdge[]): string {
  return JSON.stringify(edges, null, 2);
}
