# System Architecture

## Overview

agentloop treats AI agent execution as a deterministic state machine instead of an unconstrained loop. The engine is written in Go, compiles to a single binary, and embeds a React 19 cockpit for human supervision. It combines deterministic pipeline operations (HTTP requests, database queries, expressions) with probabilistic LLM reasoning steps.

```
+-----------------------------------------------------------------+
|                       Enterprise Triggers                       |
|          Webhooks, Cron Schedules, Message Queues, CLI          |
+-----------------------------------------------------------------+
                                |
                                v
+-----------------------------------------------------------------+
|                      Workflow DAG Engine                        |
|   Topological Sort, Cycle Detection, Context & State Store      |
+-----------------------------------------------------------------+
          |                                             |
          v                                             v
+-------------------------------+             +-------------------------------+
|      Deterministic Tools      |             |         LLM Agent Node        |
| - HTTP REST / GraphQL         |             | - Prompt Interpolation        |
| - SQLite / SQL Queries        |             | - Multi-Provider Dispatch     |
| - JSON Expression Transforms  |             | - Structured JSON Extraction  |
+-------------------------------+             +-------------------------------+
          |                                             |
          +----------------------+----------------------+
                                 |
                                 v
              +-------------------------------------+
              |         Execution Sentinels         |
              | - JSON Schema Validation            |
              | - Token Confidence Scoring          |
              | - Execution Timeout & Cost Budget   |
              +-------------------------------------+
                                 |
               +-----------------+-----------------+
               | Passed                            | Failed
               v                                   v
+-------------------------------+   +-------------------------------+
| Transition to Next DAG Node   |   |      Self-Healing Loop        |
+-------------------------------+   | - Error Classification        |
                                    | - Dynamic Prompt Mutation     |
                                    | - Exponential Backoff         |
                                    +-------------------------------+
                                                   |
                                 +-----------------+-----------------+
                                 | Recovered <= N    | Exhausted > N
                                 v                   v
                  +---------------------+   +-----------------------+
                  | Resume DAG Traversal|   | Dead-Letter Queue     |
                  +---------------------+   | - State Snapshot      |
                                            | - Escalation Alert    |
                                            | - Operator Console    |
                                            +-----------------------+
```

## Core Components

### 1. Workflow DAG Engine (`internal/engine/dag.go`)
Workflows are defined declaratively in YAML. The engine validates the graph structure on load using Kahn's topological sort algorithm. If circular dependencies exist, the parser rejects the workflow before any execution begins.

Branching is handled through:
- Sequential edges: Default progression when no router is present.
- Explicit `depends_on`: Declaring prerequisite completion.
- Explicit `next`: Direct jumps between nodes.
- Conditional routers: Evaluating expressions against the state map to choose branch targets.

### 2. Transactional Memory State (`internal/engine/state.go`)
Every workflow execution maintains a thread-safe state container. Variable access supports dot notation (such as `extracted_data.total_amount`). State mutations occur atomically, and snapshots are recorded before and after each node transition to enable state rollback and compliance auditing.

### 3. Execution Sentinels & Confidence Scorer (`internal/llm/`)
Probabilistic nodes cannot advance without passing validation sentinels:
- Schema Sentinel: Validates output payloads against Draft 2020 JSON Schema definitions.
- Confidence Sentinel: Computes a score based on semantic markers, hesitation phrases, and validation results. When confidence falls below the node threshold, the sentinel triggers either self-healing or an immediate pause to the Dead-Letter Queue.

### 4. Self-Healing Subsystem (`internal/healing/`)
When a failure occurs, the engine categorizes the root cause:
- `SCHEMA_MISMATCH`: Missing required keys or invalid field types.
- `JSON_SYNTAX_ERROR`: Truncated outputs or conversational wrapper formatting.
- `LOW_CONFIDENCE`: Scores below the configured threshold.
- `NETWORK_TRANSIENT`: HTTP 429, 502, 503, or 504 responses.
- `TOOL_ERROR`: Query execution or command errors.

For LLM failures, the reflection engine synthesizes a surgical repair prompt that describes the exact schema failure and preserves valid prior data.

### 5. Dead-Letter Queue & HITL Cockpit (`internal/dlq/`, `web/`)
When retries are exhausted or a human approval gate is reached, execution freezes. The complete memory state, failure reason, and stack trace are stored in SQLite WAL mode. Escalation alerts are sent to webhooks or Slack. Operators can review the incident in the React 19 cockpit, edit state variables, supply guidance, and resume execution with one click.
