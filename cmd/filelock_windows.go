//go:build windows

package cmd

// withExclusiveLock on Windows calls fn directly; advisory file locking
// via flock is not available on this platform.
func withExclusiveLock(_ string, fn func() error) error {
	return fn()
}
