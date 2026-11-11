package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/alexandrmotologa/agentloop/internal/engine"
	"github.com/go-chi/chi/v5"
)

// HandleSSE streams real-time execution events over Server-Sent Events.
func HandleSSE(broker *engine.EventBroker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		filterExecID := chi.URLParam(r, "id")
		if filterExecID == "" {
			filterExecID = r.URL.Query().Get("execution_id")
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		eventChan := broker.Subscribe()
		defer broker.Unsubscribe(eventChan)

		// Send initial connected event
		fmt.Fprintf(w, "event: connected\ndata: {\"status\":\"connected\"}\n\n")
		flusher.Flush()

		ctx := r.Context()
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-eventChan:
				if !ok {
					return
				}

				// Filter by execution ID if requested
				if filterExecID != "" && event.ExecutionID != filterExecID {
					continue
				}

				bytes, err := json.Marshal(event)
				if err != nil {
					continue
				}

				fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, string(bytes))
				flusher.Flush()
			}
		}
	}
}
