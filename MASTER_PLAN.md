# agentloop - Architecture & Engineering Master Plan
> Self-Healing Autonomous Enterprise Workflow Engine with Dead-Letter Queues & Human-in-the-Loop
> Target: High-Reliability Enterprise AI Automation, Sub-25ms Engine Overhead, Zero Data Loss

---

## 1. Executive Summary & Market Problem

### 1.1 The Enterprise AI Automation Crisis
Enterprises are moving from single-turn chat interfaces to autonomous multi-step AI agent workflows such as automated invoice reconciliation, security incident remediation, customer KYC validation, and multi-system data synchronization. However, more than 85% of enterprise AI agent deployments encounter failures or stall in pilot stages.

The primary root causes in production:
1. Brittle Tool Execution: Third-party APIs change field names, return 502 Bad Gateway responses, or timeout. Naive agents enter unbounded retry loops or hallucinate non-existent parameters.
2. Infinite Hallucination Loops: When an agent receives an unexpected tool output, it often repeats the same tool call with minor variations until token limits or cost budgets run out.
3. Lack of State Persistence and Rollback: If a multi-step business workflow fails at Step 6, the entire transaction is typically lost. The system cannot rewind to Step 5, repair the context, and resume.
4. Absence of Human-in-the-Loop (HITL) Safety Nets: Without graceful degradation, an uncertain agent either makes an unverified autonomous choice or crashes without an escalation path.

### 1.2 The agentloop Solution
agentloop is a single-binary enterprise workflow engine written in Go with an embedded React 19 cockpit. It treats AI agent execution as a deterministic, recoverable state machine:
- Deterministic Workflow DAG: Combines deterministic business logic (guards, conditions, API calls, SQL queries) with probabilistic LLM reasoning nodes.
- Autonomous Self-Healing Loop: When a tool or structured extraction fails, an internal reflection sub-agent analyzes the error, inspects schema definitions, and synthesizes a surgical repair prompt without discarding prior context.
- Dead-Letter Queue (DLQ) and Checkpoint Freezing: When self-healing attempts exceed configured bounds, execution pauses at the exact node, preserving the full memory state and audit trail in SQLite WAL storage.
- Visual Human-in-the-Loop Console: An operator inspects the live DAG, reviews self-healing attempts, edits state variables, provides plain-language instructions, and resumes execution with one click.
- State Rewind and Time-Travel: Allows operators to roll back execution state to any prior node snapshot.

---

## 2. High-Level System Architecture

```
                                  +---------------------------------------+
                                  |         Enterprise Trigger            |
                                  | (Webhook, Cron, Queue, Manual CLI/UI) |
                                  +---------------------------------------+
                                                      |
                                                      v
                                  +---------------------------------------+
                                  |         agentloop Orchestrator        |
                                  |   (State Machine Engine, Go Binary)   |
                                  +---------------------------------------+
                                                      |
                         +----------------------------+----------------------------+
                         |                                                         |
                         v                                                         v
         +-------------------------------+                         +-------------------------------+
         |    Deterministic Step Nodes   |                         |      Agentic LLM Nodes        |
         |  - HTTP REST / GraphQL        |                         |  - Reasoning & Planning       |
         |  - SQLite / SQL Query         |                         |  - Multimodal Extraction      |
         |  - Data Transform (Expr/JSON) |                         |  - Structured Tool Calling    |
         |  - Sandboxed File I/O         |                         |  - Schema-Enforced Generation |
         +-------------------------------+                         +-------------------------------+
                         |                                                         |
                         +----------------------------+----------------------------+
                                                      |
                                                      v
                                       +------------------------------+
                                       |      Execution Sentinel      |
                                       | - Schema Validation (JSON)   |
                                       | - Confidence Scoring (<0.85) |
                                       | - Execution Budget / Timeout |
                                       +------------------------------+
                                                      |
                                   +------------------+------------------+
                                   | (Success)                           | (Failure / Low Confidence)
                                   v                                     v
                       +----------------------+          +------------------------------+
                       | Transition Next Node |          |     Self-Healing Engine      |
                       +----------------------+          | - Error Classification       |
                                                         | - Dynamic Prompt Mutation    |
                                                         | - Fallback Tool Selection    |
                                                         | - Exponential Backoff        |
                                                         +------------------------------+
                                                                         |
                                                      +------------------+------------------+
                                                      | (Recovered <= N)                    | (Exhausted > N)
                                                      v                                     v
                                          +----------------------+          +------------------------------+
                                          | Resume Execution DAG |          |    Dead-Letter Queue (DLQ)   |
                                          +----------------------+          | - State Freeze & Snapshot    |
                                                                            | - Alerting (Slack/Teams/Web) |
                                                                            +------------------------------+
                                                                                            |
                                                                                            v
                                                                            +------------------------------+
                                                                            |     Embedded HITL Cockpit    |
                                                                            |  - Visual State Inspection   |
                                                                            |  - Interactive Variable Edit |
                                                                            |  - One-Click Resume / Abort  |
                                                                            |  - State Rewind Time-Travel  |
                                                                            +------------------------------+
```

