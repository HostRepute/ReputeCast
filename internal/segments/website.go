package segments

import (
	"regexp"
	"strconv"
	"strings"
)

// Members of the publication website (design expansion phase 7): "Signed
// in to the website" since a date or a number of days ago, not since then,
// ever or never. The contacts view reads it from website_members.

func init() {
	opLabels["since"] = "since"
	opLabels["not_since"] = "not since"
	opLabels["ever"] = "ever"
	builtinFields["signed_in"] = fieldSpec{"Signed in to the website", KindDate, []string{"since", "not_since", "ever", "never"}}
	builtinOrder = insertAfter(builtinOrder, "joined", "signed_in")
	extraValidate["signed_in"] = validateSignedIn
	extraCompile["signed_in"] = compileSignedIn
}

// websiteColumns and websiteJoins add when a contact last signed in to the
// website to the contacts view.
const websiteColumns = `,
           wm.last_signed_in AS signed_in_at`

const websiteJoins = `
    LEFT JOIN public.website_members wm ON wm.contact_id = c.id`

// daysAgoRe is a relative day ("7d": seven days ago).
var daysAgoRe = regexp.MustCompile(`^(\d{1,4})d$`)

func validateSignedIn(c SegmentCondition) error {
	if c.Op == "ever" || c.Op == "never" {
		return nil
	}
	v := strings.TrimSpace(c.Value)
	if m := daysAgoRe.FindStringSubmatch(v); m != nil {
		if n, _ := strconv.Atoi(m[1]); n >= 1 && n <= 3650 {
			return nil
		}
		return problemf("Pick between 1 and 3650 days ago.")
	}
	if !dateRe.MatchString(v) {
		return problemf("Pick a date like 2026-09-29, or a number of days ago.")
	}
	return nil
}

// signedInSince is the start of "since": a date, or now minus N days.
func signedInSince(t *sqlText, v string) {
	v = strings.TrimSpace(v)
	if m := daysAgoRe.FindStringSubmatch(v); m != nil {
		n, _ := strconv.Atoi(m[1])
		t.raw("now() - interval '" + strconv.Itoa(n) + " days'")
		return
	}
	t.bind(str(v))
	t.raw("::date")
}

func compileSignedIn(t *sqlText, c SegmentCondition) {
	switch c.Op {
	case "ever":
		t.raw("c.signed_in_at IS NOT NULL")
	case "never":
		t.raw("c.signed_in_at IS NULL")
	case "not_since":
		t.raw("(c.signed_in_at IS NULL OR c.signed_in_at < ")
		signedInSince(t, c.Value)
		t.raw(")")
	default:
		t.raw("c.signed_in_at >= ")
		signedInSince(t, c.Value)
	}
}
