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
	"go.temporal.io/sdk/converter"
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
	encodeFailureAttributes := os.Getenv("ENCODE_FAILURE_ATTRIBUTES") == "true"
	dataConverter := converter.GetDefaultDataConverter()
	if encodeFailureAttributes {
		dataConverter = converter.NewCodecDataConverter(dataConverter, converter.NewZlibCodec(converter.ZlibCodecOptions{AlwaysEncode: true}))
	}

	if serverAddr == "" || namespace == "" || taskQueue == "" {
		log.Fatal("Missing required environment variables: HANDLER_SERVER_ADDR, HANDLER_NAMESPACE, HANDLER_TASK_QUEUE")
	}

	// Create Temporal client
	c, err := client.Dial(client.Options{
		HostPort:  serverAddr,
		Namespace: namespace,
		FailureConverter: temporal.NewDefaultFailureConverter(temporal.DefaultFailureConverterOptions{
			EncodeCommonAttributes: encodeFailureAttributes,
			DataConverter:          dataConverter,
		}),
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
	syncOp := nexus.NewSyncOperation("sync-op", func(ctx context.Context, outcome string, opts nexus.StartOperationOptions) (string, error) {
		// Handle different test scenarios
		switch outcome {
		case "operation-failed-error":
			return "", nexus.NewOperationFailedError("operation failed for test")
		case "operation-canceled-error-with-app-failure-cause":
			return "", &nexus.OperationError{
				State: nexus.OperationStateCanceled,
				Cause: temporal.NewApplicationError("application error for test", "TestErrorType", "details"),
			}
		case "handler-error-with-failure-error-cause":
			return "", &nexus.HandlerError{
				Type: nexus.HandlerErrorTypeBadRequest,
				Cause: &nexus.FailureError{
					Failure: nexus.Failure{
						Message: "cause",
					},
				},
			}
		case "wrapped-application-error":
			return "", &nexus.OperationError{
				State: nexus.OperationStateFailed,
				Cause: temporal.NewApplicationError("application error for test", "TestErrorType", "details"),
			}
		case "application-error":
			return "", temporal.NewApplicationErrorWithOptions("application error for test", "TestErrorType", temporal.ApplicationErrorOptions{
				NonRetryable: true,
				Details:      []any{"details"},
			})
		case "handler-error":
			return "", nexus.HandlerErrorf(nexus.HandlerErrorTypeBadRequest, "handler error for test")
		case "canceled":
			return "", nexus.NewOperationCanceledError("operation canceled for test")
		default:
			return "", nexus.HandlerErrorf(nexus.HandlerErrorTypeBadRequest, "invalid outcome: %q", outcome)
		}
	})

	// Async operation
	asyncOp := temporalnexus.NewWorkflowRunOperation(
		"async-op",
		AsyncHandlerWorkflow,
		func(ctx context.Context, input string, opts nexus.StartOperationOptions) (client.StartWorkflowOptions, error) {
			// To prevent local server from sending response to itself if both caller and handler are on
			// the same host since they will have a shared cluster ID.
			delete(opts.CallbackHeader, "source")
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
func AsyncHandlerWorkflow(ctx workflow.Context, outcome string) (string, error) {
	switch outcome {
	case "plain-error":
		return "", fmt.Errorf("workflow failed for test")
	case "application-error":
		return "", temporal.NewApplicationError("application error for test", "TestErrorType", "details")
	case "wait-for-cancel":
		// Wait indefinitely for cancellation
		return "", workflow.Await(ctx, func() bool { return false })
	default:
		return "", fmt.Errorf("invalid outcome: %q", outcome)
	}
}
