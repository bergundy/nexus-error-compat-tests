package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// TestConfig holds the configuration for the test harness
type TestConfig struct {
	CallerServer     ServerConfig
	HandlerServer    ServerConfig
	CallerNamespace  string
	HandlerNamespace string
	TestTimeout      time.Duration
	NewHandlerWorker bool
}

// ServerConfig holds the configuration for a Temporal server instance
type ServerConfig struct {
	New          bool
	Command      []string      // Command to start the server (e.g., ["temporal", "server", "start-dev"])
	GRPCAddr     string        // gRPC address (e.g., "localhost:7233")
	HTTPAddr     string        // HTTP address (e.g., "localhost:7243")
	StartTimeout time.Duration // Timeout for server startup
	TLS          bool          // Whether to use TLS
	LogToFile    bool
}

// WorkerConfig holds the configuration for the handler worker process
type WorkerConfig struct {
	New        bool
	Command    []string // Command to start the worker
	ServerAddr string   // Server gRPC address
	Namespace  string   // Namespace name
	TaskQueue  string   // Task queue name
	LogToFile  bool
}

// DefaultTestConfig returns a default test configuration for local development
func DefaultTestConfig() TestConfig {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	wd = filepath.Dir(wd)

	defaultCallerServerCmd := "temporal server start-dev --port 7233 --http-port 7243 --ui-port 7244"
	newCallerServer := getBool("NEW_CALLER_SERVER", false)
	if newCallerServer {
		defaultCallerServerCmd = "go run -C ../../temporal ./cmd/server --config-file config/development-sqlite.yaml --allow-no-auth start"
	}

	defaultHandlerServerCmd := `temporal server start-dev --port 8233 --http-port 8243 --ui-port 8244 --dynamic-config-value component.callbacks.allowedAddresses=[{"Pattern":"*","AllowInsecure":true}]`
	newHandlerServer := getBool("NEW_HANDLER_SERVER", false)
	if newHandlerServer {
		configFile := filepath.Join(wd, "/development-sqlite-alt-port.yaml")
		defaultHandlerServerCmd = fmt.Sprintf("go run -C ../../temporal ./cmd/server --config-file %s --allow-no-auth start", configFile)
	}

	return TestConfig{
		CallerServer: ServerConfig{
			New:          newCallerServer,
			Command:      ParseCommand(getEnv("CALLER_SERVER_CMD", defaultCallerServerCmd)),
			GRPCAddr:     getEnv("CALLER_GRPC_ADDR", "localhost:7233"),
			HTTPAddr:     getEnv("CALLER_HTTP_ADDR", "localhost:7243"),
			StartTimeout: getDuration("CALLER_START_TIMEOUT", 30*time.Second),
			TLS:          getBool("CALLER_TLS", false),
			LogToFile:    getBool("CALLER_SERVER_LOG_TO_FILE", true),
		},
		HandlerServer: ServerConfig{
			New:          newHandlerServer,
			Command:      ParseCommand(getEnv("HANDLER_SERVER_CMD", defaultHandlerServerCmd)),
			GRPCAddr:     getEnv("HANDLER_GRPC_ADDR", "localhost:8233"),
			HTTPAddr:     getEnv("HANDLER_HTTP_ADDR", "localhost:8243"),
			StartTimeout: getDuration("HANDLER_START_TIMEOUT", 30*time.Second),
			TLS:          getBool("HANDLER_TLS", false),
			LogToFile:    getBool("HANDLER_SERVER_LOG_TO_FILE", true),
		},
		CallerNamespace:  getEnv("CALLER_NAMESPACE", "caller-ns"),
		HandlerNamespace: getEnv("HANDLER_NAMESPACE", "handler-ns"),
		TestTimeout:      getDuration("TEST_TIMEOUT", 60*time.Second),
		NewHandlerWorker: getBool("NEW_HANDLER_WORKER", false),
	}
}

// Helper functions for environment variable parsing

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}

func getBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}

func ParseCommand(cmd string) []string {
	// Simple command parsing - split by spaces but preserve quoted strings
	// For more complex commands, users can set the command programmatically
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return nil
	}
	return parts
}