---

## 3. Declarative Workflow Specification (YAML)

Workflows are defined in clear, declarative YAML specifications:

```yaml
version: "1"
workflow:
  id: "vendor-invoice-reconciliation"
  name: "Automated Vendor Invoice Auditing"
  timeout: "5m"
  max_healing_attempts: 3

models:
  primary:
    provider: "openai"
    model: "gpt-4o"
    temperature: 0.1
  healing:
    provider: "openai"
    model: "gpt-4o-mini"
    temperature: 0.0

state:
  invoice_pdf_path: ""
  extracted_data: {}
  erp_purchase_order: {}
  discrepancies: []
  approval_status: "PENDING"

nodes:
  - id: "extract_invoice"
    type: "llm_agent"
    description: "Extract line items, tax IDs, and totals from invoice payload"
    prompt: |
      Extract structured invoice information from: {{ .state.invoice_pdf_path }}
    output_schema:
      type: "object"
      required: ["vendor_name", "tax_id", "total_amount", "currency", "line_items"]
      properties:
        vendor_name: { type: "string" }
        tax_id: { type: "string" }
        total_amount: { type: "number" }
        currency: { type: "string" }
        line_items:
          type: "array"
          items:
            type: "object"
            required: ["description", "quantity", "unit_price", "total"]
    save_to: "extracted_data"
    on_failure:
      strategy: "self_heal"
      prompt_repair: "Verify table borders, check OCR artifacts, and align numbers strictly."

  - id: "fetch_erp_po"
    type: "tool_execution"
    tool: "sqlite_query"
    params:
      query: "SELECT id, vendor_tax_id, total_amount, currency, status FROM purchase_orders WHERE vendor_tax_id = ? AND status = 'APPROVED'"
      args: ["{{ .state.extracted_data.tax_id }}"]
    save_to: "erp_purchase_order"
    retry:
      max_attempts: 3
      backoff: "exponential"

  - id: "reconcile_and_audit"
    type: "llm_agent"
    description: "Compare extracted invoice against approved purchase order"
    prompt: |
      Compare the invoice total {{ .state.extracted_data.total_amount }} with ERP PO total {{ .state.erp_purchase_order.total_amount }}.
      Flag any price variance greater than 2% or unknown line items.
    output_schema:
      type: "object"
      required: ["discrepancies", "is_approved", "variance_percentage", "summary"]
      properties:
        discrepancies: { type: "array", items: { type: "string" } }
        is_approved: { type: "boolean" }
        variance_percentage: { type: "number" }
        summary: { type: "string" }
    save_to: "reconciliation_result"
    confidence_threshold: 0.85
    on_low_confidence:
      action: "pause_to_dlq"
      reason: "Confidence below 85% on financial disbursement"

  - id: "post_approval_or_reject"
    type: "conditional_router"
    conditions:
      - condition: "len(.state.reconciliation_result.discrepancies) == 0 && .state.extracted_data.total_amount < 5000"
        next: "auto_approve_payment"
      - condition: "default"
        next: "pause_for_human_review"

  - id: "auto_approve_payment"
    type: "tool_execution"
    tool: "http_request"
    params:
      method: "POST"
      url: "https://erp.internal.net/api/v1/payments/approve"
      headers:
        Content-Type: "application/json"
      body:
        tax_id: "{{ .state.extracted_data.tax_id }}"
        amount: "{{ .state.extracted_data.total_amount }}"
        status: "APPROVED"
    save_to: "payment_confirmation"

  - id: "pause_for_human_review"
    type: "human_checkpoint"
    description: "Manual approval gate for invoices with discrepancies or totals exceeding threshold"
    save_to: "human_review"

notifications:
  webhook:
    url: "${ALERT_WEBHOOK_URL}"
  slack:
    webhook_url: "${SLACK_ALERTS_WEBHOOK}"
    channel: "#finance-ai-ops"
```

