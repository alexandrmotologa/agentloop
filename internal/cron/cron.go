package cron

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/alexandrmotologa/agentloop/internal/engine"
	"github.com/alexandrmotologa/agentloop/internal/store"
	"github.com/robfig/cron/v3"
)

// ScheduledJob describes a registered cron job.
type ScheduledJob struct {
	EntryID    cron.EntryID `json:"entry_id"`
	WorkflowID string       `json:"workflow_id"`
	Schedule   string       `json:"schedule"`
	NextRun    time.Time    `json:"next_run"`
	PrevRun    time.Time    `json:"prev_run"`
}

// Scheduler manages periodic execution of workflows with a configured schedule expression.
type Scheduler struct {
	db       *store.DB
	executor *engine.Executor
	cron     *cron.Cron
	mu       sync.RWMutex
	jobs     map[string]cron.EntryID // workflowID -> EntryID
}

// NewScheduler creates an automated workflow scheduler.
func NewScheduler(db *store.DB, executor *engine.Executor) *Scheduler {
	return &Scheduler{
		db:       db,
		executor: executor,
		cron:     cron.New(cron.WithParser(cron.NewParser(
			cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
		))),
		jobs: make(map[string]cron.EntryID),
	}
}

// Start boots up the cron engine and scans all stored workflows for schedules.
func (s *Scheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cron.Start()
	return s.reloadLocked()
}

// Stop gracefully terminates scheduled tasks.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cron.Stop()
}

// Reload refreshes active cron jobs based on the latest database state.
func (s *Scheduler) Reload() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.reloadLocked()
}

func (s *Scheduler) reloadLocked() error {
	// Clear existing jobs
	for wfID, entryID := range s.jobs {
		s.cron.Remove(entryID)
		delete(s.jobs, wfID)
	}

	workflows, err := s.db.ListWorkflows()
	if err != nil {
		return fmt.Errorf("failed to list workflows for cron scheduler: %w", err)
	}

	for _, wf := range workflows {
		scheduleExpr := wf.Schedule
		if scheduleExpr == "" {
			continue
		}

		targetWfID := wf.ID
		entryID, err := s.cron.AddFunc(scheduleExpr, func() {
			log.Printf("[Cron] Triggering scheduled execution for workflow: %s (%s)", targetWfID, scheduleExpr)
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
			defer cancel()

			initialState := map[string]any{
				"triggered_by": "cron",
				"cron_time":    time.Now().UTC().Format(time.RFC3339),
			}

			exec, err := s.executor.Start(ctx, targetWfID, initialState)
			if err != nil {
				log.Printf("[Cron] Error executing scheduled workflow %s: %v", targetWfID, err)
				return
			}
			log.Printf("[Cron] Workflow %s launched with execution %s (status: %s)", targetWfID, exec.ID, exec.Status)
		})

		if err != nil {
			log.Printf("[Cron] Failed to schedule workflow %s with expression '%s': %v", targetWfID, scheduleExpr, err)
			continue
		}

		s.jobs[targetWfID] = entryID
		log.Printf("[Cron] Registered schedule '%s' for workflow %s", scheduleExpr, targetWfID)
	}

	return nil
}

// ListJobs returns the currently scheduled cron jobs.
func (s *Scheduler) ListJobs() []ScheduledJob {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []ScheduledJob
	entries := s.cron.Entries()
	entryMap := make(map[cron.EntryID]cron.Entry)
	for _, e := range entries {
		entryMap[e.ID] = e
	}

	for wfID, entryID := range s.jobs {
		if entry, ok := entryMap[entryID]; ok {
			result = append(result, ScheduledJob{
				EntryID:    entryID,
				WorkflowID: wfID,
				NextRun:    entry.Next,
				PrevRun:    entry.Prev,
			})
		}
	}

	return result
}
