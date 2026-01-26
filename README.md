# Nexus Compatibility Test Harness

A Go test harness for testing Nexus operation compatibility between different versions of Temporal servers and SDKs.

## Architecture

The test harness supports a multi-process architecture for comprehensive compatibility testing:

- **Test Process**: Runs tests, caller worker, and client (single process)
- **Handler Worker**: Separate process hosting Nexus operation handlers
- **Caller Server**: Temporal server for the caller namespace
- **Handler Server**: Temporal server for the handler namespace

### Endpoint Configuration

- **Handler Side**: Worker-targeted endpoint pointing to handler worker's task queue
- **Caller Side**: External endpoint pointing to handler server's Nexus HTTP API

## Project Structure

```
nexus-error-compat-tests/
├── bin/
│   └── handler-worker      # Handler worker binary
├── config/
│   └── config.go          # Configuration structures
├── harness/
│   ├── server.go          # Server process management
│   ├── worker.go          # Worker process management
│   ├── setup.go           # Namespace and endpoint creation
│   └── coordinator.go     # Test environment orchestration
├── worker/
│   └── main.go            # Standalone handler worker
└── tests/
    ├── common.go          # Test utilities
    ├── sync_test.go       # Sync operation tests
    └── async_test.go      # Async operation tests
```

## Building

### Build Handler Worker

```bash
cd worker
go build -o ../bin/handler-worker .
```

## Running Tests

### Prerequisites

You need to have Temporal server binaries available. You can:

1. Use the Temporal CLI: `temporal server start-dev`
2. Build Temporal server from source
3. Download pre-built binaries

### Quick Start with Default Configuration

The default configuration starts two local dev servers:

```bash
# Build the handler worker first
cd worker && go build -o ../bin/handler-worker .

# Run tests
cd ../tests
go test -v .
```

### Configuration via Environment Variables

You can customize the test configuration using environment variables:

```bash
# Caller server configuration
export CALLER_SERVER_CMD="temporal server start-dev --port 7233 --http-port 7243"
export CALLER_GRPC_ADDR="localhost:7233"
export CALLER_HTTP_ADDR="localhost:7243"
export CALLER_NAMESPACE="caller-ns"

# Handler server configuration
export HANDLER_SERVER_CMD="temporal server start-dev --port 8233 --http-port 8243"
export HANDLER_GRPC_ADDR="localhost:8233"
export HANDLER_HTTP_ADDR="localhost:8243"
export HANDLER_NAMESPACE="handler-ns"

# Handler worker binary
export HANDLER_WORKER_BIN="./bin/handler-worker"

# Test timeout
export TEST_TIMEOUT="60s"

# Run tests
go test -v ./tests/
```

### Testing Different Version Combinations

The power of this test harness is the ability to test different version combinations:

#### Example 1: Old Caller SDK + New Handler SDK

```bash
# Build caller-side code with SDK v1.25.0
go mod edit -require=go.temporal.io/sdk@v1.25.0
go test -c -o bin/caller-test ./tests/

# Build handler worker with SDK v1.26.0
cd worker
go mod edit -require=go.temporal.io/sdk@v1.26.0
go build -o ../bin/handler-worker-v1.26 .

# Run tests
export HANDLER_WORKER_BIN="./bin/handler-worker-v1.26"
./bin/caller-test -test.v
```

#### Example 2: New SDK + Old Server

```bash
# Use latest SDK (already installed)
cd worker && go build -o ../bin/handler-worker .

# Configure to use old server binaries
export CALLER_SERVER_CMD="/path/to/temporal-v1.24/temporal server start-dev --port 7233"
export HANDLER_SERVER_CMD="/path/to/temporal-v1.24/temporal server start-dev --port 8233"

# Run tests
go test -v ./tests/
```

#### Example 3: Old Caller Server + New Handler Server

```bash
export CALLER_SERVER_CMD="/path/to/temporal-v1.24/temporal server start-dev --port 7233"
export HANDLER_SERVER_CMD="/path/to/temporal-v1.26/temporal server start-dev --port 8233"

go test -v ./tests/
```

## Test Scenarios

### Sync Operation Tests

- **TestSyncOperationSuccess**: Successful sync operation execution
- **TestSyncOperationFailure**: Tests various failure scenarios:
  - Operation failures
  - Application errors
  - Handler errors
  - Cancellation
- **TestSyncOperationEcho**: Echo functionality with arbitrary input

### Async Operation Tests

- **TestAsyncOperationSuccess**: Successful async operation execution
- **TestAsyncOperationFailure**: Tests workflow failures and application errors
- **TestAsyncOperationCancellation**: Tests cancellation scenarios:
  - Cancel after operation started
  - Cancel before operation sent
- **TestAsyncOperationEcho**: Echo functionality for async operations

## Handler Worker Operations

The handler worker (`worker/main.go`) implements two Nexus operations:

### Sync Operation: `sync-op`

Handles different test scenarios based on input:
- `"success"` - Returns success
- `"operation-failure"` - Returns operation failed error
- `"application-error"` - Returns application error
- `"handler-error"` - Returns handler error
- `"canceled"` - Returns canceled error
- Any other input - Echoes with prefix

### Async Operation: `async-op`

Backed by a workflow that handles:
- `"success"` - Returns success
- `"workflow-failure"` - Returns workflow failure
- `"application-error"` - Returns application error
- `"wait-for-cancel"` - Waits for cancellation
- Any other input - Echoes with "async" prefix

## Configuration Structure

The test harness uses a structured configuration:

```go
type TestConfig struct {
    CallerServer     ServerConfig
    HandlerServer    ServerConfig
    CallerNamespace  string
    HandlerNamespace string
    TestTimeout      time.Duration
}

type ServerConfig struct {
    Command      []string      // Command to start server
    GRPCAddr     string        // gRPC address
    HTTPAddr     string        // HTTP address
    StartTimeout time.Duration // Startup timeout
    TLS          bool          // Use TLS
}
```

## Troubleshooting

### Servers Not Starting

Check the server logs in `/tmp/temporal-server-*.log`. The test harness captures stdout/stderr to temporary files.

### Worker Not Ready

Check the worker logs in `/tmp/handler-worker-*.log`. The worker must print `WORKER_READY` to be considered ready.

### Port Conflicts

If ports are already in use, configure different ports via environment variables:

```bash
export CALLER_GRPC_ADDR="localhost:9233"
export CALLER_HTTP_ADDR="localhost:9243"
export HANDLER_GRPC_ADDR="localhost:10233"
export HANDLER_HTTP_ADDR="localhost:10243"
```

### Cleanup Issues

The test harness automatically cleans up resources. If processes are left running:

```bash
# Find and kill any stray temporal processes
pkill -f "temporal server"
```

## Development

### Adding New Tests

1. Create a new test function in `tests/sync_test.go` or `tests/async_test.go`
2. Use `NewTestContext(t, config.DefaultTestConfig())` to set up the environment
3. Define your caller workflow
4. Start a caller worker and register your workflow
5. Execute and verify the workflow

### Adding New Operation Scenarios

1. Edit `worker/main.go`
2. Add new cases to the switch statements in sync or async operation handlers
3. Rebuild the worker binary: `cd worker && go build -o ../bin/handler-worker .`
4. Add corresponding tests in `tests/`

## License

MIT
