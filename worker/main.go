package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/nexus-rpc/sdk-go/nexus"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	temporalnexus "go.temporal.io/sdk/temporalnexus"
)

const (
	serviceName = "test-service"
)

func main() {
	// Get configuration from environment variables
	serverAddr := os.Getenv("HANDLER_SERVER_ADDR")
	namespace := os.Getenv("HANDLER_NAMESPACE")
	taskQueue := os.Getenv("HANDLER_TASK_QUEUE")

	if serverAddr == "" || namespace == "" || taskQueue == "" {
		log.Fatal("Missing required environment variables: HANDLER_SERVER_ADDR, HANDLER_NAMESPACE, HANDLER_TASK_QUEUE")
	}

	// Create Temporal client
	c, err := client.Dial(client.Options{
		HostPort:  serverAddr,
		Namespace: namespace,
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer c.Close()

	// Create worker
	w := worker.New(c, taskQueue, worker.Options{})

	// Register workflows
	w.RegisterWorkflow(AsyncHandlerWorkflow)

	// Register Nexus service with sync and async operations
	service := nexus.NewService(serviceName)

	// Sync operation
	syncOp := nexus.NewSyncOperation("sync-op", func(ctx context.Context, input string, opts nexus.StartOperationOptions) (string, error) {
		// TODO:
		// - OperationError and HandlerError with both message and cause
		// - Direct ApplicationError
		// - OperationError with CanceledError as cause
		// Handle different test scenarios
		switch input {
		case "success":
			return "success", nil
		case "operation-failure":
			return "", nexus.NewOperationFailedError("operation failed for test")
		case "application-error":
			return "", &nexus.OperationError{
				State: nexus.OperationStateFailed,
				Cause: temporal.NewApplicationError("application error for test", "TestErrorType", "details"),
			}
		case "handler-error":
			return "", nexus.HandlerErrorf(nexus.HandlerErrorTypeBadRequest, "handler error for test")
		case "canceled":
			return "", nexus.NewOperationCanceledError("operation canceled for test")
		default:
			return fmt.Sprintf("echo: %s", input), nil
		}
	})

	// Async operation
	asyncOp := temporalnexus.NewWorkflowRunOperation(
		"async-op",
		AsyncHandlerWorkflow,
		func(ctx context.Context, input string, opts nexus.StartOperationOptions) (client.StartWorkflowOptions, error) {
			// Use RequestID as workflow ID for idempotency
			return client.StartWorkflowOptions{
				ID: opts.RequestID,
			}, nil
		},
	)

	// Register operations
	if err := service.Register(syncOp); err != nil {
		log.Fatalf("Failed to register sync operation: %v", err)
	}
	if err := service.Register(asyncOp); err != nil {
		log.Fatalf("Failed to register async operation: %v", err)
	}

	// Register service with worker
	w.RegisterNexusService(service)

	// Start worker
	if err := w.Start(); err != nil {
		log.Fatalf("Failed to start worker: %v", err)
	}

	// Print ready marker
	fmt.Println("WORKER_READY")

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down worker...")
	w.Stop()
}

// AsyncHandlerWorkflow is the workflow that backs the async Nexus operation
func AsyncHandlerWorkflow(ctx workflow.Context, input string) (string, error) {
	switch input {
	case "success":
		return "success", nil
	case "workflow-failure":
		return "", fmt.Errorf("workflow failed for test")
	case "application-error":
		return "", temporal.NewApplicationError("application error for test", "TestErrorType", "details")
	case "wait-for-cancel":
		// Wait indefinitely for cancellation
		return "", workflow.Await(ctx, func() bool { return false })
	default:
		return fmt.Sprintf("async echo: %s", input), nil
	}
}
