//go:build !windows

package cmd

import (
	"fmt"
	"os"
	"syscall"
)

// withExclusiveLock acquires an exclusive advisory lock on a sibling .lock file
// next to path, calls fn, then releases the lock. Multiple concurrent invocations
// of this tool on the same output file will serialize rather than corrupt data.
// When path is empty (stdout), fn is called directly without locking.
func withExclusiveLock(path string, fn func() error) error {
	if path == "" {
		return fn()
	}

	lockPath := path + ".lock"
	lf, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("open lock file %q: %w", lockPath, err)
	}
	defer func() {
		_ = syscall.Flock(int(lf.Fd()), syscall.LOCK_UN)
		_ = lf.Close()
		_ = os.Remove(lockPath)
	}()

	if err := syscall.Flock(int(lf.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("acquire lock on %q: %w", lockPath, err)
	}

	return fn()
}
