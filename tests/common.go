package tests

import (
	"context"
	"testing"

	"github.com/temporalio/nexus-error-compat-tests/config"
	"github.com/temporalio/nexus-error-compat-tests/harness"
	"go.temporal.io/sdk/client"
)

// TestContext provides access to the test environment for individual tests
type TestContext struct {
	Config           config.TestConfig
	CallerClient     client.Client
	HandlerClient    client.Client
	CallerEndpoint   string
	CallerTaskQueue  string
	HandlerTaskQueue string
	CallerNamespace  string
	HandlerNamespace string
	Environment      *harness.TestEnvironment
}

// NewTestContext creates a new test context with the given configuration
// It sets up the complete test environment and registers cleanup with t.Cleanup
func NewTestContext(t *testing.T, cfg config.TestConfig) *TestContext {
	t.Helper()

	ctx := context.Background()

	// Setup test environment
	env, err := harness.SetupTestEnvironment(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}

	// Register cleanup
	t.Cleanup(func() {
		if err := env.Cleanup(); err != nil {
			t.Logf("Warning: cleanup failed: %v", err)
		}
	})

	return &TestContext{
		Config:           cfg,
		CallerClient:     env.CallerClient,
		HandlerClient:    env.HandlerClient,
		CallerEndpoint:   env.CallerEndpointName,
		CallerTaskQueue:  env.CallerTaskQueue,
		HandlerTaskQueue: env.HandlerTaskQueue,
		CallerNamespace:  env.CallerNamespace,
		HandlerNamespace: env.HandlerNamespace,
		Environment:      env,
	}
}
