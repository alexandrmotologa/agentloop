# agentloop

> Self-Healing Autonomous Enterprise Workflow Engine with Dead-Letter Queues & Human-in-the-Loop

agentloop is a deterministic workflow engine written in Go with an embedded React 19 cockpit. It treats AI agent execution as a recoverable state machine with sub-25ms node transition overhead, automated self-healing loops, state freezing in Dead-Letter Queues (DLQ), and one-click human intervention.

---

## The Problem

More than 85% of autonomous AI agent pilots stall or fail in production environments due to recurring failure modes:

1. Brittle Tool Execution: Third-party APIs change field names, time out, or return 502 Bad Gateway responses. Unconstrained agents enter recursive retry loops or hallucinate parameters.
2. Context and State Loss: If a multi-step business process fails at step 5 of 6, typical agent frameworks discard the entire transaction without an ability to rewind and repair context.
3. Lack of Human Safety Nets: When an agent encounters low confidence or unrecoverable schema errors, it either makes an unverified guess or crashes without an escalation path.
4. Runaway Spend and Unpredictable Costs: Infinite reflection loops or prompt explosions rapidly deplete API quotas without deterministic circuit breakers.

agentloop resolves these issues by combining deterministic directed acyclic graphs (DAGs) with autonomous self-healing sentinels, token and cost budget guards, checkpoint state freezing, and an operator console.

---

## Core Capabilities

- Deterministic Workflow DAG: Combines deterministic business logic (HTTP REST, SQL queries, JSON expression transforms) with probabilistic LLM reasoning nodes.
- Autonomous Self-Healing Loop: When a node fails schema validation or confidence checks, an internal reflection engine analyzes the error, categorizes the fault, and synthesizes a surgical repair prompt without losing prior state.
- Cost & Token Budget Sentinel: Enforces hard limits on token usage and estimated dollar spend per execution. If limits are reached, the execution freezes to DLQ to prevent cost overruns.
- Multi-Model Fallback Cascades: Automatically fails over across providers (e.g. OpenAI to Anthropic or local Ollama) if a primary model is throttled or times out.
- Webhook Ingestion Gateway: Ingests external webhook triggers with HMAC-SHA256 signature verification and idempotency keys to prevent duplicate execution.
- Lightweight Cron Scheduler: Runs scheduled workflows deterministically using standard cron expressions and interval descriptors.
- Breakpoint & Step-Through Debugger: Allows developers to set execution breakpoints on DAG nodes, step forward node-by-node, inspect memory state, and resume execution.
- Side-by-Side Visual State Diff: Visualizes state deltas between baseline state and frozen DLQ memory in the React 19 console with highlight styling for added, modified, and removed variables.
- Compliance Audit Report Exporter: Compiles end-to-end execution lineage, token budgets, self-healing events, and operator actions into GitHub-flavored Markdown or structured JSON.
- Dead-Letter Queue (DLQ) & Checkpoint Freezing: When healing attempts exceed configured bounds or low confidence thresholds are reached, execution freezes at the exact node, persisting full serialized memory state to SQLite WAL storage.
- Embedded React 19 HITL Cockpit: Operators inspect live DAG execution progress, review error traces, edit memory variables, provide natural language guidance, and resume executions with one click.
- State Rewind & Time-Travel: Allows operators to roll back execution state to any prior node snapshot and resume execution from that point.
- Single-Binary Deployment: Compiles into a single standalone binary with embedded React 19 frontend assets via Go embed.
- Sub-25ms Execution Overhead: Core state machine transitions execute with sub-millisecond to low single-digit millisecond latency.

---

## Architecture

