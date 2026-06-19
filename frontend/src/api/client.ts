import axios from 'axios';
import type { Flow, FlowNode, FlowEdge, Protocol, Execution, PageResult } from '../types';

const api = axios.create({
  baseURL: '/api/v1',
  timeout: 30000,
});

/** 解包 GRPC 响应: {requestID, data: <payload>} → payload */
function unwrap<T>(r: { data: { data: T } }): T {
  return r.data.data;
}

/** 前端 FlowNode → Proto FlowNode 格式 */
function toProtoNode(n: FlowNode) {
  return {
    id: n.id,
    name: n.name,
    type: n.type,
    protocol: n.protocol,
    configJSON: n.config ? JSON.stringify(n.config) : '',
    positionX: n.position?.x ?? 0,
    positionY: n.position?.y ?? 0,
  };
}

function toProtoEdge(e: FlowEdge) {
  return { id: e.id, source: e.source, target: e.target, type: e.type, expr: e.expr };
}

// Flow API
export const flowApi = {
  list: (page = 1, pageSize = 20, name = '') =>
    api.get('/flows', { params: { page, pageSize, name } }).then(r => unwrap<PageResult<Flow>>(r)),
  get: (id: number) =>
    api.get(`/flows/${id}`).then(r => unwrap<{ flow: Flow }>(r).flow),
  create: (data: { name: string; description: string; nodes: FlowNode[]; edges: FlowEdge[] }) =>
    api.post('/flows', {
      ...data,
      nodes: data.nodes.map(toProtoNode),
      edges: data.edges.map(toProtoEdge),
    }).then(r => unwrap<{ flow: Flow }>(r).flow),
  update: (data: { id: number; name?: string; description?: string; nodes?: FlowNode[]; edges?: FlowEdge[] }) =>
    api.put(`/flows/${data.id}`, {
      ...data,
      nodes: data.nodes?.map(toProtoNode),
      edges: data.edges?.map(toProtoEdge),
    }).then(r => unwrap<{ flow: Flow }>(r).flow),
  delete: (id: number) =>
    api.delete(`/flows/${id}`).then(r => unwrap<{ status: string }>(r)),
  validate: (nodes: FlowNode[], edges: FlowEdge[]) =>
    api.post('/flows/validate', {
      nodes: nodes.map(toProtoNode),
      edges: edges.map(toProtoEdge),
    }).then(r => unwrap<{ valid: boolean; error?: string }>(r)),
};

// Protocol API
export const protocolApi = {
  list: () =>
    api.get('/protocols').then(r => unwrap<{ items: Protocol[] }>(r).items),
  get: (name: string) =>
    api.get(`/protocols/${name}`).then(r => unwrap<{ protocol: Protocol }>(r).protocol),
};

// Execution API
export const executionApi = {
  run: (flowId: number) =>
    api.post('/executions/run', { flowId }).then(r => unwrap<{ execution: Execution }>(r).execution),
  get: (id: string) =>
    api.get(`/executions/${id}`).then(r => unwrap<{ execution: Execution }>(r).execution),
  list: () =>
    api.get('/executions').then(r => unwrap<{ items: Execution[] }>(r).items),
};

export default api;
