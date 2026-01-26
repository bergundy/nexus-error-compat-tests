package harness

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/api/nexus/v1"
	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"google.golang.org/protobuf/types/known/durationpb"
)

// EndpointInfo holds information about a created Nexus endpoint
type EndpointInfo struct {
	Name      string
	URLPrefix string
}

// CreateNamespace creates a namespace on the given client (idempotent)
func CreateNamespace(ctx context.Context, c client.Client, namespace string) error {
	_, err := c.WorkflowService().RegisterNamespace(ctx, &workflowservice.RegisterNamespaceRequest{
		Namespace: namespace,
		// Use a short retention period for tests
		WorkflowExecutionRetentionPeriod: durationpb.New(24 * time.Hour),
	})

	// Ignore AlreadyExists errors (idempotent)
	if err != nil {
		if _, ok := err.(*serviceerror.NamespaceAlreadyExists); !ok {
			return fmt.Errorf("failed to create namespace %s: %w", namespace, err)
		}
	}

	return nil
}

// CreateWorkerEndpoint creates a worker-targeted Nexus endpoint
func CreateWorkerEndpoint(ctx context.Context, c client.Client, name, namespace, taskQueue string) (*EndpointInfo, error) {
	req := &operatorservice.CreateNexusEndpointRequest{
		Spec: &nexus.EndpointSpec{
			Name: name,
			Target: &nexus.EndpointTarget{
				Variant: &nexus.EndpointTarget_Worker_{
					Worker: &nexus.EndpointTarget_Worker{
						Namespace: namespace,
						TaskQueue: taskQueue,
					},
				},
			},
		},
	}

	resp, err := c.OperatorService().CreateNexusEndpoint(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create worker endpoint %s: %w", name, err)
	}

	// Wait for endpoint to propagate
	time.Sleep(200 * time.Millisecond)

	return &EndpointInfo{
		Name:      name,
		URLPrefix: resp.Endpoint.UrlPrefix,
	}, nil
}

// CreateExternalEndpoint creates an external Nexus endpoint pointing to a URL
func CreateExternalEndpoint(ctx context.Context, c client.Client, name, targetURL string) (*EndpointInfo, error) {
	req := &operatorservice.CreateNexusEndpointRequest{
		Spec: &nexus.EndpointSpec{
			Name: name,
			Target: &nexus.EndpointTarget{
				Variant: &nexus.EndpointTarget_External_{
					External: &nexus.EndpointTarget_External{
						Url: targetURL,
					},
				},
			},
		},
	}

	resp, err := c.OperatorService().CreateNexusEndpoint(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create external endpoint %s: %w", name, err)
	}

	// Wait for endpoint to propagate
	time.Sleep(200 * time.Millisecond)

	return &EndpointInfo{
		Name:      name,
		URLPrefix: resp.Endpoint.UrlPrefix,
	}, nil
}
