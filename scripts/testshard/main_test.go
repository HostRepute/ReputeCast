package main

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestSplitDealsEveryPackageOnce(t *testing.T) {
	pkgs := []string{"a", "b", "c", "d", "e", "f", "g"}
	times := map[string]float64{"a": 100, "b": 60, "c": 50, "d": 40, "e": 10}
	shards := split(pkgs, times, 3)
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
	times := map[string]float64{"a": 100, "b": 60, "c": 50, "d": 40, "e": 10}
	shards := split([]string{"e", "d", "c", "b", "a"}, times, 2)
	// a, then b and c on the other shard, d with a, e with b and c.
	if want := [][]string{{"a", "d"}, {"b", "c", "e"}}; !slices.EqualFunc(shards, want, slices.Equal) {
		t.Fatalf("shards %v, want %v", shards, want)
	}
}

func TestUnknownPackagesWeighTheMedian(t *testing.T) {
	// The median of 1, 2 and 90 is 2: the new package isn't put with the
	// 90-second one as if it took nothing.
	times := map[string]float64{"slow": 90, "x": 1, "y": 2}
	shards := split([]string{"new", "slow", "x", "y"}, times, 2)
	if want := [][]string{{"slow"}, {"new", "x", "y"}}; !slices.EqualFunc(shards, want, slices.Equal) {
		t.Fatalf("shards %v, want %v", shards, want)
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
	if err := run(strings.NewReader(events), &out, true, 0, 0, ""); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if got, want := lines[2:], []string{"61.0 m/b", "4.2 m/a", "0.0 m/c"}; !slices.Equal(got, want) {
		t.Fatalf("recorded %q, want %q after the header", got, want)
	}
	// What -record prints, the shards read back.
	times, err := readTimes(strings.NewReader(out.String()))
	if err != nil || times["m/b"] != 61 || len(times) != 3 {
		t.Fatalf("read back %v (%v)", times, err)
	}
}

func TestRunPrintsOneShard(t *testing.T) {
	durations := filepath.Join(t.TempDir(), "durations.txt")
	if err := os.WriteFile(durations, []byte("# header\n100 m/a\n60 m/b\n50 m/c\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := run(strings.NewReader("m/a\nm/b\nm/c\n"), &out, false, 2, 2, durations); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "m/b\nm/c\n" {
		t.Fatalf("shard 2 of 2: %q", got)
	}
	for _, bad := range [][2]int{{0, 2}, {3, 2}, {1, 0}} {
		if err := run(strings.NewReader("m/a\n"), &out, false, bad[0], bad[1], durations); err == nil {
			t.Fatalf("shard %d of %d was accepted", bad[0], bad[1])
		}
	}
}
