import { create } from 'zustand';
import type { Flow, FlowNode, FlowEdge, Protocol, Execution, NodeInput } from '../types';
import { flowApi, protocolApi, executionApi } from '../api/client';

interface FlowStore {
  flows: Flow[];
  total: number;
  currentFlow: Flow | null;
  loading: boolean;
  loadFlows: (page?: number, name?: string) => Promise<void>;
  loadFlow: (id: number) => Promise<void>;
  createFlow: (data: { name: string; description: string; nodes: FlowNode[]; edges: FlowEdge[] }) => Promise<Flow>;
  deleteFlow: (id: number) => Promise<void>;
  setCurrentFlow: (flow: Flow | null) => void;
}

export const useFlowStore = create<FlowStore>((set) => ({
  flows: [],
  total: 0,
  currentFlow: null,
  loading: false,

  loadFlows: async (page = 1, name = '') => {
    set({ loading: true });
    try {
      const result = await flowApi.list(page, 20, name);
      set({ flows: result.items || [], total: result.total || 0 });
    } finally {
      set({ loading: false });
    }
  },

  loadFlow: async (id: number) => {
    set({ loading: true });
    try {
      const flow = await flowApi.get(id);
      set({ currentFlow: flow });
    } finally {
      set({ loading: false });
    }
  },

  createFlow: async (data) => {
    const flow = await flowApi.create(data);
    set((s) => ({ flows: [flow, ...s.flows] }));
    return flow;
  },

  deleteFlow: async (id: number) => {
    await flowApi.delete(id);
    set((s) => ({ flows: s.flows.filter((f) => f.id !== id) }));
  },

  setCurrentFlow: (flow) => set({ currentFlow: flow }),
}));

interface ProtocolStore {
  protocols: Protocol[];
  loadProtocols: () => Promise<void>;
}

export const useProtocolStore = create<ProtocolStore>((set) => ({
  protocols: [],
  loadProtocols: async () => {
    const protocols = await protocolApi.list();
    set({ protocols });
  },
}));

interface ExecutionStore {
  executions: Execution[];
  currentExec: Execution | null;
  loading: boolean;
  runFlow: (flowId: number, nodeInputs?: NodeInput[]) => Promise<Execution>;
  pollExecution: (id: string) => Promise<void>;
  loadExecutions: () => Promise<void>;
}

export const useExecutionStore = create<ExecutionStore>((set, get) => ({
  executions: [],
  currentExec: null,
  loading: false,

  runFlow: async (flowId: number, nodeInputs?: NodeInput[]) => {
    const exec = await executionApi.run(flowId, nodeInputs);
    set((s) => ({ executions: [exec, ...s.executions], currentExec: exec }));
    // Start polling
    get().pollExecution(exec.id);
    return exec;
  },

  pollExecution: async (id: string) => {
    const poll = async () => {
      const exec = await executionApi.get(id);
      set((s) => ({
        currentExec: exec,
        // 同步更新列表中的对应记录
        executions: s.executions.map((e) => (e.id === id ? exec : e)),
      }));
      if (exec.state === 'running' || exec.state === 'pending') {
        setTimeout(poll, 2000);
      }
    };
    setTimeout(poll, 1000);
  },

  loadExecutions: async () => {
    set({ loading: true });
    try {
      const executions = await executionApi.list();
      set({ executions: executions || [] });
    } finally {
      set({ loading: false });
    }
  },
}));
