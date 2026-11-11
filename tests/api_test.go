package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexandrmotologa/agentloop/internal/api"
	"github.com/alexandrmotologa/agentloop/internal/dlq"
	"github.com/alexandrmotologa/agentloop/internal/engine"
	"github.com/alexandrmotologa/agentloop/internal/store"
	"github.com/alexandrmotologa/agentloop/internal/types"
)

func TestAPIEndpoints(t *testing.T) {
	db, err := store.OpenDB(":memory:")
	if err != nil {
		t.Fatalf("OpenDB failed: %v", err)
	}
	defer db.Close()

	dlqMgr := dlq.NewManager(db)
	executor := engine.NewExecutor(db, dlqMgr)
	router := api.NewRouter(db, executor, nil)

	// 1. Health check
	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// 2. Create Workflow
	wfYAML := `
version: "1"
workflow:
  id: "api-test-wf"
  name: "API Test Workflow"
nodes:
  - id: "step_one"
    type: "transform"
    params:
      expr: "{ status: 'OK' }"
    save_to: "result"
`
	req = httptest.NewRequest("POST", "/api/workflows", bytes.NewReader([]byte(wfYAML)))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201 for workflow creation, got %d", w.Code)
	}

	// 3. List Workflows
	req = httptest.NewRequest("GET", "/api/workflows", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// 4. Run Workflow
	runBody := `{"input": {}}`
	req = httptest.NewRequest("POST", "/api/workflows/api-test-wf/run", bytes.NewReader([]byte(runBody)))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for workflow run, got %d", w.Code)
	}

	var exec types.Execution
	if err := json.Unmarshal(w.Body.Bytes(), &exec); err != nil {
		t.Fatalf("Failed to decode execution response: %v", err)
	}
	if exec.Status != types.StatusCompleted {
		t.Errorf("Expected execution completed, got %s", exec.Status)
	}

	// 5. Get Executions list
	req = httptest.NewRequest("GET", "/api/executions", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for executions list, got %d", w.Code)
	}

	// 6. Get Audit logs
	req = httptest.NewRequest("GET", "/api/executions/"+exec.ID+"/audit", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for audit logs, got %d", w.Code)
	}
}
