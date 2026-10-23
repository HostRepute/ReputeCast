package transactional

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// The starters' words in the catalog say what the starters above say in
// English, every one of them is translated, and a starter in Vietnamese
// keeps its Liquid as it was.
func TestStarterWords(t *testing.T) {
	en, vi := i18n.Default, i18n.For(i18n.Vietnamese)
	missing := texts.Missing(i18n.Vietnamese)
	check := func(key, english string) {
		t.Helper()
		m, err := texts.Lookup(key)
		if english == "" {
			if err == nil {
				t.Errorf("%s has a message but the starter has no text", key)
			}
			return
		}
		if err != nil {
			t.Errorf("%s is missing", key)
			return
		}
		if got := m.Text(en, nil); got != english {
			t.Errorf("%s: catalog %q, starter %q", key, got, english)
		}
		if slices.Contains(missing, key) {
			t.Errorf("%s isn't translated", key)
		}
	}
	for _, st := range starterList {
		p := "starter." + st.Key + "."
		check(p+"name", st.Name)
		check(p+"description", st.Description)
		check(p+"subject", st.Subject)
		check(p+"preheader", st.Preheader)
		check(p+"heading", st.heading)
		check(p+"button", st.button)
		check(p+"after", st.after)
		check(p+"why", st.why)
		for i, b := range st.body {
			check(p+"body."+strconv.Itoa(i), b)
		}
		for i, b := range st.bodyAfter {
			check(p+"bodyAfter."+strconv.Itoa(i), b)
		}
		for i, td := range st.TestData {
			check(fmt.Sprintf("%stest.%d", p, i), td.Name)
		}
		// Every Liquid token of the English is in the translation.
		vs := st.in(vi)
		for _, pair := range [][2]string{{st.Subject, vs.Subject}, {st.heading, vs.heading}, {st.button, vs.button}, {strings.Join(st.body, " "), strings.Join(vs.body, " ")},
			{strings.Join(st.bodyAfter, " "), strings.Join(vs.bodyAfter, " ")}, {st.why, vs.why}} {
			for _, tok := range liquidTokens(pair[0]) {
				if !strings.Contains(pair[1], tok) {
					t.Errorf("%s: %q lost %s", st.Key, pair[1], tok)
				}
			}
		}
	}

	// A starter in Vietnamese: its words, its first-name fallback and its
	// sample values; English stays as it is.
	st, _ := starterByKey("password-reset")
	vs := st.in(vi)
	if vs.Name != "Đặt lại mật khẩu" || vs.Subject != "Đặt lại mật khẩu {{product}} của bạn" || vs.body[0] != "Chào {{ first_name }}," {
		t.Fatalf("starter (vi): %+v", vs)
	}
	if vs.Variables[0].Default != "bạn" || vs.Variables[2].Sample != "30 phút" || vs.TestData[0].Name != "Người dùng mới" || vs.TestData[0].Data["expires_in"] != "30 phút" {
		t.Fatalf("variables (vi): %+v %+v", vs.Variables, vs.TestData)
	}
	if st.Variables[0].Default != "there" || st.TestData[0].Data["expires_in"] != "30 minutes" {
		t.Fatal("the English starter changed")
	}
	mj := vs.mjmlFor(vi, "Acme Cloud", "")
	if !strings.Contains(mj, "Bạn nhận email này vì tài khoản Acme Cloud của bạn. Không thể tắt email tài khoản.") || !strings.Contains(mj, "Tên công ty · Số nhà và tên đường") {
		t.Fatalf("footer (vi): %s", mj)
	}
	if mj := st.mjmlFor(en, "Acme & Co", ""); !strings.Contains(mj, "You get this email because of your Acme &amp; Co account. Account emails can't be turned off.") {
		t.Fatalf("footer (en): %s", mj)
	}
	d, _ := starterByKey("product-delivery")
	if vd := d.in(vi); vd.TestData[1].Data["file"] != "3 tệp, 38 MB" || vd.Variables[3].Sample != "tệp PDF, 4,2 MB" {
		t.Fatalf("delivery (vi): %+v %+v", vd.Variables[3], vd.TestData[1])
	}
}

// A template's ID from a name in another language keeps its letters,
// without their marks.
func TestKeyFromNameFoldsMarks(t *testing.T) {
	for in, want := range map[string]string{"Mẫu chưa đặt tên": "mau-chua-dat-ten", "Password reset": "password-reset", "Đặt lại mật khẩu": "dat-lai-mat-khau"} {
		if got := KeyFromName(in); got != want {
			t.Errorf("KeyFromName(%q) = %q, want %q", in, got, want)
		}
	}
}

// liquidTokens are the {{ }} and {% %} tags of a text.
func liquidTokens(s string) []string {
	var out []string
	for {
		i := strings.IndexAny(s, "{")
		if i < 0 || i+1 >= len(s) || (s[i+1] != '{' && s[i+1] != '%') {
			if i < 0 || i+1 >= len(s) {
				return out
			}
			s = s[i+1:]
			continue
		}
		end := "}}"
		if s[i+1] == '%' {
			end = "%}"
		}
		j := strings.Index(s[i:], end)
		if j < 0 {
			return out
		}
		out = append(out, s[i:i+j+2])
		s = s[i+j+2:]
	}
}
