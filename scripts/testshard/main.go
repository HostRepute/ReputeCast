// Command testshard splits the module's Go packages into CI shards that
// take about as long as each other.
//
//	go list ./... | go run ./scripts/testshard -shard 2 -of 4
//
// prints the packages of shard 2 of 4, one per line. A package weighs what
// its tests took in .github/go-test-durations.txt ("seconds import/path"
// lines); a package the file doesn't list weighs the median. Packages are
// dealt heaviest first to the lightest shard, so every package is in
// exactly one shard whatever the file says.
//
//	go test -race -count=1 -json ./... | go run ./scripts/testshard -record
//
// prints the file's lines from a run of the whole suite (make
// test-durations).
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
)

func main() {
	shard := flag.Int("shard", 0, "the shard to print, from 1")
	of := flag.Int("of", 0, "how many shards")
	durations := flag.String("durations", ".github/go-test-durations.txt", "the packages' test times")
	record := flag.Bool("record", false, "read go test -json and print the packages' test times")
	flag.Parse()
	if err := run(os.Stdin, os.Stdout, *record, *shard, *of, *durations); err != nil {
		fmt.Fprintln(os.Stderr, "testshard:", err)
		os.Exit(1)
	}
}

func run(in io.Reader, out io.Writer, record bool, shard, of int, durations string) error {
	if record {
		times, err := recordTimes(in)
		if err != nil {
			return err
		}
		return writeTimes(out, times)
	}
	if of < 1 || shard < 1 || shard > of {
		return fmt.Errorf("want -shard between 1 and -of, got %d of %d", shard, of)
	}
	pkgs, err := readLines(in)
	if err != nil {
		return err
	}
	if len(pkgs) == 0 {
		return errors.New("no packages on stdin")
	}
	f, err := os.Open(durations) //nolint:gosec // a path the workflow passes
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	times, err := readTimes(f)
	if err != nil {
		return fmt.Errorf("%s: %w", durations, err)
	}
	for _, p := range split(pkgs, times, of)[shard-1] {
		fmt.Fprintln(out, p)
	}
	return nil
}

// split deals the packages heaviest first, each to the shard with the
// least time so far (the first on a tie), and returns each shard's
// packages in name order.
func split(pkgs []string, times map[string]float64, n int) [][]string {
	fallback := median(times)
	weight := func(p string) float64 {
		if t, ok := times[p]; ok {
			return t
		}
		return fallback
	}
	sorted := append([]string(nil), pkgs...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if wi, wj := weight(sorted[i]), weight(sorted[j]); wi != wj {
			return wi > wj
		}
		return sorted[i] < sorted[j]
	})
	shards := make([][]string, n)
	totals := make([]float64, n)
	for _, p := range sorted {
		k := 0
		for i := 1; i < n; i++ {
			if totals[i] < totals[k] {
				k = i
			}
		}
		shards[k] = append(shards[k], p)
		totals[k] += weight(p)
	}
	for _, s := range shards {
		sort.Strings(s)
	}
	return shards
}

func median(times map[string]float64) float64 {
	if len(times) == 0 {
		return 1
	}
	v := make([]float64, 0, len(times))
	for _, t := range times {
		v = append(v, t)
	}
	sort.Float64s(v)
	return v[len(v)/2]
}

func readLines(r io.Reader) ([]string, error) {
	var out []string
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		if l := strings.TrimSpace(sc.Text()); l != "" {
			out = append(out, l)
		}
	}
	return out, sc.Err()
}

// readTimes reads "seconds import/path" lines; blank lines and lines
// starting with # are skipped.
func readTimes(r io.Reader) (map[string]float64, error) {
	lines, err := readLines(r)
	if err != nil {
		return nil, err
	}
	times := map[string]float64{}
	for _, l := range lines {
		if strings.HasPrefix(l, "#") {
			continue
		}
		secs, pkg, ok := strings.Cut(l, " ")
		t, err := strconv.ParseFloat(secs, 64)
		if !ok || err != nil || t < 0 {
			return nil, fmt.Errorf("want \"seconds import/path\", got %q", l)
		}
		times[strings.TrimSpace(pkg)] = t
	}
	return times, nil
}

// recordTimes reads go test -json events and keeps each package's time:
// the Elapsed of its pass, fail or skip event.
func recordTimes(r io.Reader) (map[string]float64, error) {
	times := map[string]float64{}
	dec := json.NewDecoder(r)
	for {
		var ev struct {
			Action  string
			Package string
			Test    string
			Elapsed float64
		}
		if err := dec.Decode(&ev); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, err
		}
		if ev.Test != "" || ev.Package == "" {
			continue
		}
		switch ev.Action {
		case "pass", "fail", "skip":
			times[ev.Package] = ev.Elapsed
		}
	}
	if len(times) == 0 {
		return nil, errors.New("no package results on stdin")
	}
	return times, nil
}

// writeTimes prints the times slowest first, with a header saying how to
// refresh them.
func writeTimes(w io.Writer, times map[string]float64) error {
	pkgs := make([]string, 0, len(times))
	for p := range times {
		pkgs = append(pkgs, p)
	}
	sort.Slice(pkgs, func(i, j int) bool {
		if times[pkgs[i]] != times[pkgs[j]] {
			return times[pkgs[i]] > times[pkgs[j]]
		}
		return pkgs[i] < pkgs[j]
	})
	bw := bufio.NewWriter(w)
	fmt.Fprintln(bw, "# Seconds each package's tests took under go test -race, slowest first.")
	fmt.Fprintln(bw, "# scripts/testshard splits CI's Go tests by them; make test-durations refreshes them.")
	for _, p := range pkgs {
		fmt.Fprintf(bw, "%.1f %s\n", times[p], p)
	}
	return bw.Flush()
}
