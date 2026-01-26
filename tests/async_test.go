package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/nexus-error-compat-tests/config"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

// TestAsyncOperationFailure tests failure scenarios for async operations
func TestAsyncOperationFailure(t *testing.T) {
	tc := NewTestContext(t, config.DefaultTestConfig())

	// Define caller workflow
	callerWorkflow := func(ctx workflow.Context, input string) (string, error) {
		c := workflow.NewNexusClient(tc.CallerEndpoint, "test-service")
		fut := c.ExecuteOperation(ctx, "async-op", input, workflow.NexusOperationOptions{})

		var result string
		return result, fut.Get(ctx, &result)
	}

	// Start caller worker
	w := worker.New(tc.CallerClient, tc.CallerTaskQueue, worker.Options{})
	w.RegisterWorkflow(callerWorkflow)
	require.NoError(t, w.Start())
	defer w.Stop()

	tests := []struct {
		name     string
		outcome  string
		checkErr func(t *testing.T, err error)
	}{
		{
			"PlainError",
			"plain-error",
			func(t *testing.T, err error) {
				var nexusErr *temporal.NexusOperationError
				require.ErrorAs(t, err, &nexusErr)

				// Verify error metadata
				require.Equal(t, tc.CallerEndpoint, nexusErr.Endpoint)
				require.Equal(t, "test-service", nexusErr.Service)
				require.Equal(t, "async-op", nexusErr.Operation)

				var appErr *temporal.ApplicationError
				require.ErrorAs(t, nexusErr.Cause, &appErr)
				require.Equal(t, "workflow failed for test", appErr.Message())
			},
		},
		{
			"ApplicationError",
			"application-error",
			func(t *testing.T, err error) {
				var nexusErr *temporal.NexusOperationError
				require.ErrorAs(t, err, &nexusErr)

				var appErr *temporal.ApplicationError
				require.ErrorAs(t, nexusErr.Cause, &appErr)
				require.Equal(t, "application error for test", appErr.Message())
				require.Equal(t, "TestErrorType", appErr.Type())
				var details string
				require.NoError(t, appErr.Details(&details))
				require.Equal(t, "details", details)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			run, err := tc.CallerClient.ExecuteWorkflow(ctx,
				client.StartWorkflowOptions{
					TaskQueue: tc.CallerTaskQueue,
				},
				callerWorkflow, tt.outcome)
			require.NoError(t, err)

			var result string
			err = run.Get(ctx, &result)

			// Verify we got a workflow execution error
			var execErr *temporal.WorkflowExecutionError
			require.ErrorAs(t, err, &execErr)

			unwrapped := execErr.Unwrap()
			tt.checkErr(t, unwrapped)
		})
	}
}

// TestAsyncOperationCancellation tests cancellation of async operations
func TestAsyncOperationCancellation(t *testing.T) {
	tc := NewTestContext(t, config.DefaultTestConfig())

	// Define caller workflow with cancellation support
	callerWorkflow := func(ctx workflow.Context, input string) (string, error) {
		c := workflow.NewNexusClient(tc.CallerEndpoint, "test-service")
		ctx, cancel := workflow.WithCancel(ctx)
		defer cancel()

		fut := c.ExecuteOperation(ctx, "async-op", input, workflow.NexusOperationOptions{})

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
	require.NoError(t, w.Start())
	defer w.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	run, err := tc.CallerClient.ExecuteWorkflow(ctx,
		client.StartWorkflowOptions{
			TaskQueue: tc.CallerTaskQueue,
		},
		callerWorkflow, "wait-for-cancel")
	require.NoError(t, err)

	// Give operation time to start
	time.Sleep(2 * time.Second)

	// Send cancel signal
	err = tc.CallerClient.SignalWorkflow(ctx, run.GetID(), run.GetRunID(), "cancel-op", "wait-for-started")
	require.NoError(t, err)

	var result string
	err = run.Get(ctx, &result)
	require.ErrorAs(t, err, new(*temporal.CanceledError))
}
