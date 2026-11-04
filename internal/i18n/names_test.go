package i18n

import (
	"testing"
	"testing/fstest"
)

func TestVietnameseLanguageNamesTakeTheirPlacesCase(t *testing.T) {
	fsys := fstest.MapFS{
		"messages/en.json": {Data: []byte(`{
			"add": "Add {language}",
			"label": "{language}",
			"readers": "{language} readers get {gets}",
			"default": "{language} is the default. {other} is the fallback.",
			"status": "{status, select, outdated {{language} is out of date} other {the {language} version}}",
			"bold": "<b>{language}</b> is the default",
			"code": "{language} · {code}",
			"missing": "Missing · gets {language}",
			"list": "Live in {languages}",
			"named": "Sent to {list}",
			"menu": "Language: {language}"
		}`)},
		"messages/vi.json": {Data: []byte(`{
			"add": "Thêm {language}",
			"label": "{language}",
			"readers": "Người đọc {language} nhận {gets}",
			"default": "{language} là ngôn ngữ mặc định. {other} là ngôn ngữ dự phòng.",
			"status": "{status, select, outdated {{language} lỗi thời} other {phiên bản {language}}}",
			"bold": "<b>{language}</b> là ngôn ngữ mặc định",
			"code": "{language} · {code}",
			"missing": "Thiếu · nhận {language}",
			"list": "Đang chạy bằng {languages}",
			"named": "Đã gửi đến {list}",
			"menu": "Ngôn ngữ: {language}"
		}`)},
	}
	c, err := Load(fsys, "namestest")
	if err != nil {
		t.Fatal(err)
	}
	vi := For(Vietnamese)
	for _, tc := range []struct {
		key  string
		args Args
		want string
	}{
		{"add", Args{"language": "Tiếng Tây Ban Nha"}, "Thêm tiếng Tây Ban Nha"},
		{"label", Args{"language": "Tiếng Anh"}, "Tiếng Anh"},
		{"label", Args{"language": "tiếng Anh"}, "Tiếng Anh"},
		{"readers", Args{"language": "Tiếng Tây Ban Nha", "gets": "Tiếng Anh"}, "Người đọc tiếng Tây Ban Nha nhận tiếng Anh"},
		{"default", Args{"language": "tiếng Đức", "other": "Tiếng Anh"}, "Tiếng Đức là ngôn ngữ mặc định. Tiếng Anh là ngôn ngữ dự phòng."},
		{"status", Args{"status": "outdated", "language": "Tiếng Pháp"}, "Tiếng Pháp lỗi thời"},
		{"status", Args{"status": "draft", "language": "Tiếng Pháp"}, "phiên bản tiếng Pháp"},
		{"bold", Args{"language": "Tiếng Ý"}, "Tiếng Ý là ngôn ngữ mặc định"},
		{"code", Args{"language": "Tiếng Anh (Mỹ)", "code": "en-US"}, "Tiếng Anh (Mỹ) · en-US"},
		{"missing", Args{"language": "Tiếng Anh"}, "Thiếu · nhận tiếng Anh"},
		{"menu", Args{"language": "Tiếng Việt"}, "Ngôn ngữ: Tiếng Việt"},
		{"list", Args{"languages": "Tiếng Anh, Tiếng Pháp và Tiếng Ả Rập"}, "Đang chạy bằng tiếng Anh, tiếng Pháp và tiếng Ả Rập"},
		{"label", Args{"language": "Tiếng Anh, Tiếng Pháp hoặc Tiếng Đức"}, "Tiếng Anh, tiếng Pháp hoặc tiếng Đức"},
		// Names that aren't languages keep their case.
		{"named", Args{"list": "Tiếng Anh cho người đi làm"}, "Đã gửi đến Tiếng Anh cho người đi làm"},
		{"named", Args{"list": "Weekly Digest và Tiếng Anh"}, "Đã gửi đến Weekly Digest và Tiếng Anh"},
	} {
		if got := c.M(tc.key).Text(vi, tc.args); got != tc.want {
			t.Errorf("%s %v: got %q, want %q", tc.key, tc.args, got, tc.want)
		}
	}
	// English text is left alone, and so is HTML output's markup.
	if got := c.M("add").Text(For(English), Args{"language": "Tiếng Anh"}); got != "Add Tiếng Anh" {
		t.Errorf("English: %q", got)
	}
	if got := c.M("bold").HTML(vi, Args{"language": "Tiếng Ý"}, nil); got != "<b>Tiếng Ý</b> là ngôn ngữ mặc định" {
		t.Errorf("HTML: %q", got)
	}
}

