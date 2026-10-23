package segments

import (
	"errors"
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// Every field, operator and fixed value of the builder has its words in
// messages/en.json; only names (Gmail, Chrome) don't.
func TestCatalogWordsHaveMessages(t *testing.T) {
	ops := map[string]bool{}
	for key, spec := range builtinFields {
		if _, err := msgs.Lookup("field." + key); err != nil {
			t.Errorf("field %s has no message", key)
		}
		for _, op := range spec.Ops {
			ops[op] = true
		}
	}
	for _, typ := range []string{"text", "number", "date", "boolean", "select"} {
		for _, op := range customOps(typ) {
			ops[op] = true
		}
	}
	for op := range ops {
		if _, err := msgs.Lookup("op." + op); err != nil {
			t.Errorf("operator %s has no message", op)
		}
	}
	for field, values := range enumValues {
		for _, v := range values {
			_, err := msgs.Lookup("option." + field + "." + v[0])
			switch {
			case err != nil && v[1] == "":
				t.Errorf("option %s.%s has no message", field, v[0])
			case err == nil && v[1] != "":
				t.Errorf("option %s.%s is a name (%s) and has a message too", field, v[0], v[1])
			}
		}
	}
	// The English the builder showed before stays.
	en := i18n.Default
	for key, want := range map[string]string{"mailbox_provider": "Mailbox provider", "list": "List membership", "paid_ltv": "Paid subscription · lifetime value", "custom:plan": "Custom field · plan"} {
		if got := fieldLabel(en, key); got != want {
			t.Errorf("field %s: %q, want %q", key, got, want)
		}
	}
	if got := opLabel(en, "not_in_last"); got != "not in the last" {
		t.Errorf("operator: %q", got)
	}
	if got := optionLabel(en, "consent", "pending", ""); got != "Imported, pending" {
		t.Errorf("option: %q", got)
	}
	if got := optionLabel(i18n.For(i18n.Vietnamese), "mailbox_provider", "gmail", "Gmail"); got != "Gmail" {
		t.Errorf("a name: %q", got)
	}
	for _, missing := range msgs.Missing(i18n.Vietnamese) {
		t.Errorf("%s has no Vietnamese", missing)
	}
	// The problem for a database without SQL segments says what the error does.
	if got := mSQLDisabled.Text(i18n.Default, nil); got != ErrSQLDisabled.Error() {
		t.Errorf("SQL disabled: %q, want %q", got, ErrSQLDisabled.Error())
	}
}

func TestSourceLabelsInVietnamese(t *testing.T) {
	vi := i18n.For(i18n.Vietnamese)
	for src, want := range map[string][2]string{
		"import:brevo":            {"Brevo import", "Nhập từ Brevo"},
		"import:csv":              {"CSV import", "Nhập từ CSV"},
		"import:":                 {"File import", "Nhập từ tệp"},
		"form:blog":               {"Signup form · blog", "Biểu mẫu đăng ký · blog"},
		"form:popup":              {"Popup form", "Biểu mẫu bật lên"},
		"form":                    {"Signup form", "Biểu mẫu đăng ký"},
		"api":                     {"API", "API"},
		"manual":                  {"Added by the team", "Do nhóm thêm vào"},
		"store:woocommerce:abc":   {"WooCommerce", "WooCommerce"},
		"":                        {"Unknown", "Không rõ"},
		"something_else:entirely": {"something_else:entirely", "something_else:entirely"},
	} {
		if got := SourceLabel(src); got != want[0] {
			t.Errorf("%q in English: %q, want %q", src, got, want[0])
		}
		if got := SourceLabelIn(vi, src); got != want[1] {
			t.Errorf("%q in Vietnamese: %q, want %q", src, got, want[1])
		}
	}
}

// A condition's problem names its field in the reader's language.
func TestConditionProblemsInVietnamese(t *testing.T) {
	d := SegmentDefinition{Groups: []SegmentGroup{{Match: "all", Conditions: []SegmentCondition{{Field: "list", Op: "includes", Value: "not-a-list"}}}}}
	err := d.Validate(map[string]CustomField{}, nil)
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("%v", err)
	}
	if ve.Message != "Pick a list for list membership." || ve.Text(i18n.Default) != ve.Message {
		t.Errorf("English: %q / %q", ve.Message, ve.Text(i18n.Default))
	}
	if got := ve.Text(i18n.For(i18n.Vietnamese)); got != "Hãy chọn một danh sách cho thành viên danh sách." {
		t.Errorf("Vietnamese: %q", got)
	}
	var se *SQLError
	if _, err := CheckSQL("DELETE FROM contacts"); !errors.As(err, &se) {
		t.Fatalf("DELETE: %v", err)
	}
	if se.Message != "A SQL segment must be a SELECT (or WITH … SELECT) that returns contact ids." ||
		se.Text(i18n.For(i18n.Vietnamese)) != "Phân khúc SQL phải là một câu SELECT (hoặc WITH … SELECT) trả về id liên hệ." {
		t.Errorf("SQL: %q / %q", se.Message, se.Text(i18n.For(i18n.Vietnamese)))
	}
	views := (&viewsProblem{detail: "permission denied for table users"})
	if got := views.Error(); got != "Your SQL can only read the "+joinEnglish(ViewNames)+" views (permission denied for table users)." {
		t.Errorf("views: %q", got)
	}
}

func joinEnglish(xs []string) string {
	out := ""
	for i, x := range xs {
		switch {
		case i == 0:
		case i == len(xs)-1:
			out += " and "
		default:
			out += ", "
		}
		out += x
	}
	return out
}
