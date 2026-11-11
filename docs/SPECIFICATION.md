# Workflow Specification Reference

Workflows in agentloop are configured in YAML or JSON files conforming to the specification below.

## Top-Level Schema

```yaml
version: "1"

workflow:
  id: "workflow-unique-identifier"
  name: "Human Readable Workflow Title"
  description: "Optional description of workflow purpose"
  timeout: "10m"
  max_healing_attempts: 3

models:
  primary:
    provider: "openai" # openai | anthropic | ollama | mock
    model: "gpt-4o"
    temperature: 0.1
    max_tokens: 4096
    api_key_env: "OPENAI_API_KEY"
  healing:
    provider: "openai"
    model: "gpt-4o-mini"
    temperature: 0.0

state:
  initial_var: "default_value"
  counter: 0
  items: []

nodes:
  - id: "step_one"
    type: "llm_agent"
    # Node specific fields...

notifications:
  webhook:
    url: "https://ops.example.com/api/alerts"
  slack:
    webhook_url: "https://hooks.slack.com/services/..."
    channel: "#ai-incident-triage"
```

## Node Types

### 1. `llm_agent`
Performs structured generation, reasoning, or extraction using an LLM.

```yaml
- id: "extract_invoice"
  type: "llm_agent"
  description: "Extract line items and totals"
  prompt: |
    Extract structured invoice data from: {{ .state.invoice_path }}
  output_schema:
    type: "object"
    required: ["vendor_name", "tax_id", "total_amount"]
    properties:
      vendor_name: { type: "string" }
      tax_id: { type: "string" }
      total_amount: { type: "number" }
  save_to: "extracted_data"
  confidence_threshold: 0.85
  on_low_confidence:
    action: "pause_to_dlq"
    reason: "Confidence score below 85%"
  on_failure:
    strategy: "self_heal"
    prompt_repair: "Verify table borders and align numbers strictly."
```

### 2. `tool_execution`
Executes a registered tool.

```yaml
- id: "query_database"
  type: "tool_execution"
  tool: "sqlite_query"
  params:
    query: "SELECT * FROM orders WHERE id = ?"
    args: ["{{ .state.order_id }}"]
  save_to: "db_result"
  retry:
    max_attempts: 3
    backoff: "exponential"
  on_failure:
    fallback_tool: "read_backup_cache"
```

Built-in tools:
- `http_request`: `method`, `url`, `headers`, `body`, `timeout`
- `sqlite_query`: `query`, `args`, `db_path`
- `json_transform`: `expr`, `input`
- `file_read`: `path`
- `file_write`: `path`, `content`

### 3. `conditional_router`
Evaluates expressions in order and directs flow to the first matching branch.

```yaml
- id: "route_by_risk"
  type: "conditional_router"
  conditions:
    - condition: "state.risk_score > 0.8 || state.amount > 10000"
      next: "manual_approval"
    - condition: "default"
      next: "auto_approve"
```

### 4. `human_checkpoint`
An explicit pause point that halts execution and creates a DLQ item.

```yaml
- id: "manual_approval"
  type: "human_checkpoint"
  description: "Requires senior financial auditor approval"
  save_to: "auditor_review"
```

### 5. `transform`
Executes an expression to reshape or aggregate data.

```yaml
- id: "calculate_summary"
  type: "transform"
  params:
    expr: "{ total: state.subtotal + state.tax, approved: true }"
  save_to: "summary"
```
