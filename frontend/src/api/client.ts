import axios from 'axios';
import type { Flow, FlowNode, FlowEdge, Protocol, Execution, PageResult } from '../types';

const api = axios.create({
  baseURL: '/api/v1',
  timeout: 30000,
});

// Flow API
export const flowApi = {
  list: (page = 1, pageSize = 20, name = '') =>
    api.get<PageResult<Flow>>('/flows', { params: { page, pageSize, name } }).then(r => r.data),
  get: (id: number) =>
    api.get<Flow>(`/flows/${id}`).then(r => r.data),
  create: (data: { name: string; description: string; nodes: FlowNode[]; edges: FlowEdge[] }) =>
    api.post<Flow>('/flows', data).then(r => r.data),
  update: (data: { id: number; name?: string; description?: string; nodes?: FlowNode[]; edges?: FlowEdge[] }) =>
    api.put<Flow>(`/flows/${data.id}`, data).then(r => r.data),
  delete: (id: number) =>
    api.delete(`/flows/${id}`),
  validate: (nodes: FlowNode[], edges: FlowEdge[]) =>
    api.post<{ valid: boolean; error?: string }>('/flows/validate', { nodes, edges }).then(r => r.data),
};

// Protocol API
export const protocolApi = {
  list: () => api.get<Protocol[]>('/protocols').then(r => r.data),
  get: (name: string) => api.get<Protocol>(`/protocols/${name}`).then(r => r.data),
};

// Execution API
export const executionApi = {
  run: (flowId: number) =>
    api.post<Execution>('/executions/run', { flowId }).then(r => r.data),
  get: (id: string) =>
    api.get<Execution>(`/executions/${id}`).then(r => r.data),
  list: () =>
    api.get<Execution[]>('/executions').then(r => r.data),
};

export default api;
