export type ExecutionStatus = 'PENDING' | 'RUNNING' | 'COMPLETED' | 'FAILED' | 'PAUSED_DLQ' | 'HALTED_BREAKPOINT';
export type NodeStatus = 'PENDING' | 'RUNNING' | 'SUCCESS' | 'HEALED' | 'FAILED' | 'PAUSED' | 'HALTED';
export type NodeType = 'llm_agent' | 'tool_execution' | 'conditional_router' | 'human_checkpoint' | 'transform';

export interface WorkflowMetadata {
  id: string;
  name: string;
  description?: string;
  timeout?: string;
  max_healing_attempts?: number;
  schedule?: string;
  webhook_secret?: string;
}

export interface NodeSpec {
  id: string;
  type: NodeType;
  description?: string;
  depends_on?: string[];
  next?: string;
  prompt?: string;
  input?: Record<string, any>;
  output_schema?: Record<string, any>;
  confidence_threshold?: number;
  tool?: string;
  params?: Record<string, any>;
  save_to?: string;
  breakpoint?: boolean;
}

export interface BudgetConfig {
  max_total_tokens?: number;
  max_cost_usd?: number;
  on_exceeded?: string;
}

export interface WorkflowSpec {
  version: string;
  workflow: WorkflowMetadata;
  budget?: BudgetConfig;
  nodes: NodeSpec[];
  state?: Record<string, any>;
}

export interface WorkflowSummary {
  id: string;
  name: string;
  node_count: number;
  schedule?: string;
  created_at: string;
  updated_at: string;
}

export interface Execution {
  id: string;
  workflow_id: string;
  status: ExecutionStatus;
  current_node_id: string;
  state: Record<string, any>;
  healing_attempts: number;
  total_tokens?: number;
  total_cost_usd?: number;
  breakpoints?: string[];
  is_stepping?: boolean;
  started_at: string;
  updated_at: string;
  completed_at?: string;
}

export interface DLQItem {
  id: string;
  execution_id: string;
  failed_node_id: string;
  failure_reason: string;
  error_details: string;
  snapshot_state_json: string;
  resolved_by?: string;
  resolution_action?: string;
  operator_instruction?: string;
  created_at: string;
  resolved_at?: string;
}

export interface StepAuditEntry {
  id: number;
  execution_id: string;
  node_id: string;
  node_type: NodeType;
  input_payload: string;
  output_payload: string;
  tokens_consumed: number;
  latency_ms: number;
  status: NodeStatus;
  timestamp: string;
}

export interface HealingLogEntry {
  id: number;
  execution_id: string;
  node_id: string;
  attempt_number: number;
  error_category: string;
  original_error: string;
  mutated_prompt: string;
  status: string;
  timestamp: string;
}

export interface StateSnapshot {
  id: number;
  execution_id: string;
  node_id: string;
  state_json: string;
  step_sequence: number;
  created_at: string;
}

export interface ExecutionEvent {
  execution_id: string;
  workflow_id: string;
  type: string;
  node_id?: string;
  status?: string;
  message?: string;
  payload?: any;
  timestamp: string;
}

export interface ScheduledJob {
  entry_id: number;
  workflow_id: string;
  schedule: string;
  next_run: string;
  prev_run: string;
}

export interface ComplianceReport {
  generated_at: string;
  summary: {
    execution_id: string;
    workflow_id: string;
    status: string;
    started_at: string;
    completed_at: string;
    duration_ms: number;
  };
  budget: {
    total_tokens: number;
    total_cost_usd: number;
    max_tokens_limit?: number;
    max_cost_limit_usd?: number;
    within_budget: boolean;
  };
  nodes: Array<{
    node_id: string;
    node_type: string;
    status: string;
    duration_ms: number;
    error?: string;
  }>;
  healing_events: Array<{
    id: string;
    node_id: string;
    error_category: string;
    strategy: string;
    attempt_number: number;
    resolved: boolean;
    created_at: string;
  }>;
  human_in_the_loop?: {
    dlq_id: string;
    failed_node_id: string;
    reason: string;
    status: string;
    frozen_at: string;
    resolved_at?: string;
    resolved_by?: string;
    action?: string;
    instruction?: string;
  };
}
