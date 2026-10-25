package engine_test

import (
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/sending/sendtest"
)

// TestThroughput50k measures a 50,000-recipient send through the demo
// transport (REPUTECAST_PERF=1): the audience build, then the send loop with
// the provider's rate out of the way, first without delivery and engagement
// events (the send path alone), then with them. It reports throughput and
// the Go heap.
func TestThroughput50k(t *testing.T) {
	if os.Getenv("REPUTECAST_PERF") == "" {
		t.Skip("set REPUTECAST_PERF=1 to measure a 50,000-recipient send")
	}
	s := sendtest.NewSuite(t)
	for _, quiet := range []bool{true, false} {
		name := map[bool]string{true: "send path", false: "with demo events"}[quiet]
		t.Run(name, func(t *testing.T) {
			f := s.Fixture(t, "Throughput "+name)
			f.Exec(`UPDATE campaigns SET ab_test = NULL, route_override = $2 WHERE id = $1`, f.Campaign, fmt.Sprintf(`{"provider_id":%q}`, f.ProviderID("ses")))
			// 5,000 /s: the provider's rate isn't what is measured.
			f.Exec(`UPDATE providers SET health = jsonb_set(coalesce(health, '{}'::jsonb), '{rate}', '{"max": 5000}') WHERE id = $1`, f.ProviderID("ses"))
			f.AddContacts(sendtest.ContactSpec{N: perfN(), Mailbox: "gmail", Domain: "gmail.com", Engaged: true})
			f.E.QuietDemo = quiet
			defer func() { f.E.QuietDemo = false }()

			var before runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&before)
			// The start runs Send Readiness again (the send-time recheck);
			// the audience build follows.
			t0 := time.Now()
			if err := f.E.StartWith(f.Ctx, f.Campaign, engine.StartOptions{By: f.Sofia, Now: true}); err != nil {
				t.Fatalf("start: %v", err)
			}
			started := time.Since(t0)
			t0 = time.Now()
			if err := f.E.DrivePrepare(f.Ctx, f.Campaign, f.Start); err != nil {
				t.Fatalf("prepare: %v", err)
			}
			prepared := time.Since(t0)
			if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1`, f.Campaign); n != int64(perfN()) {
				t.Fatalf("%d messages queued, want 50,000", n)
			}
			peak := uint64(0)
			stop := make(chan struct{})
			go func() {
				for {
					select {
					case <-stop:
						return
					case <-time.After(50 * time.Millisecond):
						var m runtime.MemStats
						runtime.ReadMemStats(&m)
						peak = max(peak, m.HeapInuse)
					}
				}
			}()
			t1 := time.Now()
			_, done, err := f.E.Drive(f.Ctx, f.Campaign, f.Start, 2*time.Minute)
			close(stop)
			if err != nil {
				t.Fatal(err)
			}
			sending := time.Since(t1)
			sent := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND sent_at IS NOT NULL`, f.Campaign)
			if sent != int64(perfN()) || !done {
				f.Dump()
				t.Fatalf("sent %d of 50,000 (done %v)", sent, done)
			}
			events := f.Count(`SELECT count(*) FROM delivery_events WHERE campaign_id = $1`, f.Campaign)
			t.Logf("%s: start with the Send Readiness recheck %v; audience build %v (%.0f recipients/s); send %v, %.0f messages/s; %d events applied; heap in use %.0f MB before, %.0f MB peak",
				name, started.Round(time.Millisecond), prepared.Round(time.Millisecond), float64(perfN())/prepared.Seconds(), sending.Round(time.Millisecond), float64(sent)/sending.Seconds(), events,
				float64(before.HeapInuse)/1e6, float64(peak)/1e6)
		})
	}
}

func perfN() int {
	if os.Getenv("REPUTECAST_PERF") == "small" {
		return 20000
	}
	return 50000
}
