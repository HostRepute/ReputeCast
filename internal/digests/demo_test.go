package digests_test

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// TestDemoDigest checks the sample digest as the DigestOverview board shows
// it: on, owned by Sofia Rossi, #3 collecting two posts, #2 and #1 sent with
// their approvals, the first occurrence skipped, and the feed's badges.
func TestDemoDigest(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	sofia, aisha := env.Client(), env.Client()
	sofia.SignInDemo("sofia@acme.io")
	aisha.SignInDemo("aisha@acme.io")

	var list struct {
		Items []struct {
			ID, Name, State, Mode string
			Source                struct {
				Label      string
				Categories []string
			}
			Next *struct {
				Number, Posts int
				Status        string
				NeedsApproval bool
			}
			Last *struct {
				Number          int
				HumanOpens      *float64
				HumanOpensDelta *float64
			}
			Owner *struct{ Name string }
		}
	}
	aisha.Do("GET", ws+"/digests", nil).MustStatus(t, 200).JSON(t, &list)
	if len(list.Items) != 1 {
		t.Fatalf("digests: %+v", list.Items)
	}
	it := list.Items[0]
	if it.Name != "Engineering blog roundup" || it.State != "on" || it.Mode != "auto" || it.Source.Label != "blog.acme.io" || strings.Join(it.Source.Categories, ",") != "Engineering,Infrastructure" ||
		it.Owner == nil || it.Owner.Name != "Sofia Rossi" {
		t.Fatalf("row: %+v", it)
	}
	if it.Next == nil || it.Next.Number != 3 || it.Next.Status != "collecting" || it.Next.Posts != 2 || !it.Next.NeedsApproval {
		t.Fatalf("next: %+v", it.Next)
	}
	if it.Last == nil || it.Last.Number != 2 || it.Last.HumanOpens == nil || math.Abs(*it.Last.HumanOpens-38.9) > 0.05 || it.Last.HumanOpensDelta == nil || math.Abs(*it.Last.HumanOpensDelta+1.4) > 0.05 {
		t.Fatalf("last: %+v", it.Last)
	}

	var d struct {
		Feed struct {
			Status, Kind   string
			Items, Waiting int
			LeftOut        int
			LastReadVia    string
		}
		Next *struct {
			Number     int
			Status     string
			Posts      int
			Recipients int64
			Subject    string
			Subjects   []struct {
				Items int
				Text  string
			}
		}
		Approvals struct {
			Rules     []struct{ Key string }
			Approvers []string
		}
		Schedule struct {
			Frequency, Time, Timezone string
			Weekday                   int
		}
		SentIssues int
	}
	aisha.Do("GET", ws+"/digests/"+it.ID, nil).MustStatus(t, 200).JSON(t, &d)
	if d.Feed.Status != "ok" || d.Feed.Kind != "rss" || d.Feed.Items != 10 || d.Feed.Waiting != 2 || d.Feed.LeftOut != 1 || d.Feed.LastReadVia != "plugin" {
		t.Fatalf("feed: %+v", d.Feed)
	}
	// Recipients are counted live: Product Updates' 14,110 contacts minus
	// suppressed, role and duplicate addresses and the contacts the sunset
	// policy stopped Broadcasts to.
	if d.Next == nil || d.Next.Subject != "A year of on-call in numbers and 1 more from the Acme blog" || d.Next.Recipients < 12000 || d.Next.Recipients > 14110 {
		t.Fatalf("next issue: %+v", d.Next)
	}
	if len(d.Next.Subjects) != 3 || d.Next.Subjects[0].Text != "A year of on-call in numbers from the Acme blog" ||
		d.Next.Subjects[2].Text != "A year of on-call in numbers and 5 more from the Acme blog" {
		t.Fatalf("subjects: %+v", d.Next.Subjects)
	}
	if len(d.Approvals.Rules) != 2 || strings.Join(d.Approvals.Approvers, ",") != "Linh Nguyen,Marcus Chen" && strings.Join(d.Approvals.Approvers, ",") != "Marcus Chen,Linh Nguyen" {
		t.Fatalf("approvals: %+v", d.Approvals)
	}
	if d.Schedule.Frequency != "weekly" || d.Schedule.Time != "08:00" || d.SentIssues != 2 {
		t.Fatalf("schedule: %+v, sent %d", d.Schedule, d.SentIssues)
	}

	var issues struct {
		Items []struct {
			Number          *int
			Status          string
			Posts           int
			Recipients      *int64
			RecipientsDelta *int64
			HumanOpens      *float64
			Clicks          *float64
			SkipReason      string
			Approval        *struct {
				DecidedBy *struct{ Name string }
				DecidedAt *time.Time
			}
		}
	}
	aisha.Do("GET", ws+"/digests/"+it.ID+"/issues", nil).MustStatus(t, 200).JSON(t, &issues)
	if len(issues.Items) != 4 {
		t.Fatalf("issues: %+v", issues.Items)
	}
	i3, i2, i1, skipped := issues.Items[0], issues.Items[1], issues.Items[2], issues.Items[3]
	if i3.Status != "collecting" || i3.Posts != 2 {
		t.Fatalf("#3: %+v", i3)
	}
	if i2.Number == nil || *i2.Number != 2 || i2.Status != "sent" || i2.Posts != 3 || i2.Recipients == nil || *i2.Recipients != 13902 ||
		i2.RecipientsDelta == nil || *i2.RecipientsDelta != 31 || i2.Approval == nil || i2.Approval.DecidedBy == nil || i2.Approval.DecidedBy.Name != "Marcus Chen" {
		t.Fatalf("#2: %+v", i2)
	}
	if i1.Number == nil || *i1.Number != 1 || i1.Posts != 2 || i1.Recipients == nil || *i1.Recipients != 13871 || i1.HumanOpens == nil || math.Abs(*i1.HumanOpens-40.3) > 0.05 ||
		i1.Approval == nil || i1.Approval.DecidedBy == nil || i1.Approval.DecidedBy.Name != "Linh Nguyen" {
		t.Fatalf("#1: %+v", i1)
	}
	if skipped.Number != nil || skipped.Status != "skipped" || skipped.SkipReason != "no_items" {
		t.Fatalf("skipped: %+v", skipped)
	}

	var preview struct {
		Items []struct {
			Title, Status, LeftOutBy string
			SentIn                   int
			Image                    string
		}
		Total        int
		OlderSent    []struct{ Number, Count int }
		OlderLeftOut int
		OlderBefore  int
	}
	aisha.Do("GET", ws+"/digests/"+it.ID+"/feed?limit=6", nil).MustStatus(t, 200).JSON(t, &preview)
	got := []string{}
	for _, p := range preview.Items {
		got = append(got, p.Status)
	}
	if strings.Join(got, ",") != "left_out,new,new,sent,sent,sent" || preview.Items[0].LeftOutBy != "category" || preview.Items[3].SentIn != 2 || preview.Total != 10 ||
		len(preview.OlderSent) != 1 || preview.OlderSent[0].Number != 1 || preview.OlderSent[0].Count != 2 || preview.OlderLeftOut != 1 || preview.OlderBefore != 1 {
		t.Fatalf("feed preview: %+v", preview)
	}

	var history struct {
		Items []struct {
			Kind, Title, Via, Actor, BuiltBy, BuiltMode, Mode, ApprovedBy string
			Number                                                        int
		}
	}
	aisha.Do("GET", ws+"/digests/"+it.ID+"/history", nil).MustStatus(t, 200).JSON(t, &history)
	kinds := []string{}
	for _, h := range history.Items {
		kinds = append(kinds, h.Kind)
	}
	if strings.Join(kinds, ",") != "left_out,goes_in,goes_in,sent,turned_on" {
		t.Fatalf("history: %+v", history.Items)
	}
	if h := history.Items[3]; h.Number != 2 || h.ApprovedBy != "Marcus Chen" {
		t.Fatalf("history, sent: %+v", h)
	}
	if h := history.Items[4]; h.Actor != "Sofia Rossi" || h.Mode != "auto" || h.BuiltBy != "Daniel Park" || h.BuiltMode != "draft" {
		t.Fatalf("history, turned on: %+v", h)
	}

	// The issues are campaigns: sent ones under Sent with their digest; the
	// template stays out of the list.
	var campaigns struct {
		Items []struct {
			Name   string
			Status string
			Digest *struct{ Name string }
		}
	}
	sofia.Do("GET", ws+"/campaigns?pageSize=100&q=Engineering%20blog", nil).MustStatus(t, 200).JSON(t, &campaigns)
	if len(campaigns.Items) != 2 || campaigns.Items[0].Digest == nil || campaigns.Items[0].Digest.Name != "Engineering blog roundup" {
		t.Fatalf("campaigns: %+v", campaigns.Items)
	}

	// Reading the demo's blog again changes nothing: the overlay answers 304.
	var read struct {
		OK, NotModified bool
	}
	sofia.Do("POST", ws+"/digests/"+it.ID+"/read", nil).MustStatus(t, 200).JSON(t, &read)
	if !read.OK || !read.NotModified {
		t.Fatalf("read: %+v", read)
	}
	// Finding the blog from its home page works offline too.
	var found struct {
		OK        bool
		URL, Kind string
		Items     int
	}
	sofia.Do("POST", ws+"/digests/find", map[string]any{"address": "blog.acme.io"}).MustStatus(t, 200).JSON(t, &found)
	if !found.OK || found.URL != "https://blog.acme.io/feed/" || found.Kind != "rss" || found.Items != 10 {
		t.Fatalf("find: %+v", found)
	}
}
