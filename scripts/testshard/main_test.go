package main

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

func pkgTimes(m map[string]float64) times {
	return times{pkgs: m, tests: map[string]map[string]float64{}}
}

// shardPkgs is each shard's package column.
func shardPkgs(shards [][]line) [][]string {
	out := make([][]string, len(shards))
	for k, ls := range shards {
		out[k] = []string{}
		for _, l := range ls {
			out[k] = append(out[k], l.pkg)
		}
	}
	return out
}

func TestSplitDealsEveryPackageOnce(t *testing.T) {
	pkgs := []string{"a", "b", "c", "d", "e", "f", "g"}
	shards := shardPkgs(split(pkgs, pkgTimes(map[string]float64{"a": 100, "b": 60, "c": 50, "d": 40, "e": 10}), 3))
	var all []string
	for _, s := range shards {
		all = append(all, s...)
	}
	slices.Sort(all)
	if !slices.Equal(all, pkgs) {
		t.Fatalf("shards %v hold %v, want each of %v once", shards, all, pkgs)
	}
}

func TestSplitBalancesHeaviestFirst(t *testing.T) {
	shards := shardPkgs(split([]string{"e", "d", "c", "b", "a"}, pkgTimes(map[string]float64{"a": 100, "b": 60, "c": 50, "d": 40, "e": 10}), 2))
	// a, then b and c on the other shard, d with a, e with b and c.
	if want := [][]string{{"a", "d"}, {"b", "c", "e"}}; !slices.EqualFunc(shards, want, slices.Equal) {
		t.Fatalf("shards %v, want %v", shards, want)
	}
}

func TestUnknownPackagesWeighTheMedian(t *testing.T) {
	// The median of 1, 2 and 90 is 2: the new package isn't put with the
	// 90-second one as if it took nothing.
	shards := shardPkgs(split([]string{"new", "slow", "x", "y"}, pkgTimes(map[string]float64{"slow": 90, "x": 1, "y": 2}), 2))
	if want := [][]string{{"slow"}, {"new", "x", "y"}}; !slices.EqualFunc(shards, want, slices.Equal) {
		t.Fatalf("shards %v, want %v", shards, want)
	}
}

// ran is which of tests a shard's line for a split package runs, as
// -test.run and -test.skip select top-level tests.
func ran(t *testing.T, l line, tests []string) []string {
	t.Helper()
	if l.mode == "" {
		return tests
	}
	re := regexp.MustCompile(l.re)
	var out []string
	for _, name := range tests {
		if re.MatchString(name) == (l.mode == "run") {
			out = append(out, name)
		}
	}
	return out
}

func TestSplitPackageRunsEachTestOnce(t *testing.T) {
	tm := times{
		pkgs:  map[string]float64{"m/x": 80, "m/y": 70},
		tests: map[string]map[string]float64{"m/big": {"TestA": 100, "TestB": 60, "TestC": 50, "TestD": 5}},
	}
	shards := split([]string{"m/big", "m/x", "m/y"}, tm, 3)
	// TestNew isn't timed yet: it has to run too, once.
	tests := []string{"TestA", "TestB", "TestC", "TestD", "TestNew"}
	runs := map[string]int{}
	modes := map[string]int{}
	for _, ls := range shards {
		for _, l := range ls {
			if l.pkg != "m/big" {
				continue
			}
			modes[l.mode]++
			for _, name := range ran(t, l, tests) {
				runs[name]++
			}
		}
	}
	for _, name := range tests {
		if runs[name] != 1 {
			t.Fatalf("%s runs %d times in %v", name, runs[name], shards)
		}
	}
	if modes["skip"] != 1 || modes["run"] < 1 || modes[""] != 0 {
		t.Fatalf("m/big's lines %v: want one skip and the rest run", shards)
	}
}

func TestSplitPackageInOneShardRunsWhole(t *testing.T) {
	tm := times{pkgs: map[string]float64{}, tests: map[string]map[string]float64{"m/big": {"TestA": 1, "TestB": 1}}}
	shards := split([]string{"m/big"}, tm, 1)
	if len(shards[0]) != 1 || shards[0][0] != (line{pkg: "m/big"}) {
		t.Fatalf("one shard: %v, want the whole package", shards)
	}
}

func TestRecordKeepsPackageTimes(t *testing.T) {
	events := `{"Action":"run","Package":"m/a","Test":"TestX"}
{"Action":"pass","Package":"m/a","Test":"TestX","Elapsed":3}
{"Action":"pass","Package":"m/a","Elapsed":4.25}
{"Action":"fail","Package":"m/b","Elapsed":61}
{"Action":"skip","Package":"m/c","Elapsed":0}
`
	var out bytes.Buffer
	if err := run(strings.NewReader(events), &out, options{record: true}); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if got, want := lines[3:], []string{"61.0 m/b", "4.2 m/a", "0.0 m/c"}; !slices.Equal(got, want) {
		t.Fatalf("recorded %q, want %q after the header", got, want)
	}
	// What -record prints, the shards read back.
	tm, err := readTimes(strings.NewReader(out.String()))
	if err != nil || tm.pkgs["m/b"] != 61 || len(tm.pkgs) != 3 {
		t.Fatalf("read back %v (%v)", tm, err)
	}
}

func TestRecordTimesSplitPackagesByTest(t *testing.T) {
	events := `{"Action":"pass","Package":"m/big","Test":"TestA","Elapsed":30}
{"Action":"pass","Package":"m/big","Test":"TestA/sub","Elapsed":29}
{"Action":"pass","Package":"m/big","Test":"TestB","Elapsed":2}
{"Action":"pass","Package":"m/big","Elapsed":33}
{"Action":"pass","Package":"m/a","Elapsed":4}
`
	var out bytes.Buffer
	if err := run(strings.NewReader(events), &out, options{record: true, split: []string{"m/big"}}); err != nil {
		t.Fatal(err)
	}
	tm, err := readTimes(strings.NewReader(out.String()))
	if err != nil {
		t.Fatal(err)
	}
	if _, whole := tm.pkgs["m/big"]; whole || len(tm.tests["m/big"]) != 2 || tm.tests["m/big"]["TestA"] != 30 || tm.pkgs["m/a"] != 4 {
		t.Fatalf("recorded %q", out.String())
	}
	if err := run(strings.NewReader(events), &out, options{record: true, split: []string{"m/gone"}}); err == nil {
		t.Fatal("a -split package without test results was accepted")
	}
}

func TestRunPrintsOneShard(t *testing.T) {
	durations := filepath.Join(t.TempDir(), "durations.txt")
	if err := os.WriteFile(durations, []byte("# header\n100 m/a\n60 m/b\n50 m/c\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := run(strings.NewReader("m/a\nm/b\nm/c\n"), &out, options{shard: 2, of: 2, durations: durations}); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "m/b\nm/c\n" {
		t.Fatalf("shard 2 of 2: %q", got)
	}
	for _, bad := range [][2]int{{0, 2}, {3, 2}, {1, 0}} {
		if err := run(strings.NewReader("m/a\n"), &out, options{shard: bad[0], of: bad[1], durations: durations}); err == nil {
			t.Fatalf("shard %d of %d was accepted", bad[0], bad[1])
		}
	}
}
