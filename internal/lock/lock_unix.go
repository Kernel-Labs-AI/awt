//go:build unix || linux || darwin

package lock

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

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
		_ = file.Close()
		return nil, fmt.Errorf("lock is held")
	}

	// flock not supported, try O_EXCL fallback for network filesystems
	_ = file.Close()

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
	_, _ = fmt.Fprintf(exclusiveFile, "%d\n", pid)

	return &Lock{
		path: exclusivePath,
		file: exclusiveFile,
	}, nil
}

// releaseLock releases the platform-specific lock
func releaseLock(l *Lock) error {
	// Try to unlock with flock first
	_ = unix.Flock(int(l.file.Fd()), unix.LOCK_UN)
	return nil
}
