package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/alexandrmotologa/agentloop/internal/api"
	"github.com/alexandrmotologa/agentloop/internal/dlq"
	"github.com/alexandrmotologa/agentloop/internal/engine"
	"github.com/alexandrmotologa/agentloop/internal/store"
	"github.com/alexandrmotologa/agentloop/web"
)

const version = "1.0.0"

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "serve":
		cmdServe(os.Args[2:])
	case "run":
		cmdRun(os.Args[2:])
	case "dlq":
		cmdDLQ(os.Args[2:])
	case "resume":
		cmdResume(os.Args[2:])
	case "validate":
		cmdValidate(os.Args[2:])
	case "version", "--version", "-v":
		fmt.Printf("agentloop version %s (Go runtime %s)\n", version, "1.24")
	case "help", "--help", "-h":
		printHelp()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`agentloop - Self-Healing Autonomous Enterprise Workflow Engine

Usage:
  agentloop <command> [arguments]

Commands:
  serve       Start the agentloop daemon with embedded React 19 cockpit
  run         Execute a workflow YAML file synchronously from CLI
  dlq         Inspect or list Dead-Letter Queue items
  resume      Resume a paused DLQ execution with optional operator guidance
  validate    Validate a workflow YAML file without executing it
  version     Show version information
  help        Show this help message

Use "agentloop <command> --help" for more information about a command.`)
}

func cmdServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	port := fs.Int("port", 8080, "HTTP server port")
	dbPath := fs.String("db", "./agentloop.db", "SQLite database file path")
	_ = fs.Parse(args)

	db, err := store.OpenDB(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	dlqMgr := dlq.NewManager(db)
	executor := engine.NewExecutor(db, dlqMgr)

	webFS, err := web.GetFS()
	if err != nil {
		fmt.Printf("Warning: embedded web UI assets not found: %v\n", err)
	}

	addr := fmt.Sprintf("0.0.0.0:%d", *port)
	srv, err := api.StartServer(addr, db, executor, webFS)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start server: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✓ agentloop server running at http://localhost:%d\n", *port)
	fmt.Printf("✓ Cockpit console available in browser\n")
	fmt.Printf("✓ Press Ctrl+C to terminate\n\n")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nShutting down agentloop server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

func cmdRun(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	filePath := fs.String("file", "", "Path to workflow YAML file (required)")
	inputJSON := fs.String("input", "{}", "JSON input payload for the workflow")
	dbPath := fs.String("db", ":memory:", "SQLite database file path (defaults to in-memory)")
	_ = fs.Parse(args)

	if *filePath == "" {
		fmt.Fprintln(os.Stderr, "Error: --file argument is required")
		os.Exit(1)
	}

	dag, err := engine.LoadWorkflowFile(*filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading workflow YAML: %v\n", err)
		os.Exit(1)
	}

	var input map[string]any
	if err := json.Unmarshal([]byte(*inputJSON), &input); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing --input JSON: %v\n", err)
		os.Exit(1)
	}

	db, err := store.OpenDB(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Save workflow spec to database
	rawYAML, _ := os.ReadFile(*filePath)
	_ = db.SaveWorkflow(dag.Spec, string(rawYAML))

	dlqMgr := dlq.NewManager(db)
	executor := engine.NewExecutor(db, dlqMgr)

	fmt.Printf("\n[ENGINE] Starting execution for workflow '%s' (%s)...\n", dag.Spec.Workflow.Name, dag.Spec.Workflow.ID)
	startTime := time.Now()

	// Subscribe to events for live console output
	eventChan := executor.Broker.Subscribe()
	defer executor.Broker.Unsubscribe(eventChan)

	done := make(chan struct{})
	go func() {
		for event := range eventChan {
			switch event.Type {
			case engine.EventNodeStart:
				fmt.Printf("  -> [START] Node: %-25s\n", event.NodeID)
			case engine.EventNodeComplete:
				fmt.Printf("  ✓  [DONE]  Node: %-25s (%s)\n", event.NodeID, event.Message)
			case engine.EventSelfHealing:
				fmt.Printf("  ⚡ [HEAL]  Node: %-25s (%s)\n", event.NodeID, event.Message)
			case engine.EventDLQPaused:
				fmt.Printf("  ⏸  [DLQ]   Execution paused at node %s: %s\n", event.NodeID, event.Message)
			case engine.EventWorkflowCompleted:
				fmt.Printf("  ★  [SUCCESS] Workflow execution completed successfully\n")
			case engine.EventWorkflowFailed:
				fmt.Printf("  ✗  [FAILED] Workflow execution failed: %s\n", event.Message)
			}
		}
		close(done)
	}()

	exec, err := executor.StartSpec(context.Background(), dag.Spec, input)
	duration := time.Since(startTime)

	if err != nil && exec == nil {
		fmt.Fprintf(os.Stderr, "\nExecution error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nExecution Summary:\n")
	fmt.Printf("  ID:        %s\n", exec.ID)
	fmt.Printf("  Status:    %s\n", exec.Status)
	fmt.Printf("  Duration:  %v\n", duration)
	fmt.Printf("  Last Node: %s\n", exec.CurrentNodeID)

	stateBytes, _ := json.MarshalIndent(exec.State, "  ", "  ")
	fmt.Printf("\nFinal State:\n  %s\n\n", string(stateBytes))

	if exec.Status == engine.StatusPausedDLQ {
		item, _ := dlqMgr.GetItem(exec.ID)
		if item != nil {
			fmt.Printf("Work item frozen in Dead-Letter Queue (DLQ ID: %s)\n", item.ID)
			fmt.Printf("Resume with: agentloop resume --dlq-id %s --instruction \"...\"\n\n", item.ID)
		}
	}
}

func cmdDLQ(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: agentloop dlq <list|inspect> [options]")
		os.Exit(1)
	}

	subcommand := args[0]
	dbPath := "./agentloop.db"

	db, err := store.OpenDB(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database at %s: %v\n", dbPath, err)
		os.Exit(1)
	}
	defer db.Close()

	dlqMgr := dlq.NewManager(db)

	switch subcommand {
	case "list":
		fs := flag.NewFlagSet("dlq list", flag.ExitOnError)
		status := fs.String("status", "", "Filter status: PENDING, RESOLVED, or empty for all")
		_ = fs.Parse(args[1:])

		items, err := dlqMgr.ListItems(*status)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing DLQ items: %v\n", err)
			os.Exit(1)
		}

		if len(items) == 0 {
			fmt.Println("No DLQ items found.")
			return
		}

		fmt.Printf("\n%-14s %-16s %-22s %-28s %-10s\n", "DLQ ID", "EXECUTION ID", "FAILED NODE", "FAILURE REASON", "STATUS")
		fmt.Println(strings.Repeat("-", 95))
		for _, item := range items {
			st := "PENDING"
			if item.ResolvedAt != nil {
				st = "RESOLVED"
			}
			reason := item.FailureReason
			if len(reason) > 26 {
				reason = reason[:26] + ".."
			}
			fmt.Printf("%-14s %-16s %-22s %-28s %-10s\n",
				item.ID, item.ExecutionID, item.FailedNodeID, reason, st)
		}
		fmt.Println()

	case "inspect":
		fs := flag.NewFlagSet("dlq inspect", flag.ExitOnError)
		id := fs.String("id", "", "DLQ Item ID (required)")
		_ = fs.Parse(args[1:])

		if *id == "" {
			fmt.Fprintln(os.Stderr, "Error: --id argument is required")
			os.Exit(1)
		}

		item, err := dlqMgr.GetItem(*id)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching DLQ item: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("\nDLQ Item Details:\n")
		fmt.Printf("  ID:           %s\n", item.ID)
		fmt.Printf("  Execution ID: %s\n", item.ExecutionID)
		fmt.Printf("  Failed Node:  %s\n", item.FailedNodeID)
		fmt.Printf("  Reason:       %s\n", item.FailureReason)
		fmt.Printf("  Error:        %s\n", item.ErrorDetails)
		fmt.Printf("  Created At:   %s\n", item.CreatedAt.Format(time.RFC3339))
		if item.ResolvedAt != nil {
			fmt.Printf("  Resolved At:  %s by %s (Action: %s)\n", item.ResolvedAt.Format(time.RFC3339), item.ResolvedBy, item.ResolutionAction)
			if item.OperatorInstruction != "" {
				fmt.Printf("  Instruction:  %s\n", item.OperatorInstruction)
			}
		}

		var snapshotObj any
		if err := json.Unmarshal([]byte(item.SnapshotStateJSON), &snapshotObj); err == nil {
			snapBytes, _ := json.MarshalIndent(snapshotObj, "  ", "  ")
			fmt.Printf("\nFrozen State Snapshot:\n  %s\n\n", string(snapBytes))
		}
	}
}

