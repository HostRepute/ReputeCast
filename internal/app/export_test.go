package app

import (
	"github.com/hostrepute/reputecast/internal/erasure"
	"github.com/hostrepute/reputecast/internal/ingest"
)

// ErasureSources is the registry's sources, for tests outside the package.
func ErasureSources() []erasure.Source { return erasureRegistry().Sources() }

// ErasureExempt is the coverage test's exemptions, for the erase test.
func ErasureExempt() map[string]string { return erasureExempt }

// NormalizerOf is the app's webhook normalizer, with its hooks.
func NormalizerOf(a *App) *ingest.Normalizer { return a.normalizer() }
