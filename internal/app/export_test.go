package app

import "github.com/hostrepute/reputecast/internal/erasure"

// ErasureSources is the registry's sources, for tests outside the package.
func ErasureSources() []erasure.Source { return erasureRegistry().Sources() }

// ErasureExempt is the coverage test's exemptions, for the erase test.
func ErasureExempt() map[string]string { return erasureExempt }