```
                                  +---------------------------------------+
                                  |         Enterprise Trigger            |
                                  | (Webhook HMAC, Cron, Manual CLI / UI) |
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
         |  - SQLite / SQL Queries       |                         |  - Multimodal Extraction      |
         |  - Data Transform (Expr/JSON) |                         |  - Multi-Model Cascades       |
         |  - Sandboxed File I/O         |                         |  - Schema-Enforced Generation |
         +-------------------------------+                         +-------------------------------+
                         |                                                         |
                         +----------------------------+----------------------------+
                                                      |
                                                      v
                                       +------------------------------+
                                       |      Execution Sentinel      |
                                       | - JSON Schema Validation     |
                                       | - Confidence Scoring (<0.85) |
                                       | - Token & Cost Budget Guard  |
                                       | - Breakpoint Interceptor     |
                                       +------------------------------+
                                                      |
                                   +------------------+------------------+
                                   | (Success)                           | (Failure / Low Confidence / Budget)
                                   v                                     v
                       +----------------------+          +------------------------------+
                       | Transition Next Node |          |     Self-Healing Engine      |
                       +----------------------+          | - Error Classification       |
                                                         | - Surgical Prompt Mutation   |
                                                         | - Model Fallback Cascade     |
                                                         | - Exponential Backoff        |
                                                         +------------------------------+
                                                                         |
                                                      +------------------+------------------+
                                                      | (Recovered <= N)                    | (Exhausted > N)
                                                      v                                     v
                                          +----------------------+          +------------------------------+
                                          | Resume Execution DAG |          |    Dead-Letter Queue (DLQ)   |
                                          +----------------------+          | - State Freeze & Snapshot    |
                                                                            | - Alerting (Slack/Webhooks)  |
                                                                            +------------------------------+
                                                                                            |
                                                                                            v
                                                                            +------------------------------+
                                                                            |     Embedded HITL Cockpit    |
                                                                            |  - Visual State Diff (Added) |
                                                                            |  - Step Debugger (F10/F5)    |
                                                                            |  - One-Click Resume / Abort  |
                                                                            |  - State Rewind Time-Travel  |
                                                                            |  - Compliance Report Export  |
                                                                            +------------------------------+
```

---

## Quickstart

### Prerequisites
- Go 1.22+ installed
- Node.js 20+ installed (for building frontend assets)

### 1. Build
```bash
git clone https://github.com/alexandrmotologa/agentloop.git
cd agentloop
make build
```

### 2. Validate Workflow Specification
```bash
./agentloop validate --file ./examples/invoice_reconciliation.yaml
```

Output:
```
✓ Workflow 'Automated Vendor Invoice Auditing' (vendor-invoice-reconciliation) is valid.
✓ DAG contains 6 nodes, zero cycles detected.
✓ Topological execution order:
  1. extract_invoice          [llm_agent]
  2. fetch_erp_po             [tool_execution]
  3. reconcile_and_audit      [llm_agent]
  4. post_approval_or_reject  [conditional_router]
  5. auto_approve_payment     [tool_execution]
  6. pause_for_human_review   [human_checkpoint]
```

### 3. Run Workflow from CLI
```bash
./agentloop run --file ./examples/invoice_reconciliation.yaml
```

### 4. Generate Compliance Report
```bash
./agentloop report --id exec_734800df --format markdown --out audit_report.md
```

### 5. Start the Web Console & Scheduler
```bash
./agentloop serve --port 8080 --db ./agentloop.db
```

Open `http://localhost:8080` in your browser to access the HITL cockpit.

---

## Workflow Specification

Workflows are written in declarative YAML specifications:

```yaml
version: "1"
workflow:
  id: "vendor-invoice-reconciliation"
  name: "Automated Vendor Invoice Auditing"
  timeout: "5m"
  max_healing_attempts: 3
  schedule: "@hourly" # Optional recurring schedule
  webhook_secret: "prod-secret-key" # Optional HMAC-SHA256 secret

budget:
  max_total_tokens: 50000
  max_cost_usd: 0.25
  on_exceeded: "pause_to_dlq"

models:
  primary:
    provider: "openai" # openai, anthropic, ollama, or mock
    model: "gpt-4o"
    temperature: 0.1
    fallback_models: ["gpt-4o-mini", "claude-3-5-sonnet"]
  healing:
    provider: "openai"
    model: "gpt-4o-mini"
    temperature: 0.0

state:
  invoice_pdf_path: "./sample_invoice.pdf"
  extracted_data: {}
  erp_purchase_order: {}
  reconciliation_result: {}

nodes:
  - id: "extract_invoice"
    type: "llm_agent"
    prompt: |
      Extract structured invoice information from document path: {{ .state.invoice_pdf_path }}
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
    breakpoint: false # Toggle to true to pause execution before this node
    prompt: |
      Compare extracted invoice total {{ .state.extracted_data.total_amount }} with ERP PO total {{ .state.erp_purchase_order.total_amount }}.
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
      - condition: "len(state.reconciliation_result.discrepancies) == 0 && state.extracted_data.total_amount < 5000"
        next: "auto_approve_payment"
      - condition: "default"
        next: "pause_for_human_review"

  - id: "auto_approve_payment"
    type: "tool_execution"
    tool: "sqlite_query"
    params:
      query: "INSERT INTO purchase_orders (id, vendor_tax_id, vendor_name, total_amount, currency, status) VALUES ('PAY-RECORD-001', '{{ .state.extracted_data.tax_id }}', '{{ .state.extracted_data.vendor_name }}', {{ .state.extracted_data.total_amount }}, '{{ .state.extracted_data.currency }}', 'PAID')"
    save_to: "payment_confirmation"

  - id: "pause_for_human_review"
    type: "human_checkpoint"
    description: "Manual approval gate for invoices with discrepancies or totals exceeding threshold"
    save_to: "human_review"
```

