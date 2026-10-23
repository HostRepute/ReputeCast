package segments_test

import (
	"net/http"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// The builder's words and the segment problems come in the language the
// app asks for; API clients that don't ask get English.
func TestSegmentsInVietnamese(t *testing.T) {
	env := apptest.New(t)
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@example.com", Name: "Olive Owner"})
	w := env.CreateWorkspace("Segments", owner.ID)
	base := "/workspaces/" + w.Slug + "/segments"

	vi := env.Client()
	vi.Header = http.Header{"Accept-Language": {"vi"}}
	vi.SignIn("owner@example.com", "")
	en := env.Client()
	en.SignIn("owner@example.com", "")

	type catalog struct {
		Fields []struct {
			Key, Label string
			Ops        []struct{ Key, Label string }
			Options    []struct{ Value, Label string }
		}
	}
	find := func(c catalog, key string) (label string, ops, options map[string]string) {
		ops, options = map[string]string{}, map[string]string{}
		for _, f := range c.Fields {
			if f.Key != key {
				continue
			}
			for _, o := range f.Ops {
				ops[o.Key] = o.Label
			}
			for _, o := range f.Options {
				options[o.Value] = o.Label
			}
			return f.Label, ops, options
		}
		return "", ops, options
	}
	var cv, ce catalog
	vi.Do(http.MethodGet, base+"/catalog", nil).MustStatus(t, 200).JSON(t, &cv)
	en.Do(http.MethodGet, base+"/catalog", nil).MustStatus(t, 200).JSON(t, &ce)
	if label, ops, opts := find(cv, "tier"); label != "Nhóm tương tác" || ops["is_not"] != "không là" || opts["at_risk"] != "Có nguy cơ" {
		t.Errorf("tier in Vietnamese: %q %v %v", label, ops, opts)
	}
	if label, _, opts := find(cv, "mailbox_provider"); label != "Nhà cung cấp hộp thư" || opts["gmail"] != "Gmail" || opts["other"] != "Khác" {
		t.Errorf("mailbox provider in Vietnamese: %q %v", label, opts)
	}
	if label, ops, opts := find(ce, "tier"); label != "Tier" || ops["is_not"] != "is not" || opts["at_risk"] != "At risk" {
		t.Errorf("tier in English: %q %v %v", label, ops, opts)
	}

	// A group without conditions, then a list condition without a list.
	empty := map[string]any{"mode": "builder", "definition": map[string]any{"groups": []any{map[string]any{"match": "all", "conditions": []any{}}}}}
	if p := vi.Do(http.MethodPost, base+"/preview", empty).MustStatus(t, 422).Problem(t); p["detail"] != "Nhóm 1 chưa có điều kiện nào. Hãy thêm một điều kiện hoặc xóa nhóm." {
		t.Errorf("empty group: %v", p["detail"])
	}
	noList := map[string]any{"mode": "builder", "definition": map[string]any{"groups": []any{map[string]any{"match": "all",
		"conditions": []any{map[string]any{"field": "list", "op": "includes", "value": "nope"}}}}}}
	if p := vi.Do(http.MethodPost, base+"/preview", noList).MustStatus(t, 422).Problem(t); p["detail"] != "Hãy chọn một danh sách cho thành viên danh sách." {
		t.Errorf("list condition: %v", p["detail"])
	}
	if p := en.Do(http.MethodPost, base+"/preview", noList).MustStatus(t, 422).Problem(t); p["detail"] != "Pick a list for list membership." {
		t.Errorf("English: %v", p["detail"])
	}
	// A SQL segment that isn't a SELECT.
	sql := map[string]any{"mode": "sql", "sql": "DELETE FROM contacts"}
	if p := vi.Do(http.MethodPost, base+"/preview", sql).MustStatus(t, 422).Problem(t); p["detail"] != "Phân khúc SQL phải là một câu SELECT (hoặc WITH … SELECT) trả về id liên hệ." {
		t.Errorf("SQL: %v", p["detail"])
	}
	// Saving needs a name.
	if p := vi.Do(http.MethodPost, base, map[string]any{"mode": "builder", "definition": map[string]any{"groups": []any{}}}).MustStatus(t, 422).Problem(t); p["detail"] != "Hãy đặt tên cho phân khúc." {
		t.Errorf("name: %v", p["detail"])
	}
}
