package demo

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// The Footer mini form's A/B test (M60 G-25, board FormBuilderABTest): A,
// the form itself, against B, a heading that names the readership, split
// 50/50 since 26 days ago. Since then A had 4,096 views and 131 signups
// (3.2%, 118 confirmed), B 4,104 views and 168 signups (4.1%, 152
// confirmed): B leads by 28% with 97% confidence. The 30 days the builder
// shows add the three days before the test (1,212 views, 39 signups, 34
// confirmed) to 9,412 views, 338 signups and 304 confirmations; spam
// blocked is counted outside the variants, as the pipeline counts it.
const formsABDays = 26

var formsABVariants = []struct {
	key                           string
	views, submissions, confirmed int64
}{
	{"A", 4096, 131, 118},
	{"B", 4104, 168, 152},
}

func formsABTest(ctx context.Context, s *Seeder, form uuid.UUID, f formsDemoForm) error {
	const days = 30
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		loc = time.UTC
	}
	today := s.Now.In(loc)
	weights := make([]float64, days)
	for i := range days {
		w := 1.0
		switch today.AddDate(0, 0, -i).Weekday() {
		case time.Tuesday:
			w = 1.6
		case time.Wednesday:
			w = 1.25
		case time.Saturday, time.Sunday:
			w = 0.7
		}
		if i == 0 {
			w *= 0.45 // today isn't over
		}
		weights[i] = w
	}
	inTest, before := weights[:formsABDays+1], weights[formsABDays+1:]
	var testViews, testSubs, testConf int64
	type row struct {
		variant                       string
		views, subs, confirmed, block int64
	}
	rows := make([][]row, days)
	for _, v := range formsABVariants {
		testViews, testSubs, testConf = testViews+v.views, testSubs+v.submissions, testConf+v.confirmed
		views, subs, conf := Split(v.views, inTest), Split(v.submissions, inTest), Split(v.confirmed, inTest)
		// A day confirms at most its own signups; the rest goes to days with room.
		extra := int64(0)
		for i := range conf {
			if conf[i] > subs[i] {
				extra += conf[i] - subs[i]
				conf[i] = subs[i]
			}
		}
		for i := len(conf) - 1; i >= 0 && extra > 0; i-- {
			take := min(subs[i]-conf[i], extra)
			conf[i] += take
			extra -= take
		}
		for i := range inTest {
			rows[i] = append(rows[i], row{variant: v.key, views: views[i], subs: subs[i], confirmed: conf[i]})
		}
	}
	views, subs, conf := Split(f.views-testViews, before), Split(f.submissions-testSubs, before), Split(f.confirmed-testConf, before)
	blocked := Split(f.blocked, weights)
	for i := range days {
		r := row{block: blocked[i]}
		if i > formsABDays {
			j := i - formsABDays - 1
			r.views, r.subs, r.confirmed = views[j], subs[j], conf[j]
		}
		rows[i] = append(rows[i], r)
	}
	for i := range days {
		d := today.AddDate(0, 0, -i)
		day := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
		for _, r := range rows[i] {
			if _, err := s.Tx.Exec(ctx, `INSERT INTO form_daily (form_id, day, variant, views, submissions, confirmed, blocked) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
				form, day, r.variant, r.views, r.subs, r.confirmed, r.block); err != nil {
				return err
			}
		}
	}
	started := today.AddDate(0, 0, -formsABDays)
	ab, _ := json.Marshal(map[string]any{"status": "running", "startedOn": started.Format(time.DateOnly), "minViews": 500, "confidence": 0.95})
	b, _ := json.Marshal(map[string]any{"heading": "Join 48,000 readers of the Weekly Digest", "description": "What shipped and what's next, every Tuesday. One email, no noise.",
		"button": "Get the Digest", "popupDelay": 8, "popupScroll": 0})
	at := s.Ago(time.Duration(formsABDays) * 24 * time.Hour)
	if _, err := s.Tx.Exec(ctx, `UPDATE forms SET ab = $2 WHERE id = $1`, form, ab); err != nil {
		return err
	}
	for _, v := range []struct {
		key     string
		content []byte
	}{{"A", []byte(`{}`)}, {"B", b}} {
		if _, err := s.Tx.Exec(ctx, `INSERT INTO form_variants (id, form_id, key, weight, content, created_at, updated_at) VALUES ($1,$2,$3,50,$4,$5,$5)`,
			ID(), form, v.key, v.content, at); err != nil {
			return err
		}
	}
	return nil
}
