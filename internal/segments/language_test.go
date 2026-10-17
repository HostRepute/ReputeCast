package segments

import (
	"strings"
	"testing"
)

func TestLanguageCondition(t *testing.T) {
	def := func(op, v string) SegmentDefinition {
		return SegmentDefinition{Groups: []SegmentGroup{{Match: "all", Conditions: []SegmentCondition{{Field: "language", Op: op, Value: v}}}}}
	}
	for _, tc := range []struct{ op, value, want string }{
		{"is", "es", "lower(split_part(replace(c.language, '_', '-'), '-', 1)) = 'es'"},
		{"is_not", "de", "lower(split_part(replace(c.language, '_', '-'), '-', 1)) IS DISTINCT FROM 'de'"},
		{"is", "none", "c.language = ''"},
		{"is_not", "none", "c.language <> ''"},
	} {
		d := def(tc.op, tc.value)
		if err := d.Validate(custom, lists); err != nil {
			t.Fatalf("%s %s: %v", tc.op, tc.value, err)
		}
		if got := Compile(d, wsID, custom).Display(); !strings.Contains(got, "\n  AND "+tc.want+"\n") {
			t.Errorf("%s %s:\n%s\nwant %s", tc.op, tc.value, got, tc.want)
		}
	}
	for _, bad := range []string{"", "es-MX", "klingon", "xx"} {
		if err := def("is", bad).Validate(custom, lists); err == nil {
			t.Errorf("%q isn't a language the builder offers", bad)
		}
	}
	if err := def("contains", "es").Validate(custom, lists); err == nil {
		t.Error("language takes is and is_not only")
	}
}
