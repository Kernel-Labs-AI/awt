//go:build windows

package lock

import (
	"fmt"
	"os"
)

// tryAcquireLock attempts to acquire a lock using Windows-specific mechanisms
func tryAcquireLock(lockPath string) (*Lock, error) {
	// On Windows, we use O_EXCL for exclusive file creation
	// This is atomic and works well for file-based locking
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
	// No platform-specific unlock needed on Windows
	return nil
}
