// Command testshard splits the module's Go tests into CI shards that take
// about as long as each other.
//
//	go list ./... | go run ./scripts/testshard -shard 2 -of 3
//
// prints what shard 2 of 3 tests, one package per line. A package weighs
// what its tests took in .github/go-test-durations.txt ("seconds
// import/path" lines); a package the file doesn't list weighs the median.
// The file can also time a package's tests one by one ("seconds
// import/path TestName" lines). Such a package is split: its tests are
// dealt to the shards, and each shard that gets some of them prints
// "import/path run regexp" naming them, which the package's TestMain
// applies (internal/testsplit). The shard with the least of the package's
// time prints "import/path skip regexp" with the other shards' tests
// instead, so a test the file doesn't time yet runs there. Tests and
// packages are dealt heaviest first, so everything runs in exactly one
// shard whatever the file says.
//
//	go test -race -count=1 -json ./... | go run ./scripts/testshard -record -split import/path
//
// prints the file's lines from a run of the whole suite, timing the -split
// packages' tests one by one (make test-durations).
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

func main() {
	var o options
	flag.IntVar(&o.shard, "shard", 0, "the shard to print, from 1")
	flag.IntVar(&o.of, "of", 0, "how many shards")
	flag.StringVar(&o.durations, "durations", ".github/go-test-durations.txt", "the test times")
	flag.BoolVar(&o.record, "record", false, "read go test -json and print the test times")
	split := flag.String("split", "", "with -record, the packages to time test by test (comma-separated import paths)")
	flag.Parse()
	if *split != "" {
		o.split = strings.Split(*split, ",")
	}
	if err := run(os.Stdin, os.Stdout, o); err != nil {
		fmt.Fprintln(os.Stderr, "testshard:", err)
		os.Exit(1)
	}
}

type options struct {
	shard, of int
	durations string
	record    bool
	split     []string
}

func run(in io.Reader, out io.Writer, o options) error {
	if o.record {
		t, err := recordTimes(in, o.split)
		if err != nil {
			return err
		}
		return writeTimes(out, t)
	}
	if o.of < 1 || o.shard < 1 || o.shard > o.of {
		return fmt.Errorf("want -shard between 1 and -of, got %d of %d", o.shard, o.of)
	}
	pkgs, err := readLines(in)
	if err != nil {
		return err
	}
	if len(pkgs) == 0 {
		return errors.New("no packages on stdin")
	}
	f, err := os.Open(o.durations) //nolint:gosec // a path the workflow passes
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	t, err := readTimes(f)
	if err != nil {
		return fmt.Errorf("%s: %w", o.durations, err)
	}
	for _, l := range split(pkgs, t, o.of)[o.shard-1] {
		fmt.Fprintln(out, l)
	}
	return nil
}

// times is what the durations file says: seconds per package, and per
// test for the packages it times test by test.
type times struct {
	pkgs  map[string]float64
	tests map[string]map[string]float64
}

// line is what a shard tests of a package: all of it, or the tests a
// regexp runs or skips.
type line struct {
	pkg, mode, re string
}

func (l line) String() string {
	if l.mode == "" {
		return l.pkg
	}
	return l.pkg + " " + l.mode + " " + l.re
}

// split deals the tests of the packages timed test by test first: a
// package's tests run one after another, so each goes, heaviest first, to
// the shard with the least of that package's time so far (then the least
// time, then the first). The other packages follow, heaviest first, each to
// the shard with the least time so far (the first on a tie). It returns
// each shard's lines in package order.
func split(pkgs []string, t times, n int) [][]line {
	fallback := median(t.pkgs)
	type item struct {
		pkg, test string
		w         float64
	}
	var tests, whole []item
	for _, p := range pkgs {
		if ts := t.tests[p]; len(ts) > 0 {
			for name, w := range ts {
				tests = append(tests, item{p, name, w})
			}
			continue
		}
		w, ok := t.pkgs[p]
		if !ok {
			w = fallback
		}
		whole = append(whole, item{p, "", w})
	}
	for _, items := range [][]item{tests, whole} {
		sort.Slice(items, func(i, j int) bool {
			a, b := items[i], items[j]
			if a.w != b.w {
				return a.w > b.w
			}
			if a.pkg != b.pkg {
				return a.pkg < b.pkg
			}
			return a.test < b.test
		})
	}
	totals := make([]float64, n)
	own := map[string][]float64{}    // split package → its time per shard
	parts := map[string][][]string{} // split package → its tests per shard
	for _, it := range tests {
		if own[it.pkg] == nil {
			own[it.pkg] = make([]float64, n)
			parts[it.pkg] = make([][]string, n)
		}
		o := own[it.pkg]
		k := 0
		for i := 1; i < n; i++ {
			if o[i] < o[k] || (o[i] == o[k] && totals[i] < totals[k]) {
				k = i
			}
		}
		o[k] += it.w
		totals[k] += it.w
		parts[it.pkg][k] = append(parts[it.pkg][k], it.test)
	}
	out := make([][]line, n)
	for _, it := range whole {
		k := 0
		for i := 1; i < n; i++ {
			if totals[i] < totals[k] {
				k = i
			}
		}
		totals[k] += it.w
		out[k] = append(out[k], line{pkg: it.pkg})
	}
	for p, byShard := range parts {
		for k, l := range partLines(p, byShard, own[p]) {
			if l.pkg != "" {
				out[k] = append(out[k], l)
			}
		}
	}
	for _, ls := range out {
		sort.Slice(ls, func(i, j int) bool { return ls[i].pkg < ls[j].pkg })
	}
	return out
}