func cmdResume(args []string) {
	fs := flag.NewFlagSet("resume", flag.ExitOnError)
	dlqID := fs.String("dlq-id", "", "DLQ Item ID (required)")
	instruction := fs.String("instruction", "", "Natural language operator guidance for the agent")
	patchJSON := fs.String("patch", "{}", "JSON state patch to merge into execution state")
	retry := fs.Bool("retry", false, "Retry the currently failed node instead of proceeding to next")
	dbPath := fs.String("db", "./agentloop.db", "SQLite database file path")
	_ = fs.Parse(args)

	if *dlqID == "" {
		fmt.Fprintln(os.Stderr, "Error: --dlq-id argument is required")
		os.Exit(1)
	}

	var patch map[string]any
	if err := json.Unmarshal([]byte(*patchJSON), &patch); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing --patch JSON: %v\n", err)
		os.Exit(1)
	}

	db, err := store.OpenDB(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	dlqMgr := dlq.NewManager(db)
	executor := engine.NewExecutor(db, dlqMgr)

	fmt.Printf("\nResuming execution for DLQ item %s...\n", *dlqID)
	exec, err := executor.Resume(context.Background(), *dlqID, *instruction, patch, *retry)
	if err != nil && exec == nil {
		fmt.Fprintf(os.Stderr, "Error resuming execution: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Execution %s finished with status: %s\n\n", exec.ID, exec.Status)
}

func cmdValidate(args []string) {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	filePath := fs.String("file", "", "Path to workflow YAML file (required)")
	_ = fs.Parse(args)

	if *filePath == "" {
		fmt.Fprintln(os.Stderr, "Error: --file argument is required")
		os.Exit(1)
	}

	dag, err := engine.LoadWorkflowFile(*filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ Validation FAILED: %v\n", err)
		os.Exit(1)
	}

	order, err := dag.TopologicalSort()
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ DAG Cycle Detection FAILED: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Workflow '%s' (%s) is valid.\n", dag.Spec.Workflow.Name, dag.Spec.Workflow.ID)
	fmt.Printf("✓ DAG contains %d nodes, zero cycles detected.\n", len(dag.Nodes))
	fmt.Printf("✓ Topological execution order:\n")
	for i, nodeID := range order {
		node := dag.Nodes[nodeID]
		fmt.Printf("  %d. %-24s [%s]\n", i+1, node.ID, node.Type)
	}
	fmt.Println()
}
