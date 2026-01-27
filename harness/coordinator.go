package harness

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/google/uuid"
	"github.com/temporalio/nexus-error-compat-tests/config"
	"go.temporal.io/sdk/client"
)

// TestEnvironment represents the complete test environment with all resources
type TestEnvironment struct {
	CallerClient       client.Client
	HandlerClient      client.Client
	CallerEndpointName string
	CallerTaskQueue    string
	HandlerTaskQueue   string
	CallerNamespace    string
	HandlerNamespace   string
	CallerServer       *ServerProcess
	HandlerServer      *ServerProcess
	HandlerWorker      *WorkerProcess
}

// SetupTestEnvironment orchestrates the complete startup sequence for the test environment
func SetupTestEnvironment(ctx context.Context, cfg config.TestConfig) (*TestEnvironment, error) {
	env := &TestEnvironment{
		CallerNamespace:  cfg.CallerNamespace,
		HandlerNamespace: cfg.HandlerNamespace,
	}

	// Setup signal handling for cleanup on interrupt
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nReceived interrupt signal, cleaning up...")
		env.Cleanup()
		os.Exit(1)
	}()

	// Step 1: Start caller server
	fmt.Printf("Starting caller server at %s...\n", cfg.CallerServer.GRPCAddr)
	callerServer, err := StartServer(ctx, cfg.CallerServer, "caller")
	if err != nil {
		return nil, fmt.Errorf("failed to start caller server: %w", err)
	}
	env.CallerServer = callerServer
	fmt.Printf("Caller server ready (logs: %s)\n", callerServer.LogPath())

	// Step 2: Start handler server
	fmt.Printf("Starting handler server at %s...\n", cfg.HandlerServer.GRPCAddr)
	handlerServer, err := StartServer(ctx, cfg.HandlerServer, "handler")
	if err != nil {
		env.Cleanup()
		return nil, fmt.Errorf("failed to start handler server: %w", err)
	}
	env.HandlerServer = handlerServer
	fmt.Printf("Handler server ready (logs: %s)\n", handlerServer.LogPath())

	// Step 3: Create caller client
	fmt.Printf("Creating caller client for namespace %s...\n", cfg.CallerNamespace)
	callerClient, err := client.Dial(client.Options{
		HostPort:  cfg.CallerServer.GRPCAddr,
		Namespace: cfg.CallerNamespace,
	})
	if err != nil {
		env.Cleanup()
		return nil, fmt.Errorf("failed to create caller client: %w", err)
	}
	env.CallerClient = callerClient

	// Step 4: Create handler client
	fmt.Printf("Creating handler client for namespace %s...\n", cfg.HandlerNamespace)
	handlerClient, err := client.Dial(client.Options{
		HostPort:  cfg.HandlerServer.GRPCAddr,
		Namespace: cfg.HandlerNamespace,
	})
	if err != nil {
		env.Cleanup()
		return nil, fmt.Errorf("failed to create handler client: %w", err)
	}
	env.HandlerClient = handlerClient

	// Step 5: Create caller namespace
	fmt.Printf("Creating caller namespace %s...\n", cfg.CallerNamespace)
	if err := CreateNamespace(ctx, callerClient, cfg.CallerNamespace); err != nil {
		env.Cleanup()
		return nil, fmt.Errorf("failed to create caller namespace: %w", err)
	}

	// Step 6: Create handler namespace
	fmt.Printf("Creating handler namespace %s...\n", cfg.HandlerNamespace)
	if err := CreateNamespace(ctx, handlerClient, cfg.HandlerNamespace); err != nil {
		env.Cleanup()
		return nil, fmt.Errorf("failed to create handler namespace: %w", err)
	}

	// Step 7: Generate unique names for task queues and endpoints
	suffix := uuid.New().String()[:8]
	env.HandlerTaskQueue = fmt.Sprintf("handler-tq-%s", suffix)
	env.CallerTaskQueue = fmt.Sprintf("caller-tq-%s", suffix)
	handlerEndpointName := fmt.Sprintf("handler-ep-%s", suffix)
	env.CallerEndpointName = fmt.Sprintf("caller-ep-%s", suffix)

	// Step 8: Create worker-targeted endpoint on handler namespace
	fmt.Printf("Creating worker endpoint %s on handler namespace...\n", handlerEndpointName)
	handlerEndpoint, err := CreateWorkerEndpoint(ctx, handlerClient, handlerEndpointName, cfg.HandlerNamespace, env.HandlerTaskQueue)
	if err != nil {
		env.Cleanup()
		return nil, fmt.Errorf("failed to create handler endpoint: %w", err)
	}

	// Step 9: Construct external endpoint URL pointing to handler server
	scheme := "http"
	if cfg.HandlerServer.TLS {
		scheme = "https"
	}
	externalEndpointURL := fmt.Sprintf("%s://%s%s", scheme, cfg.HandlerServer.HTTPAddr, handlerEndpoint.URLPrefix)
	fmt.Printf("Handler endpoint URL: %s\n", externalEndpointURL)

	// Step 10: Create external endpoint on caller namespace
	fmt.Printf("Creating external endpoint %s on caller namespace...\n", env.CallerEndpointName)
	if _, err := CreateExternalEndpoint(ctx, callerClient, env.CallerEndpointName, externalEndpointURL); err != nil {
		env.Cleanup()
		return nil, fmt.Errorf("failed to create caller endpoint: %w", err)
	}

	// Step 11: Start handler worker
	fmt.Printf("Starting handler worker for task queue %s...\n", env.HandlerTaskQueue)
	workerCmdStr := os.Getenv("HANDLER_WORKER_CMD")
	if workerCmdStr == "" {
		workerCmdStr = "go run -C ../worker-old ."
	}
	workerCmd := config.ParseCommand(workerCmdStr)

	handlerWorker, err := StartHandlerWorker(ctx, config.WorkerConfig{
		Command:    workerCmd,
		ServerAddr: cfg.HandlerServer.GRPCAddr,
		Namespace:  cfg.HandlerNamespace,
		TaskQueue:  env.HandlerTaskQueue,
	})
	if err != nil {
		env.Cleanup()
		return nil, fmt.Errorf("failed to start handler worker: %w", err)
	}
	env.HandlerWorker = handlerWorker
	fmt.Printf("Handler worker ready (logs: %s)\n", handlerWorker.LogPath())

	fmt.Println("Test environment setup complete!")
	return env, nil
}

// Cleanup stops all processes and releases resources in reverse order of setup
func (env *TestEnvironment) Cleanup() error {
	var errs []error

	// Stop handler worker
	if env.HandlerWorker != nil {
		fmt.Println("Stopping handler worker...")
		if err := env.HandlerWorker.Stop(); err != nil {
			errs = append(errs, fmt.Errorf("failed to stop handler worker: %w", err))
		}
	}

	// Close clients
	if env.CallerClient != nil {
		fmt.Println("Closing caller client...")
		env.CallerClient.Close()
	}
	if env.HandlerClient != nil {
		fmt.Println("Closing handler client...")
		env.HandlerClient.Close()
	}

	// Stop servers
	if env.HandlerServer != nil {
		fmt.Println("Stopping handler server...")
		if err := env.HandlerServer.Stop(); err != nil {
			errs = append(errs, fmt.Errorf("failed to stop handler server: %w", err))
		}
	}
	if env.CallerServer != nil {
		fmt.Println("Stopping caller server...")
		if err := env.CallerServer.Stop(); err != nil {
			errs = append(errs, fmt.Errorf("failed to stop caller server: %w", err))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
