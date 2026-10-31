package campaigns_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// CO-14 (ComposerWarnings): warnings are dismissed with a reason, stay
// listed with who and why, come back when they change, and can be undone.
// Blocking issues are never dismissed: the API refuses them (R4.U5).

type dismissChecks struct {
	Warnings  int
	Issues    []struct{ ID, Severity, Title string }
	IssueKeys []string
	Dismissed []struct {
		Index     int
		Dismissal struct {
			ID, Key, CheckID, Reason, ReasonText, Note, By string
		}
	}
}

func TestComposerDismissWarnings(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	flash := campaignID(t, env, "Flash sale: 48 hours only")
	var d detail
	c.Do("GET", ws+"/campaigns/"+flash, nil).MustStatus(t, 200).JSON(t, &d)
	body := map[string]any{"subject": d.Subject, "preheader": d.Preheader, "mjml": d.MJML, "html": d.HTML}
	check := func() dismissChecks {
		var out dismissChecks
		c.Do("POST", ws+"/campaigns/"+flash+"/checks", body).MustStatus(t, 200).JSON(t, &out)
		if len(out.IssueKeys) != len(out.Issues) {
			t.Fatalf("keys: %+v", out)
		}
		return out
	}
	find := func(r dismissChecks, id string) int {
		for i, is := range r.Issues {
			if is.ID == id {
				return i
			}
		}
		t.Fatalf("no %s in %+v", id, r.Issues)
		return -1
	}
	r := check()
	if len(r.Dismissed) != 0 {
		t.Fatalf("nothing dismissed yet: %+v", r.Dismissed)
	}
	alt := find(r, "content.alt_text")
	var dis struct{ ID, ReasonText, By string }
	c.Do("POST", ws+"/campaigns/"+flash+"/dismissals", map[string]any{"key": r.IssueKeys[alt], "checkId": "content.alt_text", "title": r.Issues[alt].Title, "reason": "decorative"}).
		MustStatus(t, 201).JSON(t, &dis)
	if dis.ReasonText != "the image is decorative" || dis.By != "Linh Nguyen" {
		t.Fatalf("dismissal: %+v", dis)
	}
	if auditCount(t, env, "campaign.check.dismiss", flash) != 1 {
		t.Fatal("dismiss not audited")
	}
	r = check()
	if len(r.Dismissed) != 1 || r.Dismissed[0].Index != find(r, "content.alt_text") || r.Dismissed[0].Dismissal.By != "Linh Nguyen" {
		t.Fatalf("after dismiss: %+v", r.Dismissed)
	}

	// A subject warning comes back when the subject changes.
	sub := find(r, "subject.spam")
	c.Do("POST", ws+"/campaigns/"+flash+"/dismissals", map[string]any{"key": r.IssueKeys[sub], "checkId": "subject.spam", "reason": "subject"}).MustStatus(t, 201)
	if r = check(); len(r.Dismissed) != 2 {
		t.Fatalf("two dismissed: %+v", r.Dismissed)
	}
	body["subject"] = "FLASH SALE!!! 72 HOURS ONLY"
	if r = check(); len(r.Dismissed) != 1 {
		t.Fatalf("the subject changed, its warning is back: %+v", r.Dismissed)
	}

	// Blocking issues can't be dismissed: the API says so and stores nothing.
	block := find(r, "links.blocklist")
	p := c.Do("POST", ws+"/campaigns/"+flash+"/dismissals", map[string]any{"key": r.IssueKeys[block], "checkId": "links.blocklist", "title": r.Issues[block].Title, "reason": "wrong"}).
		MustStatus(t, 422).Problem(t)
	if detail, _ := p["detail"].(string); !strings.Contains(detail, "Blocking issues can't be dismissed") {
		t.Fatalf("block refused: %v", p)
	}
	for _, x := range check().Dismissed {
		if x.Dismissal.CheckID == "links.blocklist" {
			t.Fatal("a block was dismissed")
		}
	}
	var stored int
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM campaign_check_dismissals WHERE campaign_id = $1 AND check_id = 'links.blocklist'`, flash).Scan(&stored); err != nil || stored != 0 {
		t.Fatalf("a refused dismissal was stored: %d %v", stored, err)
	}
	if auditCount(t, env, "campaign.check.dismiss", flash) != 2 {
		t.Fatal("a refused dismissal was audited as done")
	}

	// "Other" needs a note; reasons are a fixed list.
	alt = find(r, "content.alt_text")
	c.Do("POST", ws+"/campaigns/"+flash+"/dismissals", map[string]any{"key": r.IssueKeys[alt], "checkId": "content.alt_text", "reason": "other"}).MustStatus(t, 422)
	c.Do("POST", ws+"/campaigns/"+flash+"/dismissals", map[string]any{"key": r.IssueKeys[alt], "checkId": "content.alt_text", "reason": "because"}).MustStatus(t, 422)

	// Undo.
	c.Do("DELETE", ws+"/campaigns/"+flash+"/dismissals/"+dis.ID, nil).MustStatus(t, 204)
	if r = check(); len(r.Dismissed) != 0 {
		t.Fatalf("after undo: %+v", r.Dismissed)
	}
	if auditCount(t, env, "campaign.check.undismiss", flash) != 1 {
		t.Fatal("undo not audited")
	}
	c.Do("DELETE", ws+"/campaigns/"+flash+"/dismissals/"+dis.ID, nil).MustStatus(t, 404)

	// Only people who edit campaigns dismiss warnings; reasons read in the reader's language.
	aisha := env.Client()
	aisha.SignInDemo("aisha@acme.io")
	aisha.Do("POST", ws+"/campaigns/"+flash+"/dismissals", map[string]any{"key": r.IssueKeys[alt], "checkId": "content.alt_text", "reason": "decorative"}).MustStatus(t, 403)
	// Review & send counts a dismissed warning as confirmed: an API key
	// that edits campaigns can't dismiss one in the name of the person who
	// made it.
	var key struct{ Secret string }
	c.Do("POST", ws+"/api-keys", map[string]any{"name": "Composer bot", "permissions": []string{"campaigns:edit"}}).MustStatus(t, 201).JSON(t, &key)
	bot := env.Client()
	bot.Bearer = key.Secret
	if p := bot.Do("POST", ws+"/campaigns/"+flash+"/dismissals", map[string]any{"key": r.IssueKeys[alt], "checkId": "content.alt_text", "reason": "decorative"}).
		MustStatus(t, 403).Problem(t); p["code"] != "humans_only" {
		t.Fatalf("an API key dismissed a warning: %v", p)
	}
	if got := check(); len(got.Dismissed) != 0 {
		t.Fatalf("after the key's try: %+v", got.Dismissed)
	}
	c.Header = http.Header{"Accept-Language": []string{"vi"}}
	var vi struct{ ReasonText string }
	c.Do("POST", ws+"/campaigns/"+flash+"/dismissals", map[string]any{"key": r.IssueKeys[alt], "checkId": "content.alt_text", "reason": "decorative"}).MustStatus(t, 201).JSON(t, &vi)
	if !strings.Contains(vi.ReasonText, "trang trí") {
		t.Fatalf("vi reason: %+v", vi)
	}
}
