package testdb

import (
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestJudgeClone(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	ts := func(d time.Duration) string { return strconv.FormatInt(now.Add(-d).Unix(), 10) }
	dead, live := 4242, os.Getpid()
	alive := func(pid int) bool { return pid != dead }
	cases := []struct {
		name string
		want verdict
	}{
		// This host: the PID decides, whatever the age.
		{"rc_test_" + hostTag + "_" + strconv.Itoa(dead) + "_" + ts(time.Minute) + "_abc123", drop},
		{"rc_fill_demo_" + hostTag + "_" + strconv.Itoa(dead) + "_" + ts(time.Minute) + "_abc123", drop},
		{"rc_fill_demo_" + hostTag + "_" + strconv.Itoa(live) + "_" + ts(5*time.Hour) + "_abc123", keep},
		// Another host: only the age.
		{"rc_test_ffffff_" + strconv.Itoa(dead) + "_" + ts(time.Minute) + "_abc123", keep},
		{"rc_test_ffffff_" + strconv.Itoa(dead) + "_" + ts(3*time.Hour) + "_abc123", drop},
		// Names from older code: dated ones by their age, the rest when idle.
		{"rc_fill_demo_" + ts(3*time.Hour) + "_c76ffc", drop},
		{"rc_fill_demo_" + ts(time.Minute) + "_c76ffc", keep},
		{"rc_test_03a5ce939448", dropIdle},
	}
	if hostTag == "ffffff" {
		t.Skip("this host's tag collides with the other-host fixture")
	}
	for _, c := range cases {
		if got := judgeClone(c.name, now, alive); got != c.want {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
}

func TestJudgeTemplate(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	used := func(d time.Duration) string { return templateComment("0123abcd", now.Add(-d)) }
	if got := judgeTemplate(used(time.Hour), now); got != keep {
		t.Errorf("used an hour ago: %v, want keep", got)
	}
	if got := judgeTemplate(used(25*time.Hour), now); got != dropIdle {
		t.Errorf("unused for a day: %v, want dropIdle", got)
	}
	// Templates from older code carry only the fingerprint.
	if got := judgeTemplate("0123abcd", now); got != dropIdle {
		t.Errorf("no use recorded: %v, want dropIdle", got)
	}
}

func TestOwnedNames(t *testing.T) {
	name := owned("rc_fill_demo")
	if len(name) > 63 || !strings.HasPrefix(name, "rc_fill_demo_"+hostTag+"_") {
		t.Fatalf("name %q", name)
	}
	// What this binary makes is never swept while it runs.
	if got := judgeClone(name, time.Now(), processAlive); got != keep {
		t.Fatalf("own clone %q: %v, want keep", name, got)
	}
}
