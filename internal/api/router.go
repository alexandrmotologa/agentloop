package api

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/alexandrmotologa/agentloop/internal/cron"
	"github.com/alexandrmotologa/agentloop/internal/engine"
	"github.com/alexandrmotologa/agentloop/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// NewRouter constructs the Chi router with API routes, SSE, and embedded SPA file serving.
func NewRouter(db *store.DB, executor *engine.Executor, scheduler *cron.Scheduler, webFS fs.FS) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "Idempotency-Key", "X-Idempotency-Key", "X-Hub-Signature-256"},
		ExposedHeaders:   []string{"Link", "Content-Disposition"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	apiHandler := NewAPI(db, executor, scheduler)

	// Top-level webhook endpoint for standard webhook providers (e.g. /webhooks/invoice-flow)
	r.Post("/webhooks/{id}", apiHandler.HandleWebhook)

	// API Routes
	r.Route("/api", func(r chi.Router) {
		r.Get("/health", apiHandler.HandleHealth)

		// Workflows
		r.Get("/workflows", apiHandler.HandleListWorkflows)
		r.Post("/workflows", apiHandler.HandleCreateWorkflow)
		r.Get("/workflows/{id}", apiHandler.HandleGetWorkflow)
		r.Delete("/workflows/{id}", apiHandler.HandleDeleteWorkflow)
		r.Post("/workflows/{id}/run", apiHandler.HandleRunWorkflow)

		// Webhooks
		r.Post("/webhooks/{id}", apiHandler.HandleWebhook)

		// Executions
		r.Get("/executions", apiHandler.HandleListExecutions)
		r.Get("/executions/{id}", apiHandler.HandleGetExecution)
		r.Get("/executions/{id}/audit", apiHandler.HandleGetAuditLogs)
		r.Get("/executions/{id}/healing", apiHandler.HandleGetHealingLogs)
		r.Get("/executions/{id}/snapshots", apiHandler.HandleGetSnapshots)
		r.Post("/executions/{id}/rewind", apiHandler.HandleRewindExecution)

		// Breakpoint & Step Debugger
		r.Post("/executions/{id}/step", apiHandler.HandleStepExecution)
		r.Post("/executions/{id}/continue", apiHandler.HandleContinueExecution)
		r.Post("/executions/{id}/breakpoints", apiHandler.HandleSetBreakpoints)

		// Compliance Audit Report
		r.Get("/executions/{id}/report", apiHandler.HandleGetComplianceReport)

		// Cron Scheduler
		r.Get("/cron/jobs", apiHandler.HandleListCronJobs)

		// Dead-Letter Queue
		r.Get("/dlq", apiHandler.HandleListDLQItems)
		r.Get("/dlq/{id}", apiHandler.HandleGetDLQItem)
		r.Post("/dlq/{id}/resume", apiHandler.HandleResumeDLQItem)
		r.Post("/dlq/{id}/abort", apiHandler.HandleAbortDLQItem)

		// Server-Sent Events (SSE) stream
		r.Get("/events", HandleSSE(executor.Broker))
		r.Get("/executions/{id}/events", HandleSSE(executor.Broker))
	})

	// Web UI SPA Handler
	if webFS != nil {
		fileServer := http.FileServer(http.FS(webFS))
		r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
			path := strings.TrimPrefix(r.URL.Path, "/")
			if path == "" {
				path = "index.html"
			}

			// Check if file exists in webFS
			f, err := webFS.Open(path)
			if err == nil {
				_ = f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}

			// Fallback to index.html for client-side SPA routing
			indexFile, err := webFS.Open("index.html")
			if err != nil {
				http.NotFound(w, r)
				return
			}
			defer indexFile.Close()

			stat, _ := indexFile.Stat()
			http.ServeContent(w, r, "index.html", stat.ModTime(), indexFile.(readSeeker))
		})
	}

	return r
}

type readSeeker interface {
	Read(p []byte) (n int, err error)
	Seek(offset int64, whence int) (int64, error)
}
