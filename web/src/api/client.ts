import { Execution, DLQItem, StepAuditEntry, HealingLogEntry, StateSnapshot, WorkflowSummary, WorkflowSpec, ScheduledJob, ComplianceReport } from '../types';

const BASE_URL = '/api';

export const api = {
  // Health
  getHealth: async (): Promise<{ status: string; version: string }> => {
    const res = await fetch(`${BASE_URL}/health`);
    return res.json();
  },

  // Workflows
  listWorkflows: async (): Promise<WorkflowSummary[]> => {
    const res = await fetch(`${BASE_URL}/workflows`);
    return res.json();
  },

  getWorkflow: async (id: string): Promise<{ spec: WorkflowSpec; raw_yaml: string }> => {
    const res = await fetch(`${BASE_URL}/workflows/${id}`);
    if (!res.ok) throw new Error('Workflow not found');
    return res.json();
  },

  runWorkflow: async (id: string, input: Record<string, any>, asyncRun = false): Promise<Execution> => {
    const res = await fetch(`${BASE_URL}/workflows/${id}/run`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ input, async: asyncRun }),
    });
    if (!res.ok) {
      const err = await res.json();
      throw new Error(err.error || 'Failed to start workflow');
    }
    return res.json();
  },

  // Executions
  listExecutions: async (status?: string, limit = 50): Promise<Execution[]> => {
    const params = new URLSearchParams();
    if (status) params.append('status', status);
    params.append('limit', limit.toString());
    const res = await fetch(`${BASE_URL}/executions?${params.toString()}`);
    if (!res.ok) return [];
    const data = await res.json();
    return Array.isArray(data) ? data : [];
  },

  getExecution: async (id: string): Promise<Execution> => {
    const res = await fetch(`${BASE_URL}/executions/${id}`);
    if (!res.ok) throw new Error('Execution not found');
    return res.json();
  },

  getAuditLogs: async (executionId: string): Promise<StepAuditEntry[]> => {
    const res = await fetch(`${BASE_URL}/executions/${executionId}/audit`);
    if (!res.ok) return [];
    const data = await res.json();
    return Array.isArray(data) ? data : [];
  },

  getHealingLogs: async (executionId: string): Promise<HealingLogEntry[]> => {
    const res = await fetch(`${BASE_URL}/executions/${executionId}/healing`);
    if (!res.ok) return [];
    const data = await res.json();
    return Array.isArray(data) ? data : [];
  },

  getSnapshots: async (executionId: string): Promise<StateSnapshot[]> => {
    const res = await fetch(`${BASE_URL}/executions/${executionId}/snapshots`);
    if (!res.ok) return [];
    const data = await res.json();
    return Array.isArray(data) ? data : [];
  },

  rewindExecution: async (executionId: string, targetNodeId: string): Promise<Execution> => {
    const res = await fetch(`${BASE_URL}/executions/${executionId}/rewind`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ target_node_id: targetNodeId }),
    });
    if (!res.ok) {
      const err = await res.json();
      throw new Error(err.error || 'Failed to rewind execution');
    }
    return res.json();
  },

  // Step-Through Debugger & Breakpoints
  stepExecution: async (executionId: string): Promise<Execution> => {
    const res = await fetch(`${BASE_URL}/executions/${executionId}/step`, {
      method: 'POST',
    });
    if (!res.ok) {
      const err = await res.json();
      throw new Error(err.error || 'Failed to step execution');
    }
    return res.json();
  },

  continueExecution: async (executionId: string): Promise<Execution> => {
    const res = await fetch(`${BASE_URL}/executions/${executionId}/continue`, {
      method: 'POST',
    });
    if (!res.ok) {
      const err = await res.json();
      throw new Error(err.error || 'Failed to continue execution');
    }
    return res.json();
  },

  setBreakpoints: async (executionId: string, breakpoints: string[]): Promise<Execution> => {
    const res = await fetch(`${BASE_URL}/executions/${executionId}/breakpoints`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ breakpoints }),
    });
    if (!res.ok) {
      const err = await res.json();
      throw new Error(err.error || 'Failed to set breakpoints');
    }
    return res.json();
  },

  // Compliance Audit Report
  getComplianceReport: async (executionId: string, format: 'json' | 'markdown' = 'json'): Promise<any> => {
    const res = await fetch(`${BASE_URL}/executions/${executionId}/report?format=${format}`, {
      headers: {
        Accept: format === 'markdown' ? 'text/markdown' : 'application/json',
      },
    });
    if (!res.ok) throw new Error('Failed to fetch compliance report');
    if (format === 'markdown') {
      return res.text();
    }
    return res.json() as Promise<ComplianceReport>;
  },

  // Cron Scheduler
  listCronJobs: async (): Promise<ScheduledJob[]> => {
    const res = await fetch(`${BASE_URL}/cron/jobs`);
    if (!res.ok) return [];
    return res.json();
  },

  // Dead-Letter Queue
  listDLQItems: async (status?: string): Promise<DLQItem[]> => {
    const params = new URLSearchParams();
    if (status) params.append('status', status);
    const res = await fetch(`${BASE_URL}/dlq?${params.toString()}`);
    if (!res.ok) return [];
    const data = await res.json();
    return Array.isArray(data) ? data : [];
  },

  getDLQItem: async (id: string): Promise<DLQItem> => {
    const res = await fetch(`${BASE_URL}/dlq/${id}`);
    if (!res.ok) throw new Error('DLQ item not found');
    return res.json();
  },

  resumeDLQ: async (
    id: string,
    options: { operator_instruction?: string; patch?: Record<string, any>; retry_current_node?: boolean }
  ): Promise<Execution> => {
    const res = await fetch(`${BASE_URL}/dlq/${id}/resume`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(options),
    });
    if (!res.ok) {
      const err = await res.json();
      throw new Error(err.error || 'Failed to resume execution');
    }
    return res.json();
  },

  abortDLQ: async (id: string, reason?: string): Promise<{ message: string }> => {
    const res = await fetch(`${BASE_URL}/dlq/${id}/abort`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ reason }),
    });
    return res.json();
  },
};

export function subscribeToEvents(
  onEvent: (event: any) => void,
  executionId?: string
): () => void {
  const url = executionId ? `${BASE_URL}/executions/${executionId}/events` : `${BASE_URL}/events`;
  const eventSource = new EventSource(url);

  eventSource.onmessage = (e) => {
    try {
      const data = JSON.parse(e.data);
      onEvent(data);
    } catch {
      // Ignored
    }
  };

  const eventTypes = [
    'workflow_started',
    'node_start',
    'node_complete',
    'self_healing_attempt',
    'dlq_paused',
    'execution_resumed',
    'workflow_completed',
    'workflow_failed',
    'breakpoint_hit',
    'budget_exceeded',
    'model_cascaded',
  ];

  eventTypes.forEach((type) => {
    eventSource.addEventListener(type, (e: any) => {
      try {
        const data = JSON.parse(e.data);
        onEvent(data);
      } catch {
        // Ignored
      }
    });
  });

  return () => {
    eventSource.close();
  };
}
