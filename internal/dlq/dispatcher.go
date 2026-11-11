package dlq

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/alexandrmotologa/agentloop/internal/engine"
)

// Dispatcher broadcasts DLQ incident alerts to configured channels.
type Dispatcher struct {
	httpClient *http.Client
}

// NewDispatcher creates an alert dispatcher.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Dispatch sends notifications for a frozen execution.
func (d *Dispatcher) Dispatch(ctx context.Context, config engine.NotificationConfig, item *engine.DLQItem) {
	// 1. Console log
	log.Printf("[ALERT: DLQ ESCALATION] Execution %s paused at node '%s'. Reason: %s",
		item.ExecutionID, item.FailedNodeID, item.FailureReason)

	// 2. Webhook notification
	if config.Webhook != nil && config.Webhook.URL != "" {
		go d.sendWebhook(config.Webhook.URL, item)
	}

	// 3. Slack notification
	if config.Slack != nil && config.Slack.WebhookURL != "" {
		go d.sendSlack(config.Slack.WebhookURL, config.Slack.Channel, item)
	}
}

func (d *Dispatcher) sendWebhook(url string, item *engine.DLQItem) {
	payload := map[string]any{
		"event":         "dlq_execution_paused",
		"dlq_id":        item.ID,
		"execution_id":  item.ExecutionID,
		"failed_node":   item.FailedNodeID,
		"reason":        item.FailureReason,
		"error_details": item.ErrorDetails,
		"timestamp":     item.CreatedAt,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return
	}

	req, err := http.NewRequestWithContext(context.Background(), "POST", url, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.httpClient.Do(req)
	if err == nil {
		resp.Body.Close()
	}
}

func (d *Dispatcher) sendSlack(webhookURL string, channel string, item *engine.DLQItem) {
	text := fmt.Sprintf(":warning: *AgentLoop DLQ Alert*\nExecution `%s` suspended at node `%s`.\n*Reason:* %s\n*Error:* `%s`",
		item.ExecutionID, item.FailedNodeID, item.FailureReason, item.ErrorDetails)

	payload := map[string]any{
		"text": text,
	}
	if channel != "" {
		payload["channel"] = channel
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return
	}

	req, err := http.NewRequestWithContext(context.Background(), "POST", webhookURL, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.httpClient.Do(req)
	if err == nil {
		resp.Body.Close()
	}
}
