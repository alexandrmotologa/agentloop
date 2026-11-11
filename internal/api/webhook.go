package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// HandleWebhook ingests external webhook triggers, verifies HMAC signatures, enforces idempotency,
// and initiates a workflow execution.
func (a *API) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	workflowID := chi.URLParam(r, "id")
	if workflowID == "" {
		writeError(w, http.StatusBadRequest, "workflow id is required")
		return
	}

	spec, _, err := a.db.GetWorkflow(workflowID)
	if err != nil {
		writeError(w, http.StatusNotFound, "workflow not found")
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	// 1. Signature Verification (HMAC SHA-256) if webhook secret is configured
	if spec.Workflow.WebhookSecret != "" {
		sigHeader := r.Header.Get("X-Hub-Signature-256")
		if sigHeader == "" {
			sigHeader = r.Header.Get("X-Signature-256")
		}
		if sigHeader == "" {
			sigHeader = r.Header.Get("X-Signature")
		}

		if sigHeader == "" {
			writeError(w, http.StatusUnauthorized, "missing signature header for protected webhook")
			return
		}

		cleanSig := strings.TrimPrefix(sigHeader, "sha256=")
		expectedSigBytes, err := hex.DecodeString(cleanSig)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid signature format")
			return
		}

		mac := hmac.New(sha256.New, []byte(spec.Workflow.WebhookSecret))
		mac.Write(bodyBytes)
		calculatedSigBytes := mac.Sum(nil)

		if !hmac.Equal(calculatedSigBytes, expectedSigBytes) {
			writeError(w, http.StatusUnauthorized, "invalid webhook signature")
			return
		}
	}

	// 2. Idempotency Check
	idempKey := r.Header.Get("Idempotency-Key")
	if idempKey == "" {
		idempKey = r.Header.Get("X-Idempotency-Key")
	}

	execID := "exec_" + uuid.New().String()[:8]

	if idempKey != "" {
		alreadyProcessed, existingExecID, err := a.db.CheckAndSetIdempotency(idempKey, workflowID, execID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to check idempotency key: "+err.Error())
			return
		}
		if alreadyProcessed {
			writeJSON(w, http.StatusOK, map[string]any{
				"status":         "idempotent_duplicate",
				"workflow_id":    workflowID,
				"execution_id":   existingExecID,
				"idempotency_key": idempKey,
				"message":        "webhook already ingested and processed for this idempotency key",
			})
			return
		}
	}

	// 3. Parse input payload
	var parsedBody any
	if len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, &parsedBody); err != nil {
			parsedBody = string(bodyBytes)
		}
	} else {
		parsedBody = map[string]any{}
	}

	// Build headers map for trigger context
	headerMap := make(map[string]string)
	for k, v := range r.Header {
		if len(v) > 0 {
			headerMap[strings.ToLower(k)] = v[0]
		}
	}

	initialState := map[string]any{
		"webhook_payload": parsedBody,
		"webhook_headers": headerMap,
		"triggered_by":    "webhook",
	}

	// If body is JSON object, merge top-level properties for convenient workflow access
	if bodyObj, ok := parsedBody.(map[string]any); ok {
		for k, v := range bodyObj {
			if _, exists := initialState[k]; !exists {
				initialState[k] = v
			}
		}
	}

	// 4. Trigger Execution
	isSync := r.URL.Query().Get("sync") == "true"
	if isSync {
		exec, err := a.executor.StartWithID(r.Context(), execID, workflowID, initialState)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to run workflow: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, exec)
		return
	}

	// Async mode: dispatch execution in background and return 202 Accepted
	go func() {
		_, _ = a.executor.StartWithID(context.Background(), execID, workflowID, initialState)
	}()

	writeJSON(w, http.StatusAccepted, map[string]any{
		"status":       "accepted",
		"workflow_id":  workflowID,
		"execution_id": execID,
		"message":      "webhook ingested successfully; workflow execution started",
	})
}