---

## Dead-Letter Queue & Human-in-the-Loop CLI

Inspect suspended items:
```bash
./agentloop dlq list --status PENDING
```

Inspect frozen state:
```bash
./agentloop dlq inspect --id dlq_957300b5
```

Resume execution with natural language operator directive and state patch:
```bash
./agentloop resume --dlq-id dlq_957300b5 \
  --instruction "Vendor tax ID was confirmed valid with corporate tax registry" \
  --patch '{"extracted_data": {"tax_id": "US-94827104"}}'
```

---

## REST API Reference

| Endpoint | Method | Description |
|---|---|---|
| `/api/health` | GET | Healthcheck and engine version |
| `/api/workflows` | GET | List registered workflow specifications |
| `/api/workflows` | POST | Upload and validate new workflow YAML |
| `/api/workflows/{id}` | GET | Get workflow specification and raw YAML |
| `/api/workflows/{id}/run` | POST | Trigger workflow execution with JSON payload |
| `/api/webhooks/{id}` | POST | Ingest webhook event with HMAC-SHA256 and idempotency check |
| `/api/cron/jobs` | GET | List all active scheduled recurring jobs |
| `/api/executions` | GET | List execution instances (filterable by status) |
| `/api/executions/{id}` | GET | Get execution details and current state |
| `/api/executions/{id}/audit` | GET | Get compliance step audit log |
| `/api/executions/{id}/healing` | GET | Get self-healing attempt history |
| `/api/executions/{id}/snapshots` | GET | Get state snapshots for time-travel |
| `/api/executions/{id}/rewind` | POST | Roll back state to target node snapshot |
| `/api/executions/{id}/step` | POST | Single-step forward from a breakpoint halt |
| `/api/executions/{id}/continue` | POST | Resume execution run from a breakpoint |
| `/api/executions/{id}/breakpoints` | POST | Configure active breakpoints on execution |
| `/api/executions/{id}/report` | GET | Download compliance audit report (Markdown or JSON) |
| `/api/dlq` | GET | List Dead-Letter Queue items |
| `/api/dlq/{id}` | GET | Get DLQ item details and snapshot state |
| `/api/dlq/{id}/resume` | POST | Resume execution with operator directive and state patch |
| `/api/dlq/{id}/abort` | POST | Abort execution with audit reason |
| `/api/events` | GET | Real-time Server-Sent Events (SSE) event stream |

---

## Performance & Overhead

| Metric | Target | Benchmark Measured |
|---|---|---|
| Node Transition Overhead | &lt; 25ms | ~0.37ms - 1.1ms |
| DAG Cycle Validation | Instant | &lt; 50µs |
| SQLite State Snapshot Persistence | &lt; 5ms (WAL mode) | ~0.2ms |
| Frontend Asset Bundle Size | Minimal | 89 kB gzipped |

---

## Documentation

- [System Architecture](docs/ARCHITECTURE.md)
- [Workflow Specification Reference](docs/SPECIFICATION.md)
- [Autonomous Self-Healing Subsystem](docs/SELF_HEALING.md)
- [Dead-Letter Queues & Human-in-the-Loop Operations](docs/HITL_DLQ.md)
- [CLI Reference Guide](docs/CLI_GUIDE.md)

---

## License

MIT License. See [LICENSE](LICENSE) for details.