// partLines says what each shard runs of a split package: all of it when
// one shard got all its tests; otherwise the shard with the least of its
// time skips the others' tests, and the others run theirs.
func partLines(pkg string, byShard [][]string, own []float64) []line {
	out := make([]line, len(byShard))
	home, used := -1, 0
	for k, tests := range byShard {
		if len(tests) == 0 {
			continue
		}
		used++
		if home < 0 || own[k] < own[home] {
			home = k
		}
	}
	if used == 1 {
		out[home] = line{pkg: pkg}
		return out
	}
	var others []string
	for k, tests := range byShard {
		if k == home || len(tests) == 0 {
			continue
		}
		out[k] = line{pkg, "run", names(tests)}
		others = append(others, tests...)
	}
	out[home] = line{pkg, "skip", names(others)}
	return out
}

// names is a -test.run regexp matching exactly these top-level tests.
func names(tests []string) string {
	q := make([]string, len(tests))
	for i, name := range tests {
		q[i] = regexp.QuoteMeta(name)
	}
	sort.Strings(q)
	return "^(" + strings.Join(q, "|") + ")$"
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

// readTimes reads "seconds import/path" and "seconds import/path TestName"
// lines; blank lines and lines starting with # are skipped.
func readTimes(r io.Reader) (times, error) {
	t := times{pkgs: map[string]float64{}, tests: map[string]map[string]float64{}}
	lines, err := readLines(r)
	if err != nil {
		return t, err
	}
	for _, l := range lines {
		if strings.HasPrefix(l, "#") {
			continue
		}
		f := strings.Fields(l)
		var secs float64
		if len(f) == 2 || len(f) == 3 {
			secs, err = strconv.ParseFloat(f[0], 64)
		}
		if (len(f) != 2 && len(f) != 3) || err != nil || secs < 0 {
			return t, fmt.Errorf("want \"seconds import/path [TestName]\", got %q", l)
		}
		if len(f) == 2 {
			t.pkgs[f[1]] = secs
			continue
		}
		if t.tests[f[1]] == nil {
			t.tests[f[1]] = map[string]float64{}
		}
		t.tests[f[1]][f[2]] = secs
	}
	return t, nil
}

// recordTimes reads go test -json events and keeps each package's time
// (the Elapsed of its pass, fail or skip event) and, for the split
// packages, each top-level test's.
func recordTimes(r io.Reader, split []string) (times, error) {
	t := times{pkgs: map[string]float64{}, tests: map[string]map[string]float64{}}
	splitting := map[string]bool{}
	for _, p := range split {
		splitting[p] = true
	}
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
			return t, err
		}
		if ev.Package == "" || (ev.Action != "pass" && ev.Action != "fail" && ev.Action != "skip") {
			continue
		}
		switch {
		case ev.Test == "" && !splitting[ev.Package]:
			t.pkgs[ev.Package] = ev.Elapsed
		case ev.Test != "" && !strings.Contains(ev.Test, "/") && splitting[ev.Package]:
			if t.tests[ev.Package] == nil {
				t.tests[ev.Package] = map[string]float64{}
			}
			t.tests[ev.Package][ev.Test] = ev.Elapsed
		}
	}
	if len(t.pkgs) == 0 && len(t.tests) == 0 {
		return t, errors.New("no package results on stdin")
	}
	for _, p := range split {
		if len(t.tests[p]) == 0 {
			return t, fmt.Errorf("no test results for %s, which -split names", p)
		}
	}
	return t, nil
}

// writeTimes prints the times slowest first, with a header saying how to
// refresh them.
func writeTimes(w io.Writer, t times) error {
	type entry struct {
		secs      float64
		pkg, test string
	}
	var es []entry
	for p, s := range t.pkgs {
		es = append(es, entry{s, p, ""})
	}
	for p, tests := range t.tests {
		for name, s := range tests {
			es = append(es, entry{s, p, name})
		}
	}
	sort.Slice(es, func(i, j int) bool {
		a, b := es[i], es[j]
		if a.secs != b.secs {
			return a.secs > b.secs
		}
		if a.pkg != b.pkg {
			return a.pkg < b.pkg
		}
		return a.test < b.test
	})
	bw := bufio.NewWriter(w)
	fmt.Fprintln(bw, "# Seconds each package's tests took under go test -race, slowest first; a")
	fmt.Fprintln(bw, "# package split across shards has a line per test. scripts/testshard splits")
	fmt.Fprintln(bw, "# CI's Go tests by them; make test-durations refreshes them.")
	for _, e := range es {
		if e.test == "" {
			fmt.Fprintf(bw, "%.1f %s\n", e.secs, e.pkg)
		} else {
			fmt.Fprintf(bw, "%.1f %s %s\n", e.secs, e.pkg, e.test)
		}
	}
	return bw.Flush()
}
