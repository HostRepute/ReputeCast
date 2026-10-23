package automations_test

import (
	"context"
	"net/http"
	"testing"
)

// A member who reads Vietnamese gets the list's trigger column, the route,
// the alerts, problems and the starter templates in Vietnamese; API
// clients still get English.
func TestAutomationsInVietnamese(t *testing.T) {
	env, c := signedIn(t, "linh@acme.io")
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE automations SET draft_version = live_version WHERE name = 'Welcome series'`); err != nil {
		t.Fatal(err)
	}
	vi := env.Client()
	vi.SignInDemo("linh@acme.io")
	vi.Header = http.Header{"Accept-Language": {"vi"}}

	var l struct {
		list
		Alerts []struct {
			Kind, Title, Detail string
			Actions             []struct{ Label string }
		}
	}
	vi.Do("GET", wsPath, nil).MustStatus(t, 200).JSON(t, &l)
	if w := findItem(t, l.list, "Welcome series"); w.Trigger != "Tham gia danh sách · Weekly Digest" {
		t.Errorf("trigger %q", w.Trigger)
	}
	if l.Route.Label != "SES, dự phòng Postmark" {
		t.Errorf("route %q", l.Route.Label)
	}
	found := false
	for _, a := range l.Alerts {
		if a.Kind == "step_failing" {
			found = true
			if a.Title != "Webhook notify-slack của “Welcome series” trả về 404" || len(a.Actions) == 0 || a.Actions[0].Label != "Mở bước" {
				t.Errorf("alert %+v", a)
			}
		}
	}
	if !found {
		t.Errorf("no failing step alert in %+v", l.Alerts)
	}

	draft := findItem(t, l.list, "Re-permission · Mailchimp import")
	if p := vi.Do("POST", wsPath+"/"+draft.ID+"/pause", map[string]any{}).MustStatus(t, 409).Problem(t); p["detail"] != "Chỉ có thể tạm dừng tự động hóa đang chạy." {
		t.Errorf("problem %v", p)
	}
	if p := c.Do("POST", wsPath+"/"+draft.ID+"/pause", map[string]any{}).MustStatus(t, 409).Problem(t); p["detail"] != "Only a live automation can be paused." {
		t.Errorf("English problem %v", p)
	}

	var tpls struct{ Templates []struct{ Key, Name string } }
	vi.Do("GET", wsPath+"/templates", nil).MustStatus(t, 200).JSON(t, &tpls)
	if len(tpls.Templates) == 0 || tpls.Templates[0].Name != "Chuỗi chào mừng" {
		t.Errorf("templates %+v", tpls.Templates)
	}

	// A flow created in an English request: its first version's summary is
	// kept with its text and reads in Vietnamese for a Vietnamese reader.
	var created struct{ ID string }
	c.Do("POST", wsPath, map[string]any{"template": "welcome-series"}).MustStatus(t, 201).JSON(t, &created)
	type versions struct {
		Versions []struct {
			Version      int
			Summary      string
			FromCreation bool
		}
	}
	var en, vn versions
	c.Do("GET", wsPath+"/"+created.ID+"/versions", nil).MustStatus(t, 200).JSON(t, &en)
	vi.Do("GET", wsPath+"/"+created.ID+"/versions", nil).MustStatus(t, 200).JSON(t, &vn)
	if len(en.Versions) != 1 || en.Versions[0].Summary != "created from the Welcome series template" || !en.Versions[0].FromCreation {
		t.Errorf("English versions %+v", en.Versions)
	}
	if len(vn.Versions) != 1 || vn.Versions[0].Summary != "tạo từ mẫu Chuỗi chào mừng" || !vn.Versions[0].FromCreation {
		t.Errorf("Vietnamese versions %+v", vn.Versions)
	}

	// Versions written before their text was kept read in English.
	welcome := findItem(t, l.list, "Welcome series")
	c.Do("GET", wsPath+"/"+welcome.ID+"/versions", nil).MustStatus(t, 200).JSON(t, &en)
	vi.Do("GET", wsPath+"/"+welcome.ID+"/versions", nil).MustStatus(t, 200).JSON(t, &vn)
	if len(en.Versions) == 0 || len(en.Versions) != len(vn.Versions) {
		t.Fatalf("versions %+v / %+v", en.Versions, vn.Versions)
	}
	for i := range en.Versions {
		if en.Versions[i].Summary != vn.Versions[i].Summary {
			t.Errorf("v%d: %q in Vietnamese, %q in English", en.Versions[i].Version, vn.Versions[i].Summary, en.Versions[i].Summary)
		}
	}

	// Created in Vietnamese, a starter's flow and emails are in Vietnamese.
	var mine struct{ ID string }
	vi.Do("POST", wsPath, map[string]any{"template": "welcome-series"}).MustStatus(t, 201).JSON(t, &mine)
	var d struct{ Name, Description string }
	vi.Do("GET", wsPath+"/"+mine.ID, nil).MustStatus(t, 200).JSON(t, &d)
	if d.Name != "Chuỗi chào mừng · Weekly Digest" && d.Name != "Chuỗi chào mừng" || d.Description != "3 email trong một tuần" {
		t.Errorf("created %+v", d)
	}
}
