# Command-Line Interface (CLI) Guide

`agentloop` provides a unified command-line tool for running workflows, validating specifications, managing the Dead-Letter Queue, and running the web console.

## Installation

### From Source
```bash
git clone https://github.com/alexandrmotologa/agentloop.git
cd agentloop
make build
```

Binary will be placed in the project root as `agentloop` (or `agentloop.exe` on Windows).

## Command Reference

### 1. `agentloop serve`
Starts the background daemon, HTTP REST API, SSE event stream, and embedded React 19 cockpit.

```bash
# Start server on default port 8080 with SQLite database
agentloop serve --port 8080 --db ./agentloop.db

# Access web cockpit in browser:
# http://localhost:8080
```

Flags:
- `--port`: HTTP port (default: 8080)
- `--db`: SQLite database file path (default: `./agentloop.db`)

### 2. `agentloop run`
Executes a workflow specification synchronously from the command line.

```bash
# Run invoice reconciliation with default parameters
agentloop run --file ./examples/invoice_reconciliation.yaml

# Run with custom JSON input payload
agentloop run --file ./examples/invoice_reconciliation.yaml \
  --input '{"invoice_pdf_path": "./data/inv_994.pdf"}' \
  --db ./agentloop.db
```

Flags:
- `--file`: Path to workflow YAML file (required)
- `--input`: JSON payload merged into initial workflow state (default: `{}`)
- `--db`: Database path (default: `:memory:`)

### 3. `agentloop dlq list`
Lists all items suspended in the Dead-Letter Queue.

```bash
# List all items
agentloop dlq list

# Filter only pending items awaiting review
agentloop dlq list --status PENDING

# Filter resolved items
agentloop dlq list --status RESOLVED
```

### 4. `agentloop dlq inspect`
Shows details, failure reasons, and the frozen memory state snapshot for a DLQ item.

```bash
agentloop dlq inspect --id dlq_957300b5
```

### 5. `agentloop resume`
Resumes a suspended execution with optional steering instructions or state patches.

```bash
# Resume and advance to next node with operator directive
agentloop resume --dlq-id dlq_957300b5 \
  --instruction "Verified vendor tax ID with state tax database"

# Resume with state variable patch
agentloop resume --dlq-id dlq_957300b5 \
  --patch '{"extracted_data": {"tax_id": "US-94827104"}}'

# Retry the currently failed node instead of advancing
agentloop resume --dlq-id dlq_957300b5 \
  --instruction "Retry extraction focusing on invoice header" \
  --retry
```

### 6. `agentloop validate`
Performs static analysis on a workflow specification without executing it. Checks YAML syntax, schema definitions, and detects circular dependency cycles using topological sorting.

```bash
agentloop validate --file ./examples/customer_kyc.yaml
```

### 7. `agentloop version`
Prints the engine version and runtime environment.

```bash
agentloop version
```
