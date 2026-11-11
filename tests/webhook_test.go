package tests

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexandrmotologa/agentloop/internal/api"
	"github.com/alexandrmotologa/agentloop/internal/dlq"
	"github.com/alexandrmotologa/agentloop/internal/engine"
	"github.com/alexandrmotologa/agentloop/internal/store"
	"github.com/alexandrmotologa/agentloop/internal/types"
)

func TestWebhookIngestionAndSecurity(t *testing.T) {
	db, err := store.OpenDB(":memory:")
	if err != nil {
		t.Fatalf("OpenDB failed: %v", err)
	}
	defer db.Close()

	dlqMgr := dlq.NewManager(db)
	executor := engine.NewExecutor(db, dlqMgr)
	router := api.NewRouter(db, executor, nil, nil)

	webhookSecret := "super-secure-webhook-secret-key-123"

	// Setup protected workflow
	wfSpec := &types.WorkflowSpec{
		Version: "1",
		Workflow: types.WorkflowMetadata{
			ID:            "webhook-order-flow",
			Name:          "Webhook Order Flow",
			WebhookSecret: webhookSecret,
		},
		Nodes: []types.NodeSpec{
			{
				ID:   "process_order",
				Type: types.NodeTypeTransform,
				Params: map[string]any{
					"expr": "{ order_received: true }",
				},
				SaveTo: "order_status",
			},
		},
	}

	if err := db.SaveWorkflow(wfSpec, ""); err != nil {
		t.Fatalf("SaveWorkflow failed: %v", err)
	}

	payload := `{"order_id": "ORD-9901", "amount": 250.75}`

	// 1. Missing HMAC signature must fail with 401 Unauthorized
	req := httptest.NewRequest("POST", "/api/webhooks/webhook-order-flow?sync=true", bytes.NewReader([]byte(payload)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 Unauthorized when signature missing, got %d (body: %s)", w.Code, w.Body.String())
	}

	// 2. Invalid HMAC signature must fail with 401 Unauthorized
	req = httptest.NewRequest("POST", "/api/webhooks/webhook-order-flow?sync=true", bytes.NewReader([]byte(payload)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hub-Signature-256", "sha256=00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 Unauthorized on invalid signature, got %d", w.Code)
	}

	// 3. Valid HMAC signature must succeed
	mac := hmac.New(sha256.New, []byte(webhookSecret))
	mac.Write([]byte(payload))
	validSig := hex.EncodeToString(mac.Sum(nil))

	req = httptest.NewRequest("POST", "/api/webhooks/webhook-order-flow?sync=true", bytes.NewReader([]byte(payload)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hub-Signature-256", "sha256="+validSig)
	req.Header.Set("Idempotency-Key", "idemp-key-xyz-100")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 OK on valid webhook call, got %d (body: %s)", w.Code, w.Body.String())
	}

	var firstExec types.Execution
	if err := json.Unmarshal(w.Body.Bytes(), &firstExec); err != nil {
		t.Fatalf("Failed to parse first execution: %v", err)
	}
	if firstExec.ID == "" {
		t.Fatalf("Execution ID is empty")
	}

	// 4. Duplicate request with same Idempotency-Key must return idempotent duplicate response with same execution ID
	req = httptest.NewRequest("POST", "/api/webhooks/webhook-order-flow?sync=true", bytes.NewReader([]byte(payload)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hub-Signature-256", "sha256="+validSig)
	req.Header.Set("Idempotency-Key", "idemp-key-xyz-100")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 on duplicate request, got %d", w.Code)
	}

	var dupResp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &dupResp); err != nil {
		t.Fatalf("Failed to parse duplicate response: %v", err)
	}

	if dupResp["status"] != "idempotent_duplicate" {
		t.Errorf("Expected status 'idempotent_duplicate', got %v", dupResp["status"])
	}
	if dupResp["execution_id"] != firstExec.ID {
		t.Errorf("Expected execution ID %s, got %v", firstExec.ID, dupResp["execution_id"])
	}

	fmt.Println("✓ Webhook HMAC signature verification and Idempotency key verified successfully")
}
