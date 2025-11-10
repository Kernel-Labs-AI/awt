package lock

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/unix"
)

const (
	// DefaultTimeout is the default lock acquisition timeout
	DefaultTimeout = 30 * time.Second

	// RetryInterval is how long to wait between lock attempts
	RetryInterval = 100 * time.Millisecond
)

// Lock represents a file-based lock
type Lock struct {
	path string
	file *os.File
}

// LockManager manages locks for the AWT system
type LockManager struct {
	locksDir string
}

// NewLockManager creates a new lock manager
func NewLockManager(gitCommonDir string) *LockManager {
	return &LockManager{
		locksDir: filepath.Join(gitCommonDir, "awt", "locks"),
	}
}

// AcquireGlobal acquires the global lock with the default timeout
func (lm *LockManager) AcquireGlobal(ctx context.Context) (*Lock, error) {
	return lm.AcquireLock(ctx, "global")
}

// AcquireTask acquires a task-specific lock with the default timeout
func (lm *LockManager) AcquireTask(ctx context.Context, taskID string) (*Lock, error) {
	return lm.AcquireLock(ctx, taskID)
}

// AcquireLock acquires a lock with the given name
func (lm *LockManager) AcquireLock(ctx context.Context, name string) (*Lock, error) {
	// Ensure locks directory exists
	if err := os.MkdirAll(lm.locksDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create locks directory: %w", err)
	}

	lockPath := filepath.Join(lm.locksDir, name+".lock")

	// Try to acquire lock with timeout
	deadline, hasDeadline := ctx.Deadline()
	if !hasDeadline {
		// No deadline set, use default timeout
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultTimeout)
		defer cancel()
		deadline, _ = ctx.Deadline()
	}

	startTime := time.Now()
	for {
		// Try to acquire the lock
		lock, err := tryAcquireLock(lockPath)
		if err == nil {
			return lock, nil
		}

		// Check if context is done (timeout or cancellation)
		select {
		case <-ctx.Done():
			elapsed := time.Since(startTime)
			return nil, fmt.Errorf("failed to acquire lock %s after %v: %w", name, elapsed, ctx.Err())
		default:
		}

		// Check if we've exceeded the deadline
		if time.Now().After(deadline) {
			elapsed := time.Since(startTime)
			return nil, fmt.Errorf("lock acquisition timeout for %s after %v", name, elapsed)
		}

		// Wait before retrying
		time.Sleep(RetryInterval)
	}
}

// tryAcquireLock attempts to acquire a lock using flock with fallback
func tryAcquireLock(lockPath string) (*Lock, error) {
	// Open or create the lock file
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open lock file: %w", err)
	}

	// Try flock first (POSIX systems)
	err = unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if err == nil {
		// Successfully acquired flock
		return &Lock{
			path: lockPath,
			file: file,
		}, nil
	}

	// Check if flock failed because lock is held
	if err == unix.EWOULDBLOCK || err == unix.EAGAIN {
		file.Close()
		return nil, fmt.Errorf("lock is held")
	}

	// flock not supported, try O_EXCL fallback for network filesystems
	file.Close()

	// Try atomic create with O_EXCL
	exclusivePath := lockPath + ".exclusive"
	exclusiveFile, err := os.OpenFile(exclusivePath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0644)
	if err != nil {
		if os.IsExist(err) {
			// Lock is held by another process
			return nil, fmt.Errorf("lock is held")
		}
		return nil, fmt.Errorf("failed to create exclusive lock: %w", err)
	}

	// Write PID to lock file for debugging
	pid := os.Getpid()
	fmt.Fprintf(exclusiveFile, "%d\n", pid)

	return &Lock{
		path: exclusivePath,
		file: exclusiveFile,
	}, nil
}

// Release releases the lock
func (l *Lock) Release() error {
	if l.file == nil {
		return nil
	}

	// Try to unlock with flock first
	_ = unix.Flock(int(l.file.Fd()), unix.LOCK_UN)

	// Close the file
	if err := l.file.Close(); err != nil {
		return fmt.Errorf("failed to close lock file: %w", err)
	}

	// If this is an exclusive lock, remove the file
	if filepath.Ext(l.path) == ".exclusive" {
		if err := os.Remove(l.path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove exclusive lock file: %w", err)
		}
	}

	l.file = nil
	return nil
}

// Cleanup removes stale lock files
// This should be called during prune operations
func (lm *LockManager) Cleanup() error {
	entries, err := os.ReadDir(lm.locksDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read locks directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		lockPath := filepath.Join(lm.locksDir, entry.Name())

		// Try to acquire the lock
		lock, err := tryAcquireLock(lockPath)
		if err == nil {
			// Lock was available, so it was stale - release it
			lock.Release()
			// Remove the lock file if it's not in use
			if filepath.Ext(lockPath) == ".lock" {
				os.Remove(lockPath)
			}
		}
		// If we can't acquire it, it's in use - leave it alone
	}

	return nil
}
