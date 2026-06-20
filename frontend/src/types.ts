// DAGFlow 类型定义

export interface Flow {
  id: number;
  name: string;
  description: string;
  nodes_json: string;
  edges_json: string;
  version: number;
  status: number;
  create_time: string;
  update_time: string;
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
  display_name: string;
  description: string;
  config_schema: {
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
  node_name: string;
  input: string;
}

export interface Execution {
  id: string;
  flow_id: number;
  flow_name: string;
  state: 'pending' | 'running' | 'succeeded' | 'failed' | 'skipped' | 'archived';
  start_time: string;
  end_time: string;
  nodes: NodeStatus[];
  task_id: string;
}

export interface NodeStatus {
  id: string;
  name: string;
  state: string;
  input: string;
  output: string;
  start_time: string;
  end_time: string;
  duration_ms: number;
  node_type: string;
  protocol: string;
}

export interface PageResult<T> {
  items: T[];
  total: number;
}
