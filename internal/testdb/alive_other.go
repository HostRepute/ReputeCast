//go:build !unix

package testdb

// processAlive can't tell on this platform, so clones are dated instead.
func processAlive(int) bool { return true }
