package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nexus-rpc/sdk-go/nexus"
	"github.com/stretchr/testify/require"
	"github.com/temporalio/nexus-error-compat-tests/config"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

// TestSyncOperationFailure tests operation failure scenarios
func TestSyncOperationFailure(t *testing.T) {
	tc := NewTestContext(t, config.DefaultTestConfig())

	// Define caller workflow
	callerWorkflow := func(ctx workflow.Context, input string) (string, error) {
		c := workflow.NewNexusClient(tc.CallerEndpoint, "test-service")
		fut := c.ExecuteOperation(ctx, "sync-op", input, workflow.NexusOperationOptions{})

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
			"OperationFailure",
			"operation-failed-error",
			func(t *testing.T, err error) {
				var nexusErr *temporal.NexusOperationError
				require.ErrorAs(t, err, &nexusErr)

				// Verify error metadata
				require.Equal(t, tc.CallerEndpoint, nexusErr.Endpoint)
				require.Equal(t, "test-service", nexusErr.Service)
				require.Equal(t, "sync-op", nexusErr.Operation)
				var appErr *temporal.ApplicationError
				require.ErrorAs(t, nexusErr.Cause, &appErr)
				require.Equal(t, "operation failed for test", appErr.Message())
			},
		},
		{
			"WrappedApplicationError",
			"wrapped-application-error",
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
		{
			"ApplicationError",
			"application-error",
			func(t *testing.T, err error) {
				var nexusErr *temporal.NexusOperationError
				require.ErrorAs(t, err, &nexusErr)
				// ApplicationError expected to be wrapped with HandlerError on the handler SDK.
				var handlerErr *nexus.HandlerError
				require.ErrorAs(t, nexusErr.Cause, &handlerErr)
				require.Equal(t, nexus.HandlerErrorTypeInternal, handlerErr.Type)
				require.False(t, handlerErr.Retryable())
				var appErr *temporal.ApplicationError
				require.ErrorAs(t, handlerErr.Cause, &appErr)
				require.Equal(t, "application error for test", appErr.Message())
				require.Equal(t, "TestErrorType", appErr.Type())
				var details string
				require.NoError(t, appErr.Details(&details))
				require.Equal(t, "details", details)
			},
		},
		{
			"HandlerError",
			"handler-error",
			func(t *testing.T, err error) {
				var nexusErr *temporal.NexusOperationError
				require.ErrorAs(t, err, &nexusErr)
				var handlerErr *nexus.HandlerError
				require.ErrorAs(t, nexusErr.Cause, &handlerErr)
				require.Equal(t, nexus.HandlerErrorTypeBadRequest, handlerErr.Type)
				require.False(t, handlerErr.Retryable())
				// Old behavior
				var appErr *temporal.ApplicationError
				require.ErrorAs(t, handlerErr.Cause, &appErr)
				require.Equal(t, "handler error for test", appErr.Message())
			},
		},
		{
			"Canceled",
			"canceled",
			func(t *testing.T, err error) {
				// The Go SDK unwraps workflow errors to check for cancelation even if the workflow was
				// never canceled, losing the error chain, Nexus operation errors are treated the same
				// as other workflow errors for consistency.
				var canceledErr *temporal.CanceledError
				require.ErrorAs(t, err, &canceledErr)
			},
		},
		{
			"OperationCancelationWithDetails",
			"operation-canceled-error-with-details",
			func(t *testing.T, err error) {
				var canceledErr *temporal.CanceledError
				require.ErrorAs(t, err, &canceledErr)
				require.Equal(t, "canceled", canceledErr.Error())
				require.False(t, canceledErr.HasDetails())
				var appErr *temporal.ApplicationError
				require.ErrorAs(t, errors.Unwrap(canceledErr), &appErr)
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
