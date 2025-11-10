package lock

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"
)

func TestLockBasic(t *testing.T) {
	// Create temp directory for testing
	tempDir, err := os.MkdirTemp("", "awt-lock-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	lm := NewLockManager(tempDir)
	ctx := context.Background()

	// Acquire global lock
	lock1, err := lm.AcquireGlobal(ctx)
	if err != nil {
		t.Fatalf("failed to acquire global lock: %v", err)
	}

	// Verify we can't acquire the same lock again (with short timeout)
	ctx2, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, err = lm.AcquireGlobal(ctx2)
	if err == nil {
		t.Error("expected error when acquiring already-held lock")
	}

	// Release the lock
	if err := lock1.Release(); err != nil {
		t.Fatalf("failed to release lock: %v", err)
	}

	// Now we should be able to acquire it
	lock2, err := lm.AcquireGlobal(ctx)
	if err != nil {
		t.Fatalf("failed to acquire lock after release: %v", err)
	}
	defer lock2.Release()
}

func TestLockConcurrency(t *testing.T) {
	// Create temp directory for testing
	tempDir, err := os.MkdirTemp("", "awt-lock-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	lm := NewLockManager(tempDir)

	const numGoroutines = 10
	var counter int
	var counterMu sync.Mutex
	var wg sync.WaitGroup

	// Launch multiple goroutines that try to increment a counter
	// The lock should serialize access
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			ctx := context.Background()
			lock, err := lm.AcquireLock(ctx, "test")
			if err != nil {
				t.Errorf("failed to acquire lock: %v", err)
				return
			}
			defer lock.Release()

			// Critical section - using mutex to protect counter access
			// The file lock is what we're testing, the mutex is just for race detector
			counterMu.Lock()
			current := counter
			counterMu.Unlock()

			time.Sleep(10 * time.Millisecond) // Simulate work

			counterMu.Lock()
			counter = current + 1
			counterMu.Unlock()
		}()
	}

	wg.Wait()

	counterMu.Lock()
	finalCount := counter
	counterMu.Unlock()

	if finalCount != numGoroutines {
		t.Errorf("counter = %d, expected %d (lock serialization failed)", finalCount, numGoroutines)
	}
}

func TestTaskLock(t *testing.T) {
	// Create temp directory for testing
	tempDir, err := os.MkdirTemp("", "awt-lock-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	lm := NewLockManager(tempDir)
	ctx := context.Background()

	// Acquire task-specific lock
	taskID := "20250110-120000-abc123"
	lock1, err := lm.AcquireTask(ctx, taskID)
	if err != nil {
		t.Fatalf("failed to acquire task lock: %v", err)
	}

	// Should be able to acquire a different task lock
	lock2, err := lm.AcquireTask(ctx, "different-task")
	if err != nil {
		t.Fatalf("failed to acquire different task lock: %v", err)
	}

	// But not the same task lock
	ctx3, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, err = lm.AcquireTask(ctx3, taskID)
	if err == nil {
		t.Error("expected error when acquiring already-held task lock")
	}

	lock1.Release()
	lock2.Release()
}

func TestLockTimeout(t *testing.T) {
	// Create temp directory for testing
	tempDir, err := os.MkdirTemp("", "awt-lock-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	lm := NewLockManager(tempDir)

	// Acquire lock
	ctx := context.Background()
	lock, err := lm.AcquireGlobal(ctx)
	if err != nil {
		t.Fatalf("failed to acquire lock: %v", err)
	}
	defer lock.Release()

	// Try to acquire with timeout
	start := time.Now()
	ctx2, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	_, err = lm.AcquireGlobal(ctx2)
	elapsed := time.Since(start)

	if err == nil {
		t.Error("expected timeout error")
	}

	// Verify timeout duration is approximately correct (with some tolerance)
	if elapsed < 400*time.Millisecond || elapsed > 700*time.Millisecond {
		t.Errorf("timeout took %v, expected ~500ms", elapsed)
	}
}

func TestLockCleanup(t *testing.T) {
	// Create temp directory for testing
	tempDir, err := os.MkdirTemp("", "awt-lock-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	lm := NewLockManager(tempDir)
	ctx := context.Background()

	// Acquire and release a lock
	lock, err := lm.AcquireGlobal(ctx)
	if err != nil {
		t.Fatalf("failed to acquire lock: %v", err)
	}
	lock.Release()

	// Run cleanup
	if err := lm.Cleanup(); err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}

	// Should be able to acquire lock again
	lock2, err := lm.AcquireGlobal(ctx)
	if err != nil {
		t.Fatalf("failed to acquire lock after cleanup: %v", err)
	}
	lock2.Release()
}

func TestLockDoubleRelease(t *testing.T) {
	// Create temp directory for testing
	tempDir, err := os.MkdirTemp("", "awt-lock-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	lm := NewLockManager(tempDir)
	ctx := context.Background()

	// Acquire lock
	lock, err := lm.AcquireGlobal(ctx)
	if err != nil {
		t.Fatalf("failed to acquire lock: %v", err)
	}

	// Release once
	if err := lock.Release(); err != nil {
		t.Fatalf("first release failed: %v", err)
	}

	// Release again - should not panic or error
	if err := lock.Release(); err != nil {
		t.Errorf("second release failed: %v", err)
	}
}
