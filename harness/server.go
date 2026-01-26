package harness

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/temporalio/nexus-error-compat-tests/config"
)

// ServerProcess represents a running Temporal server process
type ServerProcess struct {
	cmd     *exec.Cmd
	config  config.ServerConfig
	mu      sync.Mutex
	stopped bool
	logFile *os.File
}

// StartServer starts a Temporal server process and waits for it to be ready
func StartServer(ctx context.Context, cfg config.ServerConfig, name string) (*ServerProcess, error) {
	if len(cfg.Command) == 0 {
		return nil, fmt.Errorf("server command is empty")
	}

	// Prepare command
	cmd := exec.CommandContext(ctx, cfg.Command[0], cfg.Command[1:]...)
	var logFile *os.File

	if cfg.LogToFile {
		var err error
		// Create log file for server output
		logFile, err = os.CreateTemp("", fmt.Sprintf("temporal-server-%s-*.log", name))
		if err != nil {
			return nil, fmt.Errorf("failed to create log file: %w", err)
		}
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	// Set process group to allow killing child processes
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// Start the server process
	if err := cmd.Start(); err != nil {
		if cfg.LogToFile {
			logFile.Close()
		}
		return nil, fmt.Errorf("failed to start server: %w", err)
	}

	sp := &ServerProcess{
		cmd:     cmd,
		config:  cfg,
		logFile: logFile,
	}

	// Wait for server to be ready
	startCtx, cancel := context.WithTimeout(ctx, cfg.StartTimeout)
	defer cancel()

	if err := sp.WaitReady(startCtx); err != nil {
		sp.Stop()
		if cfg.LogToFile {
			return nil, fmt.Errorf("server failed to become ready: %w (logs: %s)", err, logFile.Name())
		}
		return nil, fmt.Errorf("server failed to become ready: %w", err)
	}

	return sp, nil
}

// WaitReady waits for the server to be ready by polling the gRPC port
func (sp *ServerProcess) WaitReady(ctx context.Context) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for server to be ready: %w", ctx.Err())
		case <-ticker.C:
			// Try to connect to the gRPC port
			conn, err := net.DialTimeout("tcp", sp.config.GRPCAddr, time.Second)
			if err == nil {
				conn.Close()
				// Give it a bit more time to fully initialize
				time.Sleep(500 * time.Millisecond)
				return nil
			}
		}
	}
}

// Stop gracefully stops the server process
func (sp *ServerProcess) Stop() error {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	if sp.stopped {
		return nil
	}
	sp.stopped = true

	if sp.logFile != nil {
		defer sp.logFile.Close()
	}

	if sp.cmd == nil || sp.cmd.Process == nil {
		return nil
	}

	// Try graceful shutdown first (SIGTERM)
	if err := syscall.Kill(-sp.cmd.Process.Pid, syscall.SIGTERM); err != nil {
		// If SIGTERM fails, force kill
		sp.cmd.Process.Kill()
	}

	// Wait for process to exit with timeout
	done := make(chan error, 1)
	go func() {
		done <- sp.cmd.Wait()
	}()

	select {
	case err := <-done:
		// Process exited
		if err != nil && err.Error() != "signal: terminated" {
			return fmt.Errorf("server process exited with error: %w", err)
		}
		return nil
	case <-time.After(10 * time.Second):
		// Force kill if graceful shutdown times out
		sp.cmd.Process.Kill()
		return fmt.Errorf("server process did not stop gracefully, force killed")
	}
}

// LogPath returns the path to the server's log file
func (sp *ServerProcess) LogPath() string {
	if sp.logFile == nil {
		return ""
	}
	return sp.logFile.Name()
}

// IsRunning returns true if the server process is still running
func (sp *ServerProcess) IsRunning() bool {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	if sp.stopped || sp.cmd == nil || sp.cmd.Process == nil {
		return false
	}

	// Check if process is still alive
	err := sp.cmd.Process.Signal(syscall.Signal(0))
	return err == nil
}