func TestSpanishAndFrenchLanguageNamesTakeTheirPlacesCase(t *testing.T) {
	fsys := fstest.MapFS{
		"messages/en.json": {Data: []byte(`{
			"add": "Add {language}",
			"label": "{language}",
			"readers": "{language} readers get {gets}",
			"list": "Live in {languages}",
			"named": "Sent to {list}"
		}`)},
		"messages/es.json": {Data: []byte(`{
			"add": "Añadir {language}",
			"label": "{language}",
			"readers": "Lectores en {language}: reciben {gets}",
			"list": "Publicada en {languages}",
			"named": "Enviada a {list}"
		}`)},
		"messages/fr.json": {Data: []byte(`{
			"add": "Ajouter une version en {language}",
			"label": "{language}",
			"list": "En ligne en {languages}"
		}`)},
	}
	c, err := Load(fsys, "namestest_romance")
	if err != nil {
		t.Fatal(err)
	}
	es, fr := For(Spanish), For(French)
	for _, tc := range []struct {
		l    Lang
		key  string
		args Args
		want string
	}{
		{es, "add", Args{"language": "Inglés"}, "Añadir inglés"},
		{es, "label", Args{"language": "inglés"}, "Inglés"},
		{es, "readers", Args{"language": "Español", "gets": "Inglés"}, "Lectores en español: reciben inglés"},
		{es, "list", Args{"languages": AndList(Spanish, []string{"Inglés", "Español", "Italiano"})}, "Publicada en inglés, español e italiano"},
		{es, "label", Args{"language": "Inglés, Francés u Occitano"}, "Inglés, francés u occitano"},
		{es, "label", Args{"language": "Inglés (Estados Unidos)"}, "Inglés (Estados Unidos)"},
		// Names that aren't languages keep their case.
		{es, "named", Args{"list": "Inglés para todos"}, "Enviada a Inglés para todos"},
		{es, "named", Args{"list": "Weekly Digest y Español"}, "Enviada a Weekly Digest y Español"},
		{fr, "add", Args{"language": "Anglais"}, "Ajouter une version en anglais"},
		{fr, "label", Args{"language": "anglais"}, "Anglais"},
		{fr, "list", Args{"languages": "Anglais, Espagnol et Allemand"}, "En ligne en anglais, espagnol et allemand"},
	} {
		if got := c.M(tc.key).Text(tc.l, tc.args); got != tc.want {
			t.Errorf("%s %s %v: got %q, want %q", tc.l.Locale, tc.key, tc.args, got, tc.want)
		}
	}
	// German names are capitalised nouns everywhere.
	if got := LanguageName(German, "es"); got != "Spanisch" {
		t.Errorf("de name: %q", got)
	}
	if got := LanguageName(Spanish, "en"); got != "Inglés" {
		t.Errorf("es name: %q", got)
	}
	if got := LanguageName(French, "pt-BR"); got != "Portugais" {
		t.Errorf("fr name: %q", got)
	}
	if got := LanguageName(English, "es"); got != "" {
		t.Errorf("en name: %q", got)
	}
}

func TestSpanishListsJoinBeforeTheNextSound(t *testing.T) {
	for _, tc := range []struct {
		items []string
		or    bool
		want  string
	}{
		{[]string{"español", "inglés"}, false, "español e inglés"},
		{[]string{"agua", "hielo"}, false, "agua y hielo"},
		{[]string{"francés", "hindi"}, false, "francés e hindi"},
		{[]string{"siete", "ocho"}, true, "siete u ocho"},
		{[]string{"inglés", "holandés"}, true, "inglés u holandés"},
		{[]string{"inglés", "alemán"}, true, "inglés o alemán"},
	} {
		got := AndList(Spanish, tc.items)
		if tc.or {
			got = OrList(Spanish, tc.items)
		}
		if got != tc.want {
			t.Errorf("%v: got %q, want %q", tc.items, got, tc.want)
		}
	}
	if got := AndList(German, []string{"A", "B", "C"}); got != "A, B und C" {
		t.Errorf("de: %q", got)
	}
	if got := OrList(French, []string{"A", "B"}); got != "A ou B" {
		t.Errorf("fr: %q", got)
	}
}

func TestQuoteUsesTheLanguagesMarks(t *testing.T) {
	for l, want := range map[Locale]string{
		English:    "“Weekly Digest”",
		Spanish:    "“Weekly Digest”",
		Vietnamese: "“Weekly Digest”",
		German:     "„Weekly Digest“",
		French:     "« Weekly Digest »",
	} {
		if got := Quote(l, "Weekly Digest"); got != want {
			t.Errorf("%s: %q, want %q", l, got, want)
		}
	}
}
