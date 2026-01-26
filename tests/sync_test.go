package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nexus-rpc/sdk-go/nexus"
	"github.com/temporalio/nexus-error-compat-tests/config"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

// TestSyncOperationSuccess tests successful execution of a sync Nexus operation
func TestSyncOperationSuccess(t *testing.T) {
	tc := NewTestContext(t, config.DefaultTestConfig())

	// Define caller workflow
	callerWorkflow := func(ctx workflow.Context, input string) (string, error) {
		c := workflow.NewNexusClient(tc.CallerEndpoint, "test-service")
		op := nexus.NewSyncOperation[string, string]("sync-op", func(ctx context.Context, input string, opts nexus.StartOperationOptions) (string, error) {
			// Handler is in separate process, this is just for type signature
			panic("should not be called")
		})

		fut := c.ExecuteOperation(ctx, op, input, workflow.NexusOperationOptions{})

		var result string
		if err := fut.Get(ctx, &result); err != nil {
			return "", err
		}
		return result, nil
	}

	// Start caller worker
	w := worker.New(tc.CallerClient, tc.CallerTaskQueue, worker.Options{})
	w.RegisterWorkflow(callerWorkflow)
	if err := w.Start(); err != nil {
		t.Fatalf("Failed to start worker: %v", err)
	}
	defer w.Stop()

	// Execute workflow
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	run, err := tc.CallerClient.ExecuteWorkflow(ctx,
		client.StartWorkflowOptions{
			TaskQueue: tc.CallerTaskQueue,
		},
		callerWorkflow, "success")
	if err != nil {
		t.Fatalf("Failed to start workflow: %v", err)
	}

	var result string
	if err := run.Get(ctx, &result); err != nil {
		t.Fatalf("Workflow execution failed: %v", err)
	}

	if result != "success" {
		t.Errorf("Expected result 'success', got '%s'", result)
	}

	t.Logf("Sync operation completed successfully with result: %s", result)
}

// TestSyncOperationFailure tests operation failure scenarios
func TestSyncOperationFailure(t *testing.T) {
	tc := NewTestContext(t, config.DefaultTestConfig())

	// Define caller workflow
	callerWorkflow := func(ctx workflow.Context, input string) (string, error) {
		c := workflow.NewNexusClient(tc.CallerEndpoint, "test-service")
		op := nexus.NewSyncOperation[string, string]("sync-op", func(ctx context.Context, input string, opts nexus.StartOperationOptions) (string, error) {
			panic("should not be called")
		})

		fut := c.ExecuteOperation(ctx, op, input, workflow.NexusOperationOptions{})

		var result string
		return result, fut.Get(ctx, &result)
	}

	// Start caller worker
	w := worker.New(tc.CallerClient, tc.CallerTaskQueue, worker.Options{})
	w.RegisterWorkflow(callerWorkflow)
	if err := w.Start(); err != nil {
		t.Fatalf("Failed to start worker: %v", err)
	}
	defer w.Stop()

	tests := []struct {
		name  string
		input string
	}{
		{"OperationFailure", "operation-failure"},
		{"ApplicationError", "application-error"},
		{"HandlerError", "handler-error"},
		{"Canceled", "canceled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			run, err := tc.CallerClient.ExecuteWorkflow(ctx,
				client.StartWorkflowOptions{
					TaskQueue: tc.CallerTaskQueue,
				},
				callerWorkflow, tt.input)
			if err != nil {
				t.Fatalf("Failed to start workflow: %v", err)
			}

			var result string
			err = run.Get(ctx, &result)
			if err == nil {
				t.Errorf("Expected error but got none, result: %s", result)
				return
			}

			// Verify we got a workflow execution error
			var execErr *temporal.WorkflowExecutionError
			if !errors.As(err, &execErr) {
				t.Errorf("Expected WorkflowExecutionError, got: %v", err)
				return
			}

			// Check for NexusOperationError
			unwrapped := execErr.Unwrap()
			var nexusErr *temporal.NexusOperationError
			if !errors.As(unwrapped, &nexusErr) {
				t.Errorf("Expected NexusOperationError, got: %v", unwrapped)
				return
			}

			// Verify error metadata
			if nexusErr.Endpoint != tc.CallerEndpoint {
				t.Errorf("Expected endpoint %s, got %s", tc.CallerEndpoint, nexusErr.Endpoint)
			}
			if nexusErr.Service != "test-service" {
				t.Errorf("Expected service 'test-service', got %s", nexusErr.Service)
			}
			if nexusErr.Operation != "sync-op" {
				t.Errorf("Expected operation 'sync-op', got %s", nexusErr.Operation)
			}

			t.Logf("Error scenario '%s' completed as expected: %v", tt.name, err)
		})
	}
}

// TestSyncOperationEcho tests echo functionality with arbitrary input
func TestSyncOperationEcho(t *testing.T) {
	tc := NewTestContext(t, config.DefaultTestConfig())

	// Define caller workflow
	callerWorkflow := func(ctx workflow.Context, input string) (string, error) {
		c := workflow.NewNexusClient(tc.CallerEndpoint, "test-service")
		op := nexus.NewSyncOperation[string, string]("sync-op", func(ctx context.Context, input string, opts nexus.StartOperationOptions) (string, error) {
			panic("should not be called")
		})

		fut := c.ExecuteOperation(ctx, op, input, workflow.NexusOperationOptions{})

		var result string
		if err := fut.Get(ctx, &result); err != nil {
			return "", err
		}
		return result, nil
	}

	// Start caller worker
	w := worker.New(tc.CallerClient, tc.CallerTaskQueue, worker.Options{})
	w.RegisterWorkflow(callerWorkflow)
	if err := w.Start(); err != nil {
		t.Fatalf("Failed to start worker: %v", err)
	}
	defer w.Stop()

	// Test with arbitrary input
	testInput := "hello from caller"
	expectedOutput := "echo: " + testInput

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	run, err := tc.CallerClient.ExecuteWorkflow(ctx,
		client.StartWorkflowOptions{
			TaskQueue: tc.CallerTaskQueue,
		},
		callerWorkflow, testInput)
	if err != nil {
		t.Fatalf("Failed to start workflow: %v", err)
	}

	var result string
	if err := run.Get(ctx, &result); err != nil {
		t.Fatalf("Workflow execution failed: %v", err)
	}

	if result != expectedOutput {
		t.Errorf("Expected result '%s', got '%s'", expectedOutput, result)
	}

	t.Logf("Echo test passed: %s", result)
}
