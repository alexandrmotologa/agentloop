# Dead-Letter Queues & Human-in-the-Loop Operations

## Purpose

When autonomous recovery cannot resolve an error, or when a workflow reaches an explicit policy checkpoint (e.g. financial transaction exceeding $5,000 or production deployment), agentloop freezes execution state in a Dead-Letter Queue (DLQ).

Rather than aborting the transaction and losing upstream work, the execution halts at the exact node, preserving complete state memory for human review and single-click resumption.

## Dead-Letter Queue Protocol

1. Snapshot Freezing: The current execution state, failed node identifier, failure reason, and full stack error details are serialized into `state_snapshots` and `dlq_items` tables in SQLite WAL storage.
2. Status Transition: The execution status transitions to `PAUSED_DLQ`.
3. Dispatcher Alerts: Escalation events are dispatched to configured destinations:
   - Console logs with `[ALERT: DLQ ESCALATION]`.
   - Webhook POST request containing incident metadata.
   - Slack channel message with node, reason, and error details.
4. Suspension: The execution loop terminates cleanly without holding background threads or database locks.

## Operator Review & Actions

Operators interact with suspended workflows through the embedded React 19 cockpit or the CLI:

### 1. State Inspection & Editing
Operators inspect the exact memory variables at failure. The built-in JSON state editor allows direct editing of values (for instance, correcting an unreadable tax ID or adjusting a price field).

### 2. Natural Language Steering
Operators can provide plain-language instructions (e.g. "Tax ID was confirmed valid with corporate registry"). The instruction is injected directly into the LLM node context as an `[Operator Directive]` upon resumption.

### 3. Resolution Choices
- Resume & Advance: Applies any state patches and continues traversal to the subsequent node in the DAG.
- Retry Failed Node: Retries the failed node using the updated state and operator guidance.
- Abort: Gracefully marks the execution as failed and records the operator explanation in the compliance audit trail.

## Time-Travel State Rewind

agentloop records state snapshots at each node transition. Operators can rewind an execution back to any previous node's snapshot state, modify parameters, and re-execute from that point forward.
