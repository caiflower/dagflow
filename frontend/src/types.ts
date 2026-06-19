// DAGFlow 类型定义

export interface Flow {
  id: number;
  name: string;
  description: string;
  nodesJSON: string;
  edgesJSON: string;
  version: number;
  status: number;
  createTime: string;
  updateTime: string;
}

export interface FlowNode {
  id: string;
  name: string;
  type: 'task' | 'branch' | 'start' | 'end';
  protocol: string;
  config: Record<string, unknown>;
  position?: { x: number; y: number };
}

export interface FlowEdge {
  id: string;
  source: string;
  target: string;
  type: 'control' | 'data' | 'control+data';
  expr?: string;
}

export interface Protocol {
  name: string;
  displayName: string;
  description: string;
  configSchema: {
    fields: ConfigField[];
  };
}

export interface ConfigField {
  name: string;
  label: string;
  type: 'string' | 'number' | 'boolean' | 'select' | 'textarea';
  required: boolean;
  default?: string;
  description?: string;
  options?: string[];
}

export interface NodeInput {
  nodeName: string;
  input: string;
}

export interface Execution {
  id: string;
  flowID: number;
  flowName: string;
  state: 'pending' | 'running' | 'succeeded' | 'failed' | 'skipped';
  startTime: string;
  endTime: string;
  nodes: NodeStatus[];
}

export interface NodeStatus {
  id: string;
  name: string;
  state: string;
}

export interface PageResult<T> {
  items: T[];
  total: number;
}
