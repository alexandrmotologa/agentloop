package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/alexandrmotologa/agentloop/internal/engine"
	"github.com/alexandrmotologa/agentloop/internal/store"
	"github.com/go-chi/chi/v5"
)

type API struct {
	db       *store.DB
	executor *engine.Executor
}

func NewAPI(db *store.DB, executor *engine.Executor) *API {
	return &API{
		db:       db,
		executor: executor,
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// Health check endpoint
func (a *API) HandleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "healthy",
		"version":   "1.0.0",
		"timestamp": time.Now(),
	})
}

// ListWorkflows lists all stored workflow specifications.
func (a *API) HandleListWorkflows(w http.ResponseWriter, r *http.Request) {
	list, err := a.db.ListWorkflows()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// GetWorkflow fetches a workflow specification by ID.
func (a *API) HandleGetWorkflow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	spec, rawYAML, err := a.db.GetWorkflow(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"spec":     spec,
		"raw_yaml": rawYAML,
	})
}

// CreateWorkflow uploads and validates a workflow specification.
func (a *API) HandleCreateWorkflow(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}

	dag, err := engine.ParseWorkflowYAML(body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid workflow YAML: "+err.Error())
		return
	}

	if err := a.db.SaveWorkflow(dag.Spec, string(body)); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"message":  "workflow created",
		"id":       dag.Spec.Workflow.ID,
		"workflow": dag.Spec.Workflow,
	})
}

// DeleteWorkflow removes a workflow definition.
func (a *API) HandleDeleteWorkflow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := a.db.DeleteWorkflow(id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "workflow deleted"})
}

// RunWorkflow initiates an execution.
func (a *API) HandleRunWorkflow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req struct {
		Input map[string]any `json:"input"`
		Async bool           `json:"async"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	if req.Async {
		go func() {
			_, _ = a.executor.Start(r.Context(), id, req.Input)
		}()
		writeJSON(w, http.StatusAccepted, map[string]string{"message": "workflow triggered asynchronously"})
		return
	}

	exec, err := a.executor.Start(r.Context(), id, req.Input)
	if err != nil && exec == nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, exec)
}

// ListExecutions returns execution instances.
func (a *API) HandleListExecutions(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	executions, err := a.db.ListExecutions(limit, status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, executions)
}

// GetExecution fetches details for an execution.
func (a *API) HandleGetExecution(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	exec, err := a.db.GetExecution(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, exec)
}

// GetAuditLogs returns compliance step audit logs for an execution.
func (a *API) HandleGetAuditLogs(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	logs, err := a.db.GetAuditLogs(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, logs)
}

// GetHealingLogs returns reflection and repair attempts.
func (a *API) HandleGetHealingLogs(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	logs, err := a.db.GetHealingLogs(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, logs)
}

// GetSnapshots returns state snapshots for time-travel inspection.
func (a *API) HandleGetSnapshots(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	snaps, err := a.db.GetSnapshots(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, snaps)
}

// RewindExecution rolls back execution state to a prior node snapshot.
func (a *API) HandleRewindExecution(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		TargetNodeID string `json:"target_node_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TargetNodeID == "" {
		writeError(w, http.StatusBadRequest, "target_node_id is required")
		return
	}

	exec, err := a.executor.Rewind(r.Context(), id, req.TargetNodeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, exec)
}

// ListDLQItems returns dead-letter queue items.
func (a *API) HandleListDLQItems(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	items, err := a.db.ListDLQItems(status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// GetDLQItem fetches a specific DLQ item.
func (a *API) HandleGetDLQItem(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	item, err := a.db.GetDLQItem(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// ResumeDLQItem allows operator to intervene and resume execution.
func (a *API) HandleResumeDLQItem(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		OperatorInstruction string         `json:"operator_instruction"`
		Patch               map[string]any `json:"patch"`
		RetryCurrentNode    bool           `json:"retry_current_node"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	exec, err := a.executor.Resume(r.Context(), id, req.OperatorInstruction, req.Patch, req.RetryCurrentNode)
	if err != nil && exec == nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, exec)
}

// AbortDLQItem marks execution aborted.
func (a *API) HandleAbortDLQItem(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	item, err := a.db.GetDLQItem(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	_ = a.db.ResolveDLQItem(id, "operator", "ABORT", req.Reason)

	exec, err := a.db.GetExecution(item.ExecutionID)
	if err == nil {
		now := time.Now()
		exec.Status = engine.StatusFailed
		exec.CompletedAt = &now
		_ = a.db.UpdateExecution(exec)
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "execution aborted"})
}
