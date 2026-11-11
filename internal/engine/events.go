package engine

import (
	"sync"
	"time"
)

// ExecutionEventType defines the classification of real-time SSE events.
type ExecutionEventType string

const (
	EventWorkflowStarted   ExecutionEventType = "workflow_started"
	EventNodeStart         ExecutionEventType = "node_start"
	EventNodeComplete      ExecutionEventType = "node_complete"
	EventSelfHealing       ExecutionEventType = "self_healing_attempt"
	EventDLQPaused         ExecutionEventType = "dlq_paused"
	EventExecutionResumed  ExecutionEventType = "execution_resumed"
	EventBreakpointHit     ExecutionEventType = "breakpoint_hit"
	EventBudgetExceeded    ExecutionEventType = "budget_exceeded"
	EventModelCascaded     ExecutionEventType = "model_cascaded"
	EventWorkflowCompleted ExecutionEventType = "workflow_completed"
	EventWorkflowFailed    ExecutionEventType = "workflow_failed"
)

// ExecutionEvent is published during workflow execution.
type ExecutionEvent struct {
	ExecutionID string             `json:"execution_id"`
	WorkflowID  string             `json:"workflow_id"`
	Type        ExecutionEventType `json:"type"`
	NodeID      string             `json:"node_id,omitempty"`
	Status      string             `json:"status,omitempty"`
	Message     string             `json:"message,omitempty"`
	Payload     any                `json:"payload,omitempty"`
	Timestamp   time.Time          `json:"timestamp"`
}

// EventBroker broadcasts execution events to subscribers (such as SSE streams).
type EventBroker struct {
	mu          sync.RWMutex
	subscribers map[chan ExecutionEvent]struct{}
}

// NewEventBroker creates a thread-safe event broker.
func NewEventBroker() *EventBroker {
	return &EventBroker{
		subscribers: make(map[chan ExecutionEvent]struct{}),
	}
}

// Subscribe returns a channel that receives real-time execution events.
func (b *EventBroker) Subscribe() chan ExecutionEvent {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan ExecutionEvent, 64)
	b.subscribers[ch] = struct{}{}
	return ch
}

// Unsubscribe removes a channel from the subscriber list and closes it.
func (b *EventBroker) Unsubscribe(ch chan ExecutionEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.subscribers[ch]; exists {
		delete(b.subscribers, ch)
		close(ch)
	}
}

// Publish sends an event to all active subscribers without blocking.
func (b *EventBroker) Publish(event ExecutionEvent) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	for ch := range b.subscribers {
		select {
		case ch <- event:
		default:
			// Non-blocking drop if subscriber channel buffer is full
		}
	}
}
