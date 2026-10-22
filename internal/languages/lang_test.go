package languages

import (
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
)

var vi = i18n.For(i18n.Vietnamese)

// Blocks are named in the reader's language; their English names, which
// approvals store and compare, stay as they were.
func TestBlockNamesInVietnamese(t *testing.T) {
	bs := Blocks(doc(header, hero, headline, intro, button, footer), "")
	en, vn := []string{}, []string{}
	for _, b := range bs {
		en = append(en, b.Ref().Name(i18n.Default))
		vn = append(vn, b.nameIn(vi))
		if b.nameIn(i18n.Default) != b.Name || b.Ref().Name(i18n.Default) != b.Name {
			t.Errorf("English name %q, from its ref %q", b.Name, b.Ref().Name(i18n.Default))
		}
	}
	want := "Đầu thư · logo và liên kết|Hình ảnh hero|Dòng tít · Đề mục 1|Phần mở đầu|Nút|Chân thư"
	if strings.Join(vn, "|") != want {
		t.Errorf("names: %q", vn)
	}
	cards := Blocks(doc(`<mj-section><mj-column><mj-image src="a.png"></mj-image><mj-text>One</mj-text></mj-column><mj-column><mj-image src="b.png"></mj-image><mj-text>Two</mj-text></mj-column></mj-section>`), "")[0]
	if cards.Name != "Two cards" || cards.nameIn(vi) != "Hai thẻ" || cards.Ref().Short(i18n.Default) != "two cards" || cards.Ref().Short(vi) != "hai thẻ" {
		t.Errorf("cards: %q %q", cards.Name, cards.nameIn(vi))
	}
	if got := summaryIn(vi, bs[0].Text); got != `Logo, văn bản thay thế "Acme" · Blog · Docs` {
		t.Errorf("header text: %q", got)
	}
	if got := summaryIn(vi, bs[1].Text); got != "scheduled-rollouts.png · không có văn bản thay thế" {
		t.Errorf("hero text: %q", got)
	}
	if got := summaryIn(i18n.Default, bs[1].Text); got != bs[1].Text {
		t.Errorf("English text changed: %q", got)
	}
}

// What changed after an approval reads in the reader's language; English
// reads as it always has.
func TestDiffVersionsInVietnamese(t *testing.T) {
	snap := func(lang, status string, version int, subject, mjml string) VersionSnapshot {
		v := VersionSnapshot{Language: lang, Status: status, Version: version, Hash: mjml + subject, Subject: subject}
		for _, b := range Blocks(mjml, "") {
			ref := b.ref
			v.Blocks = append(v.Blocks, BlockSnapshot{Name: b.Name, Hash: b.Hash, Raw: b.Raw, Ref: &ref})
		}
		return v
	}
	before := []VersionSnapshot{
		snap("de", storedTranslated, 3, "Rollouts", doc(headDE, introDE, buttonDE, footerDE)),
		snap("es", storedDraft, 2, "Hola", doc(headDE)),
	}
	after := []VersionSnapshot{
		snap("de", storedTranslated, 4, "Rollouts", doc(headDE, strings.Replace(introDE, "jetzt", "ab sofort", 1), buttonDE, footerDE)),
		snap("vi", storedDraft, 1, "Xin chào", doc(headDE)),
	}
	en := DiffVersions(before, after)
	if len(en) != 3 || en[0].Name != "German" || en[0].After != "v4 · intro changed" || en[1].What != "version added" || en[2].What != "version removed" {
		t.Fatalf("English: %+v", en)
	}
	vn := DiffVersionsIn(vi, before, after)
	if vn[0].Name != "Tiếng Đức" || vn[0].Before != "v3" || vn[0].After != "v4 · phần mở đầu đã thay đổi" || vn[0].What != "phần mở đầu đã thay đổi" {
		t.Errorf("edited: %+v", vn[0])
	}
	if vn[1].Name != "Tiếng Việt" || vn[1].Before != "không có" || vn[1].After != "v1 · mới thêm" || vn[1].What != "đã thêm phiên bản" {
		t.Errorf("added: %+v", vn[1])
	}
	if vn[2].Name != "Tiếng Tây Ban Nha" || vn[2].After != "đã xóa" || vn[2].What != "đã xóa phiên bản" {
		t.Errorf("removed: %+v", vn[2])
	}
	// Snapshots taken before blocks kept their refs name them in English.
	now := after[0]
	now.Blocks = append([]BlockSnapshot{}, now.Blocks...)
	for i := range now.Blocks {
		now.Blocks[i].Ref = nil
	}
	if got := DiffVersionsIn(vi, before[:1], []VersionSnapshot{now}); got[0].What != "intro đã thay đổi" {
		t.Errorf("without refs: %+v", got)
	}
}

func TestNameIn(t *testing.T) {
	cases := map[string][2]string{"es": {"Spanish", "Tiếng Tây Ban Nha"}, "pt-BR": {"Portuguese", "Tiếng Bồ Đào Nha"}, "xx": {"xx", "xx"}, "vi": {"Vietnamese", "Tiếng Việt"}}
	for code, want := range cases {
		if got := NameIn(i18n.Default, code); got != want[0] {
			t.Errorf("%s in English: %q", code, got)
		}
		if got := NameIn(vi, code); got != want[1] {
			t.Errorf("%s in Vietnamese: %q", code, got)
		}
	}
	if plainKey("Tiếng Ả Rập") != "tieng a rap" || plainKey("Đức") != "duc" {
		t.Errorf("plain keys: %q %q", plainKey("Tiếng Ả Rập"), plainKey("Đức"))
	}
}

// A saved version's label keeps its message: its language is stored as a
// code and named in each reader's language; rows without one read English.
func TestKeptLabels(t *testing.T) {
	lab := newLabel(mLabelAppliedFrom, "en", i18n.Args{"kind": "link"})
	if lab.English != "link applied from English" {
		t.Fatalf("English: %q", lab.English)
	}
	if got := labelIn(vi, lab.English, lab.raw()); got != "liên kết áp dụng từ Tiếng Anh" {
		t.Errorf("Vietnamese: %q", got)
	}
	if got := labelIn(i18n.Default, lab.English, lab.raw()); got != lab.English {
		t.Errorf("English reader: %q", got)
	}
	if got := labelIn(vi, "copied from English", nil); got != "copied from English" {
		t.Errorf("older row: %q", got)
	}
	if got := labelIn(vi, "restored v3", []byte(`{"id":"languages.label.gone"}`)); got != "restored v3" {
		t.Errorf("unknown message: %q", got)
	}
}
