package languages_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// Content languages read in the member's language: language names,
// problems, the compare view's blocks and the version history, whose
// labels were stored by an English request. API clients get English.
func TestLanguagesInVietnamese(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	en, vi := env.Client(), env.Client()
	en.SignInDemo("linh@acme.io")
	vi.SignInDemo("linh@acme.io")
	vi.Header = http.Header{"Accept-Language": {"vi"}}

	name := func(w workspaceLanguages, code string) string {
		for _, l := range w.Languages {
			if l.Code == code {
				return l.Name
			}
		}
		return ""
	}
	var list workspaceLanguages
	vi.Do("GET", ws+"/languages", nil).MustStatus(t, 200).JSON(t, &list)
	if name(list, "es") != "Tiếng Tây Ban Nha" || name(list, "de") != "Tiếng Đức" {
		t.Fatalf("names in Vietnamese: %+v", list.Languages)
	}
	en.Do("GET", ws+"/languages", nil).MustStatus(t, 200).JSON(t, &list)
	if name(list, "es") != "Spanish" {
		t.Fatalf("names in English: %+v", list.Languages)
	}
	p := vi.Do("POST", ws+"/languages", map[string]any{"code": "ar"}).MustStatus(t, 422).Problem(t)
	if p["detail"] != "Chưa hỗ trợ ngôn ngữ viết từ phải sang trái. Liên hệ dùng tiếng Ả Rập nhận tiếng Anh." {
		t.Fatalf("right to left, in Vietnamese: %v", p)
	}

	var created struct{ ID string }
	en.Do("POST", ws+"/campaigns", map[string]any{"name": "Languages in Vietnamese"}).MustStatus(t, 201).JSON(t, &created)
	camp := ws + "/campaigns/" + created.ID
	base := camp + "/languages"
	en.Do("PUT", camp+"/content", map[string]any{"baseVersion": 1, "subject": "Scheduled rollouts", "mjml": draft("Hi, you can now schedule flag changes.", "")}).MustStatus(t, 200)
	en.Do("POST", base, map[string]any{"language": "es"}).MustStatus(t, 201)
	p = vi.Do("POST", base, map[string]any{"language": "es"}).MustStatus(t, 409).Problem(t)
	if p["detail"] != "Email đã có phiên bản tiếng Tây Ban Nha." {
		t.Fatalf("already has it, in Vietnamese: %v", p)
	}

	// The version history: the label an English request stored reads in
	// Vietnamese, and a label stored before reads English.
	type history struct{ Items []struct{ Label string } }
	var h history
	vi.Do("GET", base+"/es/versions", nil).MustStatus(t, 200).JSON(t, &h)
	if len(h.Items) != 1 || h.Items[0].Label != "chép từ tiếng Anh" {
		t.Fatalf("history in Vietnamese: %+v", h)
	}
	en.Do("GET", base+"/es/versions", nil).MustStatus(t, 200).JSON(t, &h)
	if h.Items[0].Label != "copied from English" {
		t.Fatalf("history in English: %+v", h)
	}
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE content_translation_versions v SET label_text = NULL FROM content_translations t
		WHERE t.id = v.translation_id AND t.owner_id = $1`, created.ID); err != nil {
		t.Fatal(err)
	}
	vi.Do("GET", base+"/es/versions", nil).MustStatus(t, 200).JSON(t, &h)
	if h.Items[0].Label != "copied from English" {
		t.Fatalf("an older label: %+v", h)
	}

	// The compare view names the blocks.
	type compare struct {
		Rows []struct {
			Source *struct{ Name string }
		}
	}
	var c compare
	vi.Do("GET", base+"/es/compare", nil).MustStatus(t, 200).JSON(t, &c)
	if len(c.Rows) == 0 || c.Rows[0].Source == nil || c.Rows[0].Source.Name != "Dòng tít · Đề mục 1" {
		t.Fatalf("compare in Vietnamese: %+v", c.Rows)
	}
	en.Do("GET", base+"/es/compare", nil).MustStatus(t, 200).JSON(t, &c)
	if c.Rows[0].Source.Name != "Headline · Heading 1" {
		t.Fatalf("compare in English: %+v", c.Rows)
	}
}
