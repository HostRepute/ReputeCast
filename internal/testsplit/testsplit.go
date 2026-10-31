// Package testsplit runs the share of a package's tests that a CI shard was
// given. scripts/testshard deals the tests of the slowest packages to
// several shards and tells each, in REPUTECAST_TEST_RUN, which of a split
// package's tests it runs: "import/path run regexp" or "import/path skip
// regexp" lines. A package CI splits calls Main from its TestMain; without
// a line for it, it runs all its tests.
package testsplit

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"
)

// Env holds the shares.
const Env = "REPUTECAST_TEST_RUN"

// Main runs the tests of the package at import path pkg that its line in
// Env selects, or all of them.
func Main(m *testing.M, pkg string) {
	flag.Parse()
	if err := apply(os.Getenv(Env), pkg, flag.Set); err != nil {
		fmt.Fprintf(os.Stderr, "testsplit: %v\n", err)
		os.Exit(2)
	}
	os.Exit(m.Run())
}

// apply sets -test.run or -test.skip from pkg's line in env.
func apply(env, pkg string, set func(name, value string) error) error {
	for _, line := range strings.Split(env, "\n") {
		f := strings.Fields(line)
		if len(f) == 0 || f[0] != pkg {
			continue
		}
		if len(f) != 3 || (f[1] != "run" && f[1] != "skip") {
			return fmt.Errorf("%s: want \"import/path run|skip regexp\", got %q", Env, line)
		}
		return set("test."+f[1], f[2])
	}
	return nil
}
