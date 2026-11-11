# Autonomous Self-Healing Subsystem

## Problem Statement
Standard AI agents in production often fail on schema validation, API changes, or unexpected JSON syntax. Naive retry systems simply repeat the initial prompt, which tends to generate the same mistake or consume tokens without progress.

agentloop solves this by implementing structured reflection and surgical prompt mutation.

## The Self-Healing Algorithm

```
Step Failed (Schema Error / Syntax / Confidence)
                     |
                     v
       Error Classifier Analysis
                     |
       +-------------+-------------+
       |                           |
       v                           v
Schema Mismatch / Low Conf    Network 502 / Rate Limit
       |                           |
       v                           v
Dynamic Prompt Mutator        Exponential Backoff Jitter
(Injects exact error,         (Retries request with backoff)
 preserves prior context)
       |
       v
Execute Mutated Call (Attempt <= N)
       |
       +------------------+------------------+
       | Succeeded                           | Exhausted (> N)
       v                                     v
Resume DAG Traversal               Freeze to Dead-Letter Queue
                                   (Save Snapshot, Alert Operator)
```

## Error Classification Rules

Failures are categorized into 5 specific classes:

1. `SCHEMA_MISMATCH`: The JSON output does not satisfy the specified JSON Schema (e.g., missing required fields, type mismatches like string instead of number).
2. `JSON_SYNTAX_ERROR`: The model returned truncated JSON, unclosed braces, or conversational wrappers like markdown fences without valid content.
3. `LOW_CONFIDENCE`: The computed confidence falls below the node's `confidence_threshold`.
4. `NETWORK_TRANSIENT`: HTTP 429 rate limits, 502 bad gateways, 503 service unavailable, or connection timeouts.
5. `TOOL_ERROR`: Query execution failures or non-zero command exits.

## Surgical Prompt Mutation

When an LLM node fails validation, `SynthesizeRepairPrompt` generates a correction directive that:
- Retains the original prompt instructions and context.
- Appends the classification tag and exact validation failure list.
- Re-injects the required JSON schema.
- Includes the previous raw output so the model can repair specific fields without regenerating correct data from scratch.
- Includes any custom `prompt_repair` hint specified in the node YAML.

Example generated directive:

```
=== SYSTEM SELF-HEALING CORRECTION DIRECTIVE ===
Your previous response failed validation with category [SCHEMA_MISMATCH].
Exact validation failures detected:
  1. missing properties: 'total_amount', 'currency'

Operational Guidance: Verify table borders and check OCR artifacts.

You must strictly conform your response to this JSON schema:
{
  "type": "object",
  "required": ["vendor_name", "tax_id", "total_amount", "currency", "line_items"],
  ...
}

Your previous raw output was:
{"vendor_name": "Acme Corp", "tax_id": "US-123456"}

Fix the errors above, retain all verified factual details, and return only the corrected JSON payload.
```

## Retry Bounds & Backoff

Self-healing attempts are bounded by `max_healing_attempts` (default: 3). Between attempts, the engine applies exponential backoff:

```
delay = 2^(attempt - 1) * 100ms
```

If the node does not pass validation within the allowed attempts, execution freezes immediately and enters the Dead-Letter Queue.
