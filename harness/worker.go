package harness

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/temporalio/nexus-error-compat-tests/config"
)

// WorkerProcess represents a running handler worker process
type WorkerProcess struct {
	cmd     *exec.Cmd
	config  config.WorkerConfig
	logFile *os.File
	mu      sync.Mutex
	stopped bool
}

// StartHandlerWorker starts the handler worker process and waits for it to be ready
func StartHandlerWorker(ctx context.Context, cfg config.WorkerConfig) (*WorkerProcess, error) {
	var logFile *os.File
	if cfg.LogToFile {
		var err error
		// Create log file for worker output
		logFile, err = os.CreateTemp("", "handler-worker-*.log")
		if err != nil {
			return nil, fmt.Errorf("failed to create log file: %w", err)
		}
	}

	// Prepare command with environment variables
	cmd := exec.CommandContext(ctx, cfg.Command[0], cfg.Command[1:]...)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("HANDLER_SERVER_ADDR=%s", cfg.ServerAddr),
		fmt.Sprintf("HANDLER_NAMESPACE=%s", cfg.Namespace),
		fmt.Sprintf("HANDLER_TASK_QUEUE=%s", cfg.TaskQueue),
	)

	// Create a pipe to capture stdout for readiness detection
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		if logFile != nil {
			logFile.Close()
		}
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if logFile != nil {
		cmd.Stderr = logFile
	} else {
		cmd.Stderr = os.Stderr
	}

	// Set process group to allow killing child processes
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// Start the worker process
	if err := cmd.Start(); err != nil {
		logFile.Close()
		return nil, fmt.Errorf("failed to start worker: %w", err)
	}

	wp := &WorkerProcess{
		cmd:     cmd,
		config:  cfg,
		logFile: logFile,
	}

	// Wait for worker to be ready by looking for "WORKER_READY" in stdout
	readyCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	outFile := os.Stdout
	if logFile != nil {
		outFile = logFile
	}
	readyCh := make(chan error, 1)
	go func() {
		var ready bool
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			line := scanner.Text()
			// Write to log file or stdout as well
			fmt.Fprintln(outFile, line)

			// Check for ready marker
			if line == "WORKER_READY" {
				if !ready {
					ready = true
					readyCh <- nil
				}
			}
		}
		if err := scanner.Err(); err != nil {
			readyCh <- fmt.Errorf("error reading worker output: %w", err)
		} else {
			readyCh <- fmt.Errorf("worker process ended without becoming ready")
		}
	}()

	select {
	case err := <-readyCh:
		if err != nil {
			wp.Stop()
			if logFile != nil {
				return nil, fmt.Errorf("worker failed to become ready: %w (logs: %s)", err, logFile.Name())
			}
			return nil, fmt.Errorf("worker failed to become ready: %w", err)
		}
		return wp, nil
	case <-readyCtx.Done():
		wp.Stop()
		if logFile != nil {
			return nil, fmt.Errorf("timeout waiting for worker to be ready (logs: %s)", logFile.Name())
		}
		return nil, fmt.Errorf("timeout waiting for worker to be ready")
	}
}

// Stop gracefully stops the worker process
func (wp *WorkerProcess) Stop() error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if wp.stopped {
		return nil
	}
	wp.stopped = true

	defer wp.logFile.Close()

	if wp.cmd == nil || wp.cmd.Process == nil {
		return nil
	}

	// Try graceful shutdown first (SIGTERM)
	if err := syscall.Kill(-wp.cmd.Process.Pid, syscall.SIGTERM); err != nil {
		// If SIGTERM fails, force kill
		wp.cmd.Process.Kill()
	}

	// Wait for process to exit with timeout
	done := make(chan error, 1)
	go func() {
		done <- wp.cmd.Wait()
	}()

	select {
	case err := <-done:
		// Process exited
		if err != nil && err.Error() != "signal: terminated" {
			return fmt.Errorf("worker process exited with error: %w", err)
		}
		return nil
	case <-time.After(5 * time.Second):
		// Force kill if graceful shutdown times out
		wp.cmd.Process.Kill()
		return fmt.Errorf("worker process did not stop gracefully, force killed")
	}
}

// LogPath returns the path to the worker's log file
func (wp *WorkerProcess) LogPath() string {
	if wp.logFile == nil {
		return ""
	}
	return wp.logFile.Name()
}

// IsRunning returns true if the worker process is still running
func (wp *WorkerProcess) IsRunning() bool {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if wp.stopped || wp.cmd == nil || wp.cmd.Process == nil {
		return false
	}

	// Check if process is still alive
	err := wp.cmd.Process.Signal(syscall.Signal(0))
	return err == nil
}