---

## 4. Repository & Directory Structure

```
agentloop/
├── .github/
│   └── workflows/
│       ├── test.yml
│       └── release.yml
├── cmd/
│   └── agentloop/
│       └── main.go                 # CLI entrypoint (run, serve, dlq, resume, validate)
├── internal/
│   ├── config/                     # Engine and server configuration
│   │   └── config.go
│   ├── engine/                     # Workflow and state machine engine
│   │   ├── dag.go                  # Dependency graph builder and cycle detector
│   │   ├── executor.go             # Step runner and context coordinator
│   │   ├── state.go                # JSON state storage and snapshot manager
│   │   ├── expr.go                 # Safe expression evaluator for router & templates
│   │   ├── sandbox.go              # Tool execution environment
│   │   └── types.go                # Engine types and interfaces
│   ├── healing/                    # Self-healing and reflection subsystem
│   │   ├── classifier.go           # Error categorization (schema, network, logic)
│   │   ├── repair_prompt.go        # Dynamic prompt mutation generator
│   │   └── fallback_router.go      # Alternative tool selector
│   ├── dlq/                        # Dead-letter queue
│   │   ├── queue.go                # Suspended execution storage (SQLite WAL)
│   │   ├── snapshot.go             # Complete state serializer
│   │   └── dispatcher.go           # Escalation alerts (Webhook, Slack, Teams)
│   ├── llm/                        # Multi-provider client
│   │   ├── client.go               # Unified interface
│   │   ├── openai.go               # OpenAI and OpenAI-compatible provider
│   │   ├── anthropic.go            # Anthropic Claude provider
│   │   ├── ollama.go               # Local Ollama provider
│   │   ├── mock.go                 # Deterministic mock provider for offline tests
│   │   ├── structured.go           # JSON schema validation and repair
│   │   └── confidence.go           # Logprob and semantic confidence scorer
│   ├── tools/                      # Built-in deterministic tools
│   │   ├── registry.go             # Tool registry and dispatcher
│   │   ├── http.go                 # HTTP REST client
│   │   ├── sql.go                  # SQLite query executor
│   │   ├── transform.go            # JSON and expression data transformation
│   │   └── file.go                 # Sandboxed file operations
│   ├── store/                      # SQLite persistence layer (WAL mode)
│   │   ├── db.go                   # SQLite connection and migration setup
│   │   ├── workflows.go            # Workflow specs repository
│   │   ├── executions.go           # Execution state tracking
│   │   ├── dlq.go                  # DLQ item storage
│   │   └── audit_log.go            # Immutable compliance audit log
│   └── api/                        # HTTP REST and SSE endpoints
│       ├── server.go               # Web server and single-binary embed
│       ├── handlers.go             # REST API handlers
│       ├── sse.go                  # Real-time Server-Sent Events broker
│       └── router.go               # Chi router setup
├── web/                            # Embedded React 19 HITL Cockpit
│   ├── index.html
│   ├── package.json
│   ├── tsconfig.json
│   ├── vite.config.ts
│   ├── tailwind.config.js
│   ├── src/
│   │   ├── main.tsx
│   │   ├── App.tsx
│   │   ├── api/client.ts           # REST and SSE API client
│   │   ├── types/index.ts          # TypeScript interfaces
│   │   ├── components/
│   │   │   ├── Header.tsx          # Navigation and status header
│   │   │   ├── WorkflowDAGView.tsx # Interactive DAG visualization
│   │   │   ├── DLQInbox.tsx        # Suspended items awaiting human review
│   │   │   ├── StateEditor.tsx     # JSON state inspector and editor
│   │   │   ├── ExecutionReplay.tsx # Step-by-step reasoning and audit log
│   │   │   ├── HealingLogView.tsx  # Self-healing attempt diff visualizer
│   │   │   └── TriggerModal.tsx    # Workflow trigger modal
│   │   └── pages/
│   │       ├── Dashboard.tsx       # System overview, metrics, and quick actions
│   │       ├── ExecutionsList.tsx  # All workflow executions
│   │       ├── WorkflowDetail.tsx  # Live DAG and execution detail
│   │       └── DLQDetail.tsx       # HITL review and resume pane
│   └── dist/                       # Production web build (embedded in Go)
├── examples/
│   ├── invoice_reconciliation.yaml # Vendor invoice auditing with self-healing
│   ├── incident_remediation.yaml   # Automated cloud incident triage
│   └── customer_kyc.yaml           # Customer KYC compliance check
├── docs/
│   ├── ARCHITECTURE.md             # System design and execution lifecycle
│   ├── SPECIFICATION.md            # Workflow YAML reference
│   ├── SELF_HEALING.md             # Reflection and repair algorithms
│   ├── HITL_DLQ.md                 # Operator intervention playbooks
│   └── CLI_GUIDE.md                # Command-line usage examples
├── tests/
│   ├── engine_test.go              # State machine and DAG unit tests
│   ├── healing_test.go             # Error classifier and repair tests
│   ├── dlq_test.go                 # Dead-letter queue and resume tests
│   └── e2e_test.go                 # End-to-end integration workflows
├── go.mod
├── go.sum
├── Makefile
├── Dockerfile
├── LICENSE
└── README.md
```

