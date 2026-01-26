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

// TestAsyncOperationSuccess tests successful execution of an async Nexus operation
func TestAsyncOperationSuccess(t *testing.T) {
	tc := NewTestContext(t, config.DefaultTestConfig())

	// Define caller workflow
	callerWorkflow := func(ctx workflow.Context, input string) (string, error) {
		c := workflow.NewNexusClient(tc.CallerEndpoint, "test-service")
		fut := c.ExecuteOperation(ctx, "async-op", input, workflow.NexusOperationOptions{})

		// For async operations, verify we get an operation execution (with token)
		var exec workflow.NexusOperationExecution
		if err := fut.GetNexusOperationExecution().Get(ctx, &exec); err != nil {
			return "", err
		}

		// Operation token should be non-empty for async operations
		if exec.OperationToken == "" {
			return "", temporal.NewApplicationError("expected non-empty operation token", "", nil)
		}

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

	t.Logf("Async operation completed successfully with result: %s", result)
}

// TestAsyncOperationFailure tests failure scenarios for async operations
func TestAsyncOperationFailure(t *testing.T) {
	tc := NewTestContext(t, config.DefaultTestConfig())

	// Define caller workflow
	callerWorkflow := func(ctx workflow.Context, input string) (string, error) {
		c := workflow.NewNexusClient(tc.CallerEndpoint, "test-service")
		op := nexus.NewSyncOperation("async-op", func(ctx context.Context, input string, opts nexus.StartOperationOptions) (string, error) {
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
		{"WorkflowFailure", "workflow-failure"},
		{"ApplicationError", "application-error"},
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

			t.Logf("Error scenario '%s' completed as expected: %v", tt.name, err)
		})
	}
}

// TestAsyncOperationCancellation tests cancellation of async operations
func TestAsyncOperationCancellation(t *testing.T) {
	tc := NewTestContext(t, config.DefaultTestConfig())

	// Define caller workflow with cancellation support
	callerWorkflow := func(ctx workflow.Context, input string) (string, error) {
		c := workflow.NewNexusClient(tc.CallerEndpoint, "test-service")
		op := nexus.NewSyncOperation[string, string]("async-op", func(ctx context.Context, input string, opts nexus.StartOperationOptions) (string, error) {
			panic("should not be called")
		})

		ctx, cancel := workflow.WithCancel(ctx)
		defer cancel()

		fut := c.ExecuteOperation(ctx, op, input, workflow.NexusOperationOptions{})

		// Set up signal handler to cancel operation
		ch := workflow.GetSignalChannel(ctx, "cancel-op")
		workflow.Go(ctx, func(ctx workflow.Context) {
			var action string
			ch.Receive(ctx, &action)
			if action == "wait-for-started" {
				// Wait for operation to start before canceling
				fut.GetNexusOperationExecution().Get(ctx, nil)
			}
			cancel()
		})

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

	t.Run("CancelAfterStarted", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		run, err := tc.CallerClient.ExecuteWorkflow(ctx,
			client.StartWorkflowOptions{
				TaskQueue: tc.CallerTaskQueue,
			},
			callerWorkflow, "wait-for-cancel")
		if err != nil {
			t.Fatalf("Failed to start workflow: %v", err)
		}

		// Give operation time to start
		time.Sleep(2 * time.Second)

		// Send cancel signal
		if err := tc.CallerClient.SignalWorkflow(ctx, run.GetID(), run.GetRunID(), "cancel-op", "wait-for-started"); err != nil {
			t.Fatalf("Failed to send signal: %v", err)
		}

		var result string
		err = run.Get(ctx, &result)
		if err == nil {
			t.Errorf("Expected cancellation error but got none")
			return
		}

		t.Logf("Cancellation test completed as expected: %v", err)
	})

	t.Run("CancelBeforeSent", func(t *testing.T) {
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

		// Send cancel signal immediately (before operation is scheduled)
		if err := tc.CallerClient.SignalWorkflow(ctx, run.GetID(), run.GetRunID(), "cancel-op", "immediate"); err != nil {
			t.Fatalf("Failed to send signal: %v", err)
		}

		var result string
		err = run.Get(ctx, &result)
		if err == nil {
			t.Errorf("Expected cancellation error but got none")
			return
		}

		t.Logf("Early cancellation test completed as expected: %v", err)
	})
}

// TestAsyncOperationEcho tests echo functionality for async operations
func TestAsyncOperationEcho(t *testing.T) {
	tc := NewTestContext(t, config.DefaultTestConfig())

	// Define caller workflow
	callerWorkflow := func(ctx workflow.Context, input string) (string, error) {
		c := workflow.NewNexusClient(tc.CallerEndpoint, "test-service")
		op := nexus.NewSyncOperation[string, string]("async-op", func(ctx context.Context, input string, opts nexus.StartOperationOptions) (string, error) {
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
	testInput := "hello from async caller"
	expectedOutput := "async echo: " + testInput

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

	t.Logf("Async echo test passed: %s", result)
}