---

## 5. Storage Schema (SQLite WAL Mode)

```sql
PRAGMA journal_mode=WAL;
PRAGMA synchronous=NORMAL;
PRAGMA foreign_keys=ON;

-- Registered workflow definitions
CREATE TABLE IF NOT EXISTS workflows (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    spec_yaml TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Active, paused, and completed workflow executions
CREATE TABLE IF NOT EXISTS executions (
    id TEXT PRIMARY KEY,
    workflow_id TEXT NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    status TEXT NOT NULL, -- 'PENDING', 'RUNNING', 'COMPLETED', 'FAILED', 'PAUSED_DLQ'
    current_node_id TEXT,
    state_json TEXT NOT NULL,
    healing_attempts INTEGER DEFAULT 0,
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);

-- State snapshots per node execution for time-travel rollback
CREATE TABLE IF NOT EXISTS state_snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    execution_id TEXT NOT NULL REFERENCES executions(id) ON DELETE CASCADE,
    node_id TEXT NOT NULL,
    state_json TEXT NOT NULL,
    step_sequence INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Dead-Letter Queue items awaiting human review
CREATE TABLE IF NOT EXISTS dlq_items (
    id TEXT PRIMARY KEY,
    execution_id TEXT NOT NULL REFERENCES executions(id) ON DELETE CASCADE,
    failed_node_id TEXT NOT NULL,
    failure_reason TEXT NOT NULL,
    error_details TEXT,
    snapshot_state_json TEXT NOT NULL,
    resolved_by TEXT, -- Operator username or 'AUTO_RESUMED'
    resolution_action TEXT, -- 'RESUME', 'RETRY_NODE', 'ABORT'
    operator_instruction TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP
);

-- Step-by-step audit trail for compliance
CREATE TABLE IF NOT EXISTS step_audit_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    execution_id TEXT NOT NULL REFERENCES executions(id) ON DELETE CASCADE,
    node_id TEXT NOT NULL,
    node_type TEXT NOT NULL,
    input_payload TEXT,
    output_payload TEXT,
    tokens_consumed INTEGER DEFAULT 0,
    latency_ms INTEGER NOT NULL,
    status TEXT NOT NULL, -- 'SUCCESS', 'HEALED', 'FAILED', 'PAUSED'
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Self-healing attempt logs
CREATE TABLE IF NOT EXISTS healing_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    execution_id TEXT NOT NULL REFERENCES executions(id) ON DELETE CASCADE,
    node_id TEXT NOT NULL,
    attempt_number INTEGER NOT NULL,
    error_category TEXT NOT NULL,
    original_error TEXT NOT NULL,
    mutated_prompt TEXT NOT NULL,
    status TEXT NOT NULL, -- 'SUCCESS', 'FAILED', 'EXHAUSTED'
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

---

## 6. Execution Lifecycle and Self-Healing Algorithm

1. Topological DAG Traversal: The engine builds a directed acyclic graph from the YAML specification, checks for cycles, and determines initial entry nodes.
2. Step Context and Template Interpolation: Node input parameters and prompts are interpolated using the current execution state via Go template and expr engines.
3. Sentinel Evaluation:
   - For LLM nodes: Structured output is parsed against the JSON schema.
   - Confidence score is computed based on token logprobs or semantic evaluation.
   - If confidence < threshold or validation fails, self-healing activates.
4. Self-Healing Decision Loop:
   - Attempt <= N:
     a. Classifier categorizes error (e.g. SCHEMA_MISMATCH, TOOL_TIMEOUT, RATE_LIMIT, EMPTY_RESULT).
     b. Dynamic prompt mutator synthesizes targeted repair instructions highlighting missing or invalid fields.
     c. Retries node execution with exponential backoff and mutated context.
     d. Records attempt details in healing_logs.
   - Attempt > N:
     a. Execution state is frozen atomically.
     b. Execution status transitions to PAUSED_DLQ.
     c. Record inserted into dlq_items.
     d. Escalation alert dispatched to Webhook or Slack.
     e. Execution thread suspends waiting for human review.
5. Human-in-the-Loop Resumption:
   - Operator inspects frozen execution in web console or CLI.
   - Operator can modify state variables or provide a natural language steering prompt.
   - Action triggers Resume or RetryNode.
   - State machine unfreezes, applies state patches, and continues DAG traversal.

---

## 7. Implementation Roadmap & Task Checklist

### Phase 1: Core Engine & State Machine
- [x] Repository initialization, Go module setup, and dependency selection.
- [ ] Implement declarative workflow specification types and YAML parser (`internal/engine/types.go`, `dag.go`).
- [ ] Implement DAG cycle detector and topological validator.
- [ ] Implement atomic JSON state manager with snapshot support (`internal/engine/state.go`).
- [ ] Implement safe expression evaluator and template interpolator (`internal/engine/expr.go`).
- [ ] Implement SQLite persistence layer with WAL mode and schema migrations (`internal/store/`).
- [ ] Implement execution coordinator and node runner (`internal/engine/executor.go`).

### Phase 2: Multi-Provider LLM & Tool Ecosystem
- [ ] Implement unified LLM client interface (`internal/llm/client.go`).
- [ ] Implement OpenAI and Anthropic provider adapters with structured outputs (`internal/llm/openai.go`, `anthropic.go`).
- [ ] Implement Ollama and local model provider adapter (`internal/llm/ollama.go`).
- [ ] Implement deterministic mock LLM provider for zero-cost automated tests (`internal/llm/mock.go`).
- [ ] Implement JSON Schema validator and sanitization engine (`internal/llm/structured.go`).
- [ ] Implement logprob and semantic confidence scoring sentinel (`internal/llm/confidence.go`).
- [ ] Implement built-in tools: HTTP REST, SQLite queries, JSON transforms, file I/O (`internal/tools/`).

### Phase 3: Autonomous Self-Healing Subsystem & DLQ
- [ ] Implement error categorizer (`internal/healing/classifier.go`).
- [ ] Implement surgical prompt mutation engine (`internal/healing/repair_prompt.go`).
- [ ] Implement tool fallback router (`internal/healing/fallback_router.go`).
- [ ] Implement Dead-Letter Queue engine and state freezing (`internal/dlq/queue.go`, `snapshot.go`).
- [ ] Implement alert dispatcher for Webhooks, Slack, and console alerts (`internal/dlq/dispatcher.go`).
- [ ] Implement HITL resume, retry, state patch, and abort handlers (`internal/dlq/`).

### Phase 4: REST API, Server-Sent Events (SSE) & CLI
- [ ] Implement Chi HTTP REST router and handlers for workflows, executions, DLQ, and audit logs (`internal/api/`).
- [ ] Implement real-time SSE broker for streaming execution events to the web console (`internal/api/sse.go`).
- [ ] Implement complete CLI with commands: `run`, `serve`, `dlq list`, `dlq inspect`, `resume`, `validate`, `version` (`cmd/agentloop/main.go`).

### Phase 5: Embedded React 19 HITL Cockpit
- [ ] Initialize React 19 + TypeScript + Vite + Tailwind CSS project in `web/`.
- [ ] Implement API client with REST and SSE event stream subscription (`web/src/api/client.ts`).
- [ ] Build interactive Workflow DAG Visualizer with node statuses and execution path highlights.
- [ ] Build Dead-Letter Queue Inbox and incident inspection view.
- [ ] Build interactive JSON State Editor with patch validation.
- [ ] Build Execution Replay and Step Audit Log timeline.
- [ ] Build Self-Healing Attempt Diff Viewer.
- [ ] Build production assets into `web/dist` and embed into Go binary via `//go:embed`.

### Phase 6: Examples, Comprehensive Testing & Documentation
- [ ] Create realistic workflow examples: `invoice_reconciliation.yaml`, `incident_remediation.yaml`, `customer_kyc.yaml`.
- [ ] Write unit and integration tests for engine, self-healing, DLQ, and API.
- [ ] Benchmark execution overhead to verify sub-25ms node transition target.
- [ ] Author documentation in `docs/` and root `README.md` using humanizer principles.
- [ ] Push to GitHub repository.
